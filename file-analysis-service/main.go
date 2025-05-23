package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	repo := NewPostgresRepository()
	analyzer := NewAnalyzer(repo, os.Getenv("WORDCLOUD_API_URL"))
	handler := NewHandler(analyzer)

	http.HandleFunc("/analyze/", handler.AnalyzeFile)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	log.Printf("File Analysis Service is running on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
