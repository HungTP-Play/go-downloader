package godownloader

import "os"

func createFile(filename string) (*os.File, error) {
	return os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0644)
}
