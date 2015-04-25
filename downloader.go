package multipartdownloader

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"time"
)

// Info gathered from different sources
type URLInfo struct {
	url string
	fileLength int64
	etag string
	connSuccess bool
	statusCode int
}

// Chunk boundaries
type chunk struct {
	begin int64
	end int64
}

// The file downloader
type MultiDownloader struct {
	urls []string            // List of all sources for the file
	nConns int               // Number of max concurrent connections to use
	timeout time.Duration    // Timeout for all connections
	fileLength int64         // Size of the file. It could be larger than 4GB.
	filename string          // Output filename
	partFilename string      // Incomplete output filename
	etag string              // ETag (if available) of the file
	chunks []chunk           // A table of the chunks the file is divided into
}

func NewMultiDownloader(urls []string, nConns int, timeout time.Duration) *MultiDownloader {
	return &MultiDownloader{urls: urls, nConns: nConns, timeout: timeout}
}

// Get the info of the file, using the HTTP HEAD request
func (dldr *MultiDownloader) GatherInfo() (err error) {
	if len(dldr.urls) == 0 {
		return errors.New("No URLs provided")
	}

	results := make(chan URLInfo)
	defer close(results)

	// Connect to all sources concurrently
	getHead := func (url string) {
		client := http.Client{
			Timeout: time.Duration(dldr.timeout),
		}
		resp, err := client.Head(url)
		if err != nil {
			results <- URLInfo{url: url, connSuccess: false, statusCode: 0}
			return
		}
		defer resp.Body.Close()
		flen, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 0, 64)
		etag := resp.Header.Get("Etag")
		if err != nil {
			log.Println("Error reading Content-Length from HTTP header")
			flen = 0
		}
		results <- URLInfo{
			url: url,
			fileLength: flen,
			etag: etag,
			connSuccess: true,
			statusCode: resp.StatusCode,
		}
	}
	for _, url := range dldr.urls {
		go getHead(url)
	}

	// Gather the results and return if something is wrong
	resArray := make([]URLInfo, len(dldr.urls))
	for i := 0; i < len(dldr.urls); i++ {
		r := <-results
		resArray[i] = r
		if !r.connSuccess || r.statusCode != 200 {
			return errors.New(fmt.Sprintf("Failed connection to URL %s", resArray[i].url))
		}
	}

	// Check that all sources agree on file length and Etag
	// Empty Etags are also accepted
	commonFileLength := resArray[0].fileLength
	commonEtag := resArray[0].etag
	for _, r := range resArray[1:] {
		if r.fileLength != commonFileLength || (len(r.etag) != 0 && r.etag != commonEtag) {
			return errors.New("URLs must point to the same file")
		}
	}
	dldr.fileLength = commonFileLength
	dldr.etag = commonEtag
	dldr.filename = urlToFilename(resArray[0].url)
	dldr.partFilename = dldr.filename + ".part"

	LogVerbose("File length: ", dldr.fileLength)
	LogVerbose("Etag: ", dldr.etag)

	return
}

// Prepare the file used for writing the blocks of data
func (dldr *MultiDownloader) SetupFile(filename string) (os.FileInfo, error) {
	var file *os.File
	var err error
	if filename != "" {
		dldr.filename = filename
		dldr.partFilename = filename + ".part"
	}
	file, err = os.Create(dldr.filename)
	if err != nil {
		return nil, err
	}

	// Force file size in order to write arbitrary chunks
	err = file.Truncate(dldr.fileLength)
	fileInfo, err := file.Stat()
	return fileInfo, err
}

// Internal: build the chunks table, deciding boundaries
func (dldr *MultiDownloader) buildChunks() {
	// The algorithm takes care of possible rounding errors splitting into chunks
	// by taking out the remainder and distributing it among the first chunks
	n := int64(dldr.nConns)
	remainder := dldr.fileLength % n
	exactNumerator := dldr.fileLength - remainder
	chunkSize := exactNumerator / n
	dldr.chunks = make([]chunk, n)
	boundary := int64(0)
	nextBoundary := chunkSize
	for i := int64(0); i < n; i++ {
		if remainder > 0 {
			remainder--
			nextBoundary++
		}
		dldr.chunks[i] = chunk{boundary, nextBoundary}
		boundary = nextBoundary
		nextBoundary = nextBoundary + chunkSize
	}
}

// Perform the concurrent download
func (dldr *MultiDownloader) Download() (err error) {
	/*
	dldr.buildChunks()
	downloadChunk := func(begin uint64, end uint64) {
	}
	for i := 0; i < dldr.nConns; i++ {
	}
        */
	return
}

// Check SHA-256 of downloaded file
func (dldr *MultiDownloader) CheckSHA256(sha256 string) (err error, ok bool) {
	return
}

// Check ETag as MD5SUM
func (dldr *MultiDownloader) CheckETag() (err error, ok bool) {
	return
}


////////////////////////////////////////////////////////////////////////////////
// Auxiliary functions

// Get the name of the file from the URL
func urlToFilename(urlStr string) string {
	url, err := url.Parse(urlStr)
	if err != nil {
		return "downloaded-file"
	}
	_, f := path.Split(url.Path)
	return f
}
