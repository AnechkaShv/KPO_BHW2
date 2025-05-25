package main

import (
	"os"
	"testing"
)

func TestPostgresRepository(t *testing.T) {
	if os.Getenv("INTEGRATION") != "true" {
		t.Skip("Skipping integration test")
	}

	repo := NewPostgresRepository()

	t.Run("SaveAndGetAnalysis", func(t *testing.T) {
		result := AnalysisResult{
			ID:         "test",
			FileID:     "file1",
			Paragraphs: 1,
			Words:      10,
			Characters: 50,
		}

		err := repo.SaveAnalysis(result)
		if err != nil {
			t.Fatalf("SaveAnalysis failed: %v", err)
		}

		got, err := repo.GetAnalysisByFileID("file1")
		if err != nil {
			t.Fatalf("GetAnalysisByFileID failed: %v", err)
		}

		if got.Words != 10 {
			t.Errorf("Expected 10 words, got %d", got.Words)
		}
	})
}
