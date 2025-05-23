package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	// Initialize generator
	generator := NewWordCloudGenerator()

	// Initialize handler
	handler := NewWordCloudHandler(generator)

	// Router
	r := mux.NewRouter()

	r.HandleFunc("/api/wordcloud", handler.GenerateWordCloud).Methods("POST")
	r.HandleFunc("/api/wordcloud/{id}", handler.GetWordCloud).Methods("GET")
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	log.Println("Word Cloud Service is running on :8083")
	log.Fatal(http.ListenAndServe(":8083", r))
}
