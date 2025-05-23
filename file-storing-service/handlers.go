package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Handler struct {
	repo Repository
}

func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) UploadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read file content
	contentBytes, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file content", http.StatusInternalServerError)
		return
	}
	content := string(contentBytes)

	// Calculate file hash
	hash := sha256.New()
	hash.Write(contentBytes)
	hashSum := hex.EncodeToString(hash.Sum(nil))

	// Check if file already exists
	existingFile, err := h.repo.GetFileByHash(hashSum)
	if err != nil {
		http.Error(w, "Failed to check file existence", http.StatusInternalServerError)
		return
	}

	if existingFile != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"id": existingFile.ID})
		return
	}

	// Generate unique ID and location
	id := uuid.New().String()
	ext := filepath.Ext(header.Filename)
	location := strings.TrimSuffix(header.Filename, ext) + "-" + time.Now().Format("20060102150405") + ext

	// Save file metadata and content
	metadata := FileMetadata{
		ID:       id,
		Name:     header.Filename,
		Hash:     hashSum,
		Location: location,
	}

	fileID, err := h.repo.SaveFile(metadata, content)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": fileID})
}

func (h *Handler) GetFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/files/")
	if id == "" {
		http.Error(w, "File ID is required", http.StatusBadRequest)
		return
	}

	file, err := h.repo.GetFile(id)
	if err != nil {
		http.Error(w, "Failed to get file", http.StatusInternalServerError)
		return
	}

	if file == nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(file)
}

func (h *Handler) GetFileContent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	location := strings.TrimPrefix(r.URL.Path, "/files/content/")
	if location == "" {
		http.Error(w, "File location is required", http.StatusBadRequest)
		return
	}

	content, err := h.repo.GetFileContent(location)
	if err != nil {
		http.Error(w, "Failed to get file content", http.StatusInternalServerError)
		return
	}

	if content == "" {
		http.Error(w, "File content not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(content))
}
