# Go-downloader

Simple downloader written in Go, multithreaded download.

## Usage

```go

import "github.com/HungTP-Play/go-downloader/downloader"

func main() {
    url := "https://www.example.com/file.zip"
    path := "/home/user/Downloads/file.zip"
    downloader := downloader.NewDownloader()
    n, err := downloader.Download(url, path)
    if err != nil {
        fmt.Println(err)
    }
    fmt.Println("Downloaded", n, "bytes")
}

```

## Configuration

```go
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
```
