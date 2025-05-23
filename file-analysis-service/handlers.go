package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

type Handler struct {
	analyzer *Analyzer
}

func NewHandler(analyzer *Analyzer) *Handler {
	return &Handler{
		analyzer: analyzer,
	}
}

func (h *Handler) AnalyzeFile(w http.ResponseWriter, r *http.Request) {
	fileID := strings.TrimPrefix(r.URL.Path, "/analyze/")
	if fileID == "" {
		http.Error(w, "File ID is required", http.StatusBadRequest)
		return
	}

	result, err := h.analyzer.Analyze(fileID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *Handler) GetWordCloud(w http.ResponseWriter, r *http.Request) {
	imageID := strings.TrimPrefix(r.URL.Path, "/wordcloud/")
	if imageID == "" {
		http.Error(w, "Image ID is required", http.StatusBadRequest)
		return
	}

	imgData, err := h.analyzer.repo.GetWordCloud(imageID)
	if err != nil {
		http.Error(w, "Word cloud not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	if _, err := w.Write(imgData); err != nil {
		log.Printf("Failed to send word cloud: %v", err)
	}
}
