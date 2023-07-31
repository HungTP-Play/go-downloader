package downloader

import (
	"fmt"
)

type DownloadChunk struct {
	start   int64
	size    int64
	current int64

	writer WriteAtWriter
}

func (dc *DownloadChunk) String() string {
	return fmt.Sprintf("DownloadChunk{start=%d, size=%d, current=%d}", dc.start, dc.size, dc.current)
}

func (dc *DownloadChunk) GetBytesRange() string {
	return fmt.Sprintf("bytes=%d-%d", dc.start, dc.start+dc.size-1)
}

func (dc *DownloadChunk) Write(p []byte) (n int, err error) {
	if dc.current >= dc.size {
		return 0, nil
	}

	n, err = dc.writer.WriteAt(p, dc.start+dc.current)
	dc.current += int64(n)
	return n, err
}
