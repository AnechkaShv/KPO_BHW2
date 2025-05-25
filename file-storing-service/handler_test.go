package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type MockRepository struct {
	Files        map[string]FileMetadata
	FileContents map[string]string
	ErrorMode    bool
}

func (m *MockRepository) GetFileByHash(hash string) (*FileMetadata, error) {
	if m.ErrorMode {
		return nil, errors.New("mock error")
	}
	for _, file := range m.Files {
		if file.Hash == hash {
			return &file, nil
		}
	}
	return nil, nil
}

func (m *MockRepository) SaveFile(metadata FileMetadata, content string) (string, error) {
	if m.ErrorMode {
		return "", errors.New("mock error")
	}
	m.Files[metadata.ID] = metadata
	m.FileContents[metadata.Location] = content
	return metadata.ID, nil
}

func (m *MockRepository) GetFile(id string) (*FileMetadata, error) {
	if m.ErrorMode {
		return nil, errors.New("mock error")
	}
	file, exists := m.Files[id]
	if !exists {
		return nil, nil
	}
	return &file, nil
}

func (m *MockRepository) GetFileContent(location string) (string, error) {
	if m.ErrorMode {
		return "", errors.New("mock error")
	}
	content, exists := m.FileContents[location]
	if !exists {
		return "", nil
	}
	return content, nil
}

func TestFileHandlers(t *testing.T) {
	mockRepo := &MockRepository{
		Files:        make(map[string]FileMetadata),
		FileContents: make(map[string]string),
	}
	handler := NewHandler(mockRepo)

	t.Run("Upload new file", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "test.txt")
		part.Write([]byte("test content"))
		writer.Close()

		req := httptest.NewRequest("POST", "/files", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rr := httptest.NewRecorder()

		handler.UploadFile(rr, req)

		if rr.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d", http.StatusCreated, rr.Code)
		}

		var response map[string]string
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatal(err)
		}

		if _, exists := mockRepo.Files[response["id"]]; !exists {
			t.Error("file was not saved in repository")
		}
	})

	t.Run("Upload duplicate file", func(t *testing.T) {
		content := "duplicate content"
		body1 := &bytes.Buffer{}
		writer1 := multipart.NewWriter(body1)
		part1, _ := writer1.CreateFormFile("file", "test.txt")
		part1.Write([]byte(content))
		writer1.Close()

		req1 := httptest.NewRequest("POST", "/files", body1)
		req1.Header.Set("Content-Type", writer1.FormDataContentType())
		rr1 := httptest.NewRecorder()
		handler.UploadFile(rr1, req1)

		body2 := &bytes.Buffer{}
		writer2 := multipart.NewWriter(body2)
		part2, _ := writer2.CreateFormFile("file", "test.txt")
		part2.Write([]byte(content))
		writer2.Close()

		req2 := httptest.NewRequest("POST", "/files", body2)
		req2.Header.Set("Content-Type", writer2.FormDataContentType())
		rr2 := httptest.NewRecorder()
		handler.UploadFile(rr2, req2)

		if rr2.Code != http.StatusOK {
			t.Errorf("expected status %d for duplicate, got %d", http.StatusOK, rr2.Code)
		}
	})

	t.Run("Get existing file", func(t *testing.T) {
		fileID := "test-file"
		mockRepo.Files[fileID] = FileMetadata{
			ID:       fileID,
			Name:     "test.txt",
			Location: "test-location",
		}

		req := httptest.NewRequest("GET", "/files/"+fileID, nil)
		rr := httptest.NewRecorder()
		handler.GetFile(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}

		var fileResponse FileMetadata
		if err := json.NewDecoder(rr.Body).Decode(&fileResponse); err != nil {
			t.Fatal(err)
		}

		if fileResponse.ID != fileID {
			t.Errorf("expected file ID %s, got %s", fileID, fileResponse.ID)
		}
	})

	t.Run("Get non-existent file", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/files/nonexistent", nil)
		rr := httptest.NewRecorder()
		handler.GetFile(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, rr.Code)
		}
	})

	t.Run("Get file content - success", func(t *testing.T) {
		location := "test-location"
		expectedContent := "test file content"
		mockRepo.FileContents[location] = expectedContent

		req := httptest.NewRequest("GET", "/files/content/"+location, nil)
		rr := httptest.NewRecorder()
		handler.GetFileContent(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}

		if !strings.Contains(rr.Body.String(), expectedContent) {
			t.Errorf("expected content '%s', got '%s'", expectedContent, rr.Body.String())
		}
	})

	t.Run("Get non-existent file content", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/files/content/nonexistent", nil)
		rr := httptest.NewRecorder()
		handler.GetFileContent(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, rr.Code)
		}
	})

	t.Run("Upload file error", func(t *testing.T) {
		mockRepo.ErrorMode = true
		defer func() { mockRepo.ErrorMode = false }()

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "test.txt")
		part.Write([]byte("test content"))
		writer.Close()

		req := httptest.NewRequest("POST", "/files", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rr := httptest.NewRecorder()

		handler.UploadFile(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})
}
