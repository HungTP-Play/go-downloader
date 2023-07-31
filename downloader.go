package godownloader

import (
	"context"
)

type DownloaderConfig struct {
	// The maximum of retry times for file
	//
	// If some error occurs when downloading a file, entire file will be re-downloaded.
	//
	// Default is 5.
	MaxRetries int

	// The maximum of concurrent downloads
	//
	// If `MaxConcurrentDownloads` is -1, mean that all the chunks will be downloaded concurrently.
	//
	// If `MaxConcurrentDownloads` is greater than 0, mean that
	// the chunks will be downloaded concurrently by `MaxConcurrentDownloads` goroutines.
	MaxConcurrentDownloads int

	// This function determines how many chunks will be split.
	//
	// You can use the default function `DefaultPartDeterminer` or write your own function.
	//
	// You should prefer using this option over `ChunkSizeDeterminer`.
	//
	// You should use `PartDeterminer` either `ChunkSizeDeterminer`, not both.
	PartDeterminerFunc PartDeterminer

	// This function determines the size of each chunk.
	//
	// You can use the default function `DefaultChunkSizeDeterminer` or write your own function.
	//
	// You should prefer using `PartDeterminerFunc` over this option.
	//
	// You should use `PartDeterminerFunc` either `ChunkSizeDeterminerFunc`, not both.
	ChunkSizeDeterminerFunc ChunkSizeDeterminer

	// The size of each chunk
	//
	// If it is set greater than 0, mean that the chunks will be downloaded by the given size.
	//
	// Otherwise, the chunks will be downloaded by the size determined by `PartDeterminerFunc` if given, then `ChunkSizeDeterminerFunc`.
	ChunkSize int64
}

type Downloader struct {
	config DownloaderConfig
}

// Enum size units
const (
	_ = 1 << (10 * iota)
	KB
	MB
	GB
)

var defaultPartDeterminer = func(totalSize int64) int64 {
	if totalSize < 1*MB {
		return 1
	}

	if totalSize < 10*MB {
		return 4
	}

	if totalSize < 100*MB {
		return 16
	}

	return 32
}

func defaultDownloaderConfiguration() DownloaderConfig {
	return DownloaderConfig{
		MaxRetries:             5,
		MaxConcurrentDownloads: -1,
		PartDeterminerFunc:     defaultPartDeterminer,
		ChunkSizeDeterminerFunc: func(totalSize int64) int64 {
			return totalSize / defaultPartDeterminer(totalSize)
		},
	}
}

// Return new Downloader with default configuration
//
// Default configuration:
//
//	MaxRetries: 5
//	MaxConcurrentDownloads: -1
//	PartDeterminerFunc: defaultPartDeterminer
//	ChunkSizeDeterminerFunc: defaultChunkSizeDeterminer
func NewDownloader() *Downloader {
	return NewDownloaderWithConfig(defaultDownloaderConfiguration())
}

// Return new Downloader with custom configuration
func NewDownloaderWithConfig(config DownloaderConfig) *Downloader {
	return &Downloader{config: config}
}

type DownloadOption func(*Downloader)

// Config the maximum of retry times for file
//
// If some error occurs when downloading a file, entire file will be re-downloaded.
//
// Default is 5.
func WithMaxRetries(maxRetries int) DownloadOption {
	return func(d *Downloader) {
		d.config.MaxRetries = maxRetries
	}
}

// Config the maximum of concurrent downloads
//
// If `MaxConcurrentDownloads` is -1, mean that all the chunks will be downloaded concurrently.
//
// If `MaxConcurrentDownloads` is greater than 0, mean that
// the chunks will be downloaded concurrently by `MaxConcurrentDownloads` goroutines.
func WithMaxConcurrentDownloads(maxConcurrentDownloads int) DownloadOption {
	return func(d *Downloader) {
		d.config.MaxConcurrentDownloads = maxConcurrentDownloads
	}
}

// Config the function determines how many chunks will be split.
//
// You can use the default function `DefaultPartDeterminer` or write your own function.
//
// You should prefer using this option over `ChunkSizeDeterminer`.
//
// You should use `WithPartDeterminerFunc` either `WithChunkSizeDeterminerFunc`, not both.
func WithPartDeterminerFunc(partDeterminerFunc PartDeterminer) DownloadOption {
	return func(d *Downloader) {
		d.config.PartDeterminerFunc = partDeterminerFunc
	}
}

// Config the function determines the size of each chunk.
//
// You can use the default function `DefaultChunkSizeDeterminer` or write your own function.
//
// You should prefer using `WithPartDeterminerFunc` over this option.
//
// You should use `WithPartDeterminerFunc` either `WithChunkSizeDeterminerFunc`, not both.
func WithChunkSizeDeterminerFunc(chunkSizeDeterminerFunc ChunkSizeDeterminer) DownloadOption {
	return func(d *Downloader) {
		d.config.ChunkSizeDeterminerFunc = chunkSizeDeterminerFunc
	}
}

// Return new Downloader with custom options
func NewWithOptions(options ...DownloadOption) *Downloader {
	downloader := NewDownloader()
	for _, option := range options {
		option(downloader)
	}
	return downloader
}

// ---------------------------- Implement IDownloader ----------------------------

// Download the file from the given url and save it to the given filename.
func (d *Downloader) Download(url string, filename string) (int64, error) {
	return d.DownloadWithContext(context.Background(), url, filename)
}

// Download the file from the given url and save it to the given filename.
//
// The context is used to cancel the download operation.
func (d *Downloader) DownloadWithContext(ctx context.Context, url string, filename string) (int64, error) {
	file, err := createFile(filename)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	fw := &FileWriter{file: file}

	downloadManager := &downloadManager{ctx: ctx, url: url, filename: filename, writer: fw, cfg: d.config}
	return downloadManager.download()
}
