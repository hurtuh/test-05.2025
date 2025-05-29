package main

import (
	"fmt"
	"io"
	"net/http"
)

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, fmt.Sprintf("ParseMultipartForm error: %v", err), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, fmt.Sprintf("FormFile error: %v", err), http.StatusBadRequest)
		return
	}
	defer file.Close()

	var size int64
	size, err = io.Copy(io.Discard, file)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading file: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Received file %s (%d bytes)\n", header.Filename, size)
}

func main() {
	http.HandleFunc("/upload", uploadHandler)
	fmt.Println("Listening on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
