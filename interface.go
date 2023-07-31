package godownloader

import (
	"context"
	"fmt"
)

// WriteAt is the interface that wraps the basic WriteAt method.
type WriteAtWriter interface {
	// WriteAt writes len(p) bytes from p to the underlying data stream at offset off.
	// It returns the number of bytes written from p (0 <= n <= len(p)) and any error encountered that caused the write to stop early.
	// WriteAt must return a non-nil error if it returns n < len(p).
	WriteAt(p []byte, off int64) (n int, err error)
}

// `PartDeterminer` is a function that determines the number of chunks will be split.
//
// The parameter is the total size of the file.
type PartDeterminer func(int64) int64

// `ChunkSizeDeterminer` is a function that determines the size of each chunk.
//
// The parameter is the total size of the file.
type ChunkSizeDeterminer func(int64) int64

type IDownloader interface {
	// Download the file from the given url and save it to the given filename.
	Download(url string, filename string) error

	// Download the file from the given url and save it to the given filename with the given context.
	//
	// You should not give a nil context.
	DownloadWithContext(ctx context.Context, url string, filename string) error
}

type DownloadError struct {
	Message string
	Err     error
}

func (e *DownloadError) Error() string {
	return fmt.Sprintf("DownloadError{Message=%s, Err=%s}", e.Message, e.Err)
}
