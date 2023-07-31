package downloader

import (
	"os"
)

// Implement the WriteAt interface for the FileWriter struct.
//
// Write to the file at the given offset.
type FileWriter struct {
	file *os.File
}

// WriteAt writes len(p) bytes from p to the underlying data stream at offset off.
// It returns the number of bytes written from p (0 <= n <= len(p)) and any error encountered that caused the write to stop early.
// WriteAt must return a non-nil error if it returns n < len(p).
func (fw *FileWriter) WriteAt(p []byte, off int64) (n int, err error) {
	return fw.file.WriteAt(p, off)
}
