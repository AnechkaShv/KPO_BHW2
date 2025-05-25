package main

import (
	"errors"
	"testing"
)

type MockRepository struct {
	Files          map[string]string
	WordClouds     map[string][]byte
	FileMetadatas  map[string]FileMetadata
	AnalysisResult *AnalysisResult
	SimilarFiles   []SimilarFile
	ErrorMode      bool
}

func (m *MockRepository) GetFileContent(fileID string) (string, error) {
	if m.ErrorMode {
		return "", errors.New("mock error")
	}
	return m.Files[fileID], nil
}

func (m *MockRepository) FindSimilarFiles(content, currentFileID string) ([]SimilarFile, error) {
	if m.ErrorMode {
		return nil, errors.New("mock error")
	}
	return m.SimilarFiles, nil
}

func (m *MockRepository) SaveAnalysis(result AnalysisResult) error {
	if m.ErrorMode {
		return errors.New("mock error")
	}
	m.AnalysisResult = &result
	return nil
}

func (m *MockRepository) GetAnalysisByFileID(fileID string) (*AnalysisResult, error) {
	if m.ErrorMode {
		return nil, errors.New("mock error")
	}
	return m.AnalysisResult, nil
}

func (m *MockRepository) GetFileMetadata(fileID string) (*FileMetadata, error) {
	if m.ErrorMode {
		return nil, errors.New("mock error")
	}
	metadata, exists := m.FileMetadatas[fileID]
	if !exists {
		return nil, nil
	}
	return &metadata, nil
}

func (m *MockRepository) SaveWordCloud(id string, image []byte) error {
	if m.ErrorMode {
		return errors.New("mock error")
	}
	m.WordClouds[id] = image
	return nil
}

func (m *MockRepository) GetWordCloud(id string) ([]byte, error) {
	if m.ErrorMode {
		return nil, errors.New("mock error")
	}
	return m.WordClouds[id], nil
}

func (m *MockRepository) GetAllFilesExcept(fileID string) ([]FileForComparison, error) {
	if m.ErrorMode {
		return nil, errors.New("mock error")
	}
	var files []FileForComparison
	for id, content := range m.Files {
		if id != fileID {
			files = append(files, FileForComparison{
				ID:      id,
				Content: content,
			})
		}
	}
	return files, nil
}

func TestAnalyzer(t *testing.T) {
	tests := []struct {
		name          string
		fileID        string
		repoSetup     func() *MockRepository
		expectError   bool
		expectedWords int
	}{
		{
			name:   "Successful analysis with similar files",
			fileID: "file1",
			repoSetup: func() *MockRepository {
				return &MockRepository{
					Files: map[string]string{
						"file1": "This is a test content with five words",
						"file2": "Similar content with some matching words",
					},
					FileMetadatas: map[string]FileMetadata{
						"file1": {ID: "file1", Name: "test.txt"},
					},
					SimilarFiles: []SimilarFile{
						{FileID: "file2", Similarity: 0.6},
					},
					WordClouds: make(map[string][]byte),
				}
			},
			expectError:   false,
			expectedWords: 8,
		},
		{
			name:   "Error getting file content",
			fileID: "file1",
			repoSetup: func() *MockRepository {
				return &MockRepository{
					ErrorMode: true,
				}
			},
			expectError: true,
		},
		{
			name:   "Empty file content",
			fileID: "empty",
			repoSetup: func() *MockRepository {
				return &MockRepository{
					Files: map[string]string{
						"empty": "",
					},
					FileMetadatas: map[string]FileMetadata{
						"empty": {ID: "empty", Name: "empty.txt"},
					},
					WordClouds: make(map[string][]byte),
				}
			},
			expectError:   false,
			expectedWords: 0,
		},
		{
			name:   "Error saving analysis result",
			fileID: "file1",
			repoSetup: func() *MockRepository {
				return &MockRepository{
					Files: map[string]string{
						"file1": "Test content",
					},
					FileMetadatas: map[string]FileMetadata{
						"file1": {ID: "file1", Name: "test.txt"},
					},
					ErrorMode: true,
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := tt.repoSetup()
			analyzer := NewAnalyzer(mockRepo, "http://mock-wordcloud")

			result, err := analyzer.Analyze(tt.fileID)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.Words != tt.expectedWords {
				t.Errorf("expected %d words, got %d", tt.expectedWords, result.Words)
			}

			if tt.name == "Successful analysis with similar files" && len(result.SimilarFiles) == 0 {
				t.Error("expected similar files, got none")
			}
		})
	}
}

func TestWordCloudGeneration(t *testing.T) {
	mockRepo := &MockRepository{
		Files: map[string]string{
			"file1": "This is a test content for word cloud generation",
		},
		FileMetadatas: map[string]FileMetadata{
			"file1": {ID: "file1", Name: "test.txt"},
		},
		WordClouds: make(map[string][]byte),
	}

	analyzer := NewAnalyzer(mockRepo, "http://mock-wordcloud")
	result, err := analyzer.Analyze("file1")
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if result.WordCloudID == "" {
		t.Error("word cloud ID should be generated")
	}

	if len(mockRepo.WordClouds) == 0 {
		t.Error("word cloud should be saved in repository")
	}
}
