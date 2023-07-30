package godownloader

// WriteAt is the interface that wraps the basic WriteAt method.
type WriteAtWriter interface {
	// WriteAt writes len(p) bytes from p to the underlying data stream at offset off.
	// It returns the number of bytes written from p (0 <= n <= len(p)) and any error encountered that caused the write to stop early.
	// WriteAt must return a non-nil error if it returns n < len(p).
	WriteAt(p []byte, off int64) (n int, err error)
}
