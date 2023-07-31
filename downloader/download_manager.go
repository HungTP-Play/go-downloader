package downloader

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"sync"
)

// Internally use in Downloader
//
// The man who controls the download process
type downloadManager struct {
	ctx context.Context
	cfg *DownloaderConfig

	url      string
	filename string

	writer WriteAtWriter
	wg     sync.WaitGroup
	mu     sync.Mutex

	totalBytes int64
	written    int64
	err        error
}

// ---------------------------- Getter & Setter ----------------------------
func (dm *downloadManager) GetTotalBytes() int64 {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	return dm.totalBytes
}

func (dm *downloadManager) SetTotalBytes(totalBytes int64) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.totalBytes = totalBytes
}

func (dm *downloadManager) GetWritten() int64 {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	return dm.written
}

func (dm *downloadManager) SetWritten(written int64) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.written = written
}

func (dm *downloadManager) GetError() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	return dm.err
}

func (dm *downloadManager) SetError(err error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.err = err
}

func (dm *downloadManager) AddWritten(written int64) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.written += written
}

// ---------------------------- Calculate Before Download ----------------------------

func (dm *downloadManager) doHeadRequest(url string) (int64, error) {
	resp, err := http.Head(url)
	// Is Response != 200 return error
	if resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusForbidden {
		return 0, &DownloadError{Message: "Response status code is not 200", Err: http.ErrNotSupported}
	}

	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.ContentLength, nil
}

// Use when some source not support HEAD request
//
// Get file size by calling GET request with Range: bytes=0-0
//
// And return the size of the file in the Content-Range header
func (dm *downloadManager) doGetZero(url string) (int64, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, &DownloadError{Message: "Failed to create request", Err: err}
	}

	req.Header.Set("Range", "bytes=0-0")
	resp, err := http.DefaultClient.Do(req)
	// Is Response != 206 return error
	if resp.StatusCode != http.StatusPartialContent {
		return 0, &DownloadError{Message: "Response status code is not 206", Err: err}
	}

	if err != nil {
		return 0, &DownloadError{Message: "Failed to do request", Err: err}
	}

	contentRange := resp.Header.Get("Content-Range")
	fileSize := contentRange[len("bytes 0-0/"):]
	size, err := strconv.ParseInt(fileSize, 10, 64)
	if err != nil {
		return 0, &DownloadError{Message: "Failed to parse file size", Err: err}
	}
	return size, nil
}

// Get file size by calling HEAD request
func (dm *downloadManager) getFileSize(url string) (int64, error) {
	size, err := dm.doHeadRequest(url)
	if err, ok := err.(*DownloadError); ok {
		if err.Err != http.ErrNotSupported {
			size, getErr := dm.doGetZero(url)
			if getErr != nil {
				return 0, &DownloadError{Message: "Failed to get file size", Err: err}
			}

			return size, nil
		}
	}

	if err != nil {
		// If 416, the server does not support range requests
		if err == http.ErrNotSupported {
			return 0, &DownloadError{Message: "The server does not support range requests", Err: err}
		}

		// If 404, the file does not exist
		if err == http.ErrMissingFile {
			return 0, &DownloadError{Message: "The file does not exist", Err: err}
		}

		return 0, &DownloadError{Message: "Failed to get file size", Err: err}
	}
	return size, nil
}

// Calculate the number of chunks will be split.
//
// The parameter is the total size of the file.
func (dm *downloadManager) determineNumParts(totalSize int64) (numParts int64, chunkSize int64) {
	if dm.cfg.ChunkSize > 0 {
		numParts = totalSize / dm.cfg.ChunkSize
		return numParts, dm.cfg.ChunkSize
	}

	if dm.cfg.PartDeterminerFunc != nil {
		numParts = dm.cfg.PartDeterminerFunc(totalSize)
		return numParts, totalSize / numParts
	}

	if dm.cfg.ChunkSizeDeterminerFunc != nil {
		chunkSize = dm.cfg.ChunkSizeDeterminerFunc(totalSize)
		numParts = totalSize / chunkSize
		return numParts, chunkSize
	}

	numParts = defaultPartDeterminer(totalSize)
	return numParts, totalSize / numParts
}

type Batch []DownloadChunk

// Batch the chunks
//
// If `MaxConcurrentDownloads` is set to -1, all the chunks will be downloaded concurrently.
// The number of batch is the number of chunks. Each batch contains only one chunk.
//
// If `MaxConcurrentDownloads` greater than 0, the chunks will be downloaded concurrently by batch.
// The number of batch is the number of chunks divided by `MaxConcurrentDownloads`.
// Each batch contains (number of chunks / number of batch) chunks.
func (dm *downloadManager) batchChunks(numChunks int64, writer WriteAtWriter) []Batch {
	if dm.cfg.MaxConcurrentDownloads == -1 {
		return dm.batchChunksByOne(numChunks, writer)
	}
	return dm.batchChunksByMaxConcurrentParts(numChunks, writer)
}

func (dm *downloadManager) batchChunksByOne(numChunks int64, writer WriteAtWriter) []Batch {
	batches := make([]Batch, numChunks)
	for i := int64(0); i < numChunks; i++ {
		start := i * dm.cfg.ChunkSize
		size := dm.cfg.ChunkSize
		if start+size > dm.totalBytes {
			size = dm.totalBytes - start
		}
		batches[i] = make([]DownloadChunk, 1)
		batches[i][0] = DownloadChunk{start: start, size: size, writer: writer}
	}
	return batches
}

func (dm *downloadManager) batchChunksByMaxConcurrentParts(numChunks int64, writer WriteAtWriter) []Batch {
	if numChunks <= int64(dm.cfg.MaxConcurrentDownloads) {
		return dm.batchChunksByOne(numChunks, writer)
	}

	batchSize := numChunks / int64(dm.cfg.MaxConcurrentDownloads)
	batches := make([]Batch, dm.cfg.MaxConcurrentDownloads)
	for i := int64(0); i < int64(dm.cfg.MaxConcurrentDownloads); i++ {
		batches[i] = make([]DownloadChunk, 0, batchSize)
	}

	for i := int64(0); i < numChunks; i++ {
		batchIndex := i / batchSize
		start := i * dm.cfg.ChunkSize
		size := dm.cfg.ChunkSize
		if start+size > dm.totalBytes {
			size = dm.totalBytes - start
		}
		batches[batchIndex] = append(batches[batchIndex], DownloadChunk{start: start, size: size, writer: writer})
	}
	return batches
}

// ---------------------------- Download ----------------------------

// Download the file from the given url and save it to the given filename.
func (dm *downloadManager) download() (int64, error) {
	fileSize, err := dm.getFileSize(dm.url)
	if err != nil {
		return 0, err
	}
	dm.SetTotalBytes(fileSize)

	numChunks, chunkSize := dm.determineNumParts(fileSize)
	dm.cfg.ChunkSize = chunkSize
	batches := dm.batchChunks(numChunks, dm.writer)

	for i := 0; i < len(batches); i++ {
		dm.wg.Add(1)
		go dm.downloadBatch(batches[i])
	}
	dm.wg.Wait()

	if dm.GetError() != nil {
		return 0, dm.GetError()
	}

	return dm.GetWritten(), nil
}

// Download the batch of chunks
func (dm *downloadManager) downloadBatch(batch []DownloadChunk) {
	defer dm.wg.Done()

	for i := 0; i < len(batch); i++ {
		num, err := dm.downloadChunk(&batch[i])
		if err != nil {
			dm.SetError(err)
			return
		}
		dm.AddWritten(num)
	}
}

// Download the chunk
func (dm *downloadManager) downloadChunk(chunk *DownloadChunk) (int64, error) {
	var num int64
	var err error
	for i := 0; i < dm.cfg.MaxRetries; i++ {
		num, err = dm.tryDownloadChunk(chunk)
		if err == nil {
			return num, nil
		}
	}
	return num, err
}

// Try to download the chunk
func (dm *downloadManager) tryDownloadChunk(chunk *DownloadChunk) (int64, error) {
	req, err := http.NewRequest("GET", dm.url, nil)
	if err != nil {
		return 0, &DownloadError{Message: "Failed to create request", Err: err}
	}

	rangeHeader := chunk.GetBytesRange()
	req.Header.Set("Range", rangeHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, &DownloadError{Message: "Failed to do request", Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent {
		return 0, &DownloadError{Message: "Failed to download chunk", Err: err}
	}

	num, err := io.Copy(chunk, resp.Body)
	if err != nil {
		return 0, &DownloadError{Message: "Failed to write chunk", Err: err}
	}

	return num, nil
}
