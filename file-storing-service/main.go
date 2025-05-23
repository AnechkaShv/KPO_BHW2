package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	repo := NewPostgresRepository()
	handler := NewHandler(repo)

	http.HandleFunc("/files", handler.UploadFile)
	http.HandleFunc("/files/", handler.GetFile)
	http.HandleFunc("/files/content/", handler.GetFileContent)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("File Storing Service is running on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
