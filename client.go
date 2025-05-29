package main

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

func monitorMemory(limit uint64, done <-chan struct{}) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			if m.Alloc > limit {
				fmt.Fprintf(os.Stderr, "Memory limit exceeded: used=%d, limit=%d\n", m.Alloc, limit)
				os.Exit(1)
			}
		}
	}
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run client.go <max_memory_bytes> <file_path>")
		os.Exit(1)
	}

	maxMem, err := strconv.ParseUint(os.Args[1], 10, 64)
	if err != nil {
		panic(err)
	}
	filePath := os.Args[2]

	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		panic(err)
	}

	if stat.Size() > int64(maxMem) {
		panic(fmt.Sprintf("File size exceeded: used=%d, limit=%d", stat.Size(), maxMem))
	}

	fmt.Printf("Max memory allowed: %d bytes\n", maxMem)
	fmt.Printf("Uploading file: %s\n", filePath)

	done := make(chan struct{})
	go monitorMemory(maxMem, done)

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		part, err := writer.CreateFormFile("file", filepath.Base(filePath))
		if err != nil {
			pw.CloseWithError(err)
			return
		}

		if maxMem < 1024 {
			maxMem = 1024
		}
		buf := make([]byte, maxMem)

		_, err = io.CopyBuffer(part, file, buf)
		if err != nil {
			pw.CloseWithError(err)
			return
		}

		writer.Close()
		pw.Close()
	}()

	req, err := http.NewRequest("POST", "http://localhost:8080/upload", pr)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	close(done)

	_, _ = io.Copy(os.Stdout, resp.Body)
}
