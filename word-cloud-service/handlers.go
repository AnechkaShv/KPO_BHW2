package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

type WordCloudHandler struct {
	generator *WordCloudGenerator
}

func NewWordCloudHandler(generator *WordCloudGenerator) *WordCloudHandler {
	return &WordCloudHandler{generator: generator}
}

func (h *WordCloudHandler) GenerateWordCloud(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Text string `json:"text"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Generate word cloud image
	imageBytes, err := h.generator.Generate(request.Text)
	if err != nil {
		http.Error(w, "Failed to generate word cloud", http.StatusInternalServerError)
		return
	}

	// Save image to file
	location, err := saveImage(imageBytes)
	if err != nil {
		http.Error(w, "Failed to save word cloud", http.StatusInternalServerError)
		return
	}

	// Return location
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"location": location,
	})
}

func (h *WordCloudHandler) GetWordCloud(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Path[len("/api/wordcloud/"):]
	location := filepath.Join("wordclouds", vars)

	// Open image file
	file, err := os.Open(location)
	if err != nil {
		http.Error(w, "Word cloud not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	// Serve image
	w.Header().Set("Content-Type", "image/png")
	io.Copy(w, file)
}

func saveImage(imageBytes []byte) (string, error) {
	// Ensure directory exists
	dir := "wordclouds"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	// Generate unique filename
	filename := fmt.Sprintf("%s-%s.png", time.Now().Format("20060102"), uuid.New().String())
	location := filepath.Join(dir, filename)

	// Save file
	if err := os.WriteFile(location, imageBytes, 0644); err != nil {
		return "", err
	}

	return location, nil
}
