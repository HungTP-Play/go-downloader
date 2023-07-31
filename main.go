package main

import (
	"fmt"

	"github.com/HungTP-Play/go-downloader/downloader"
)

func main() {
	url := "https://files.catbox.moe/em97nz.jpg"
	filepath := "spider_name.jpg"

	// Create a new downloader
	downloader := downloader.NewDownloader()
	n, err := downloader.Download(url, filepath)
	if err != nil {
		panic(err)
	}

	fmt.Println("Downloaded", n, "bytes")
}
