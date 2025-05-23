package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type Analyzer struct {
	repo         Repository
	wordCloudAPI string
}

func NewAnalyzer(repo Repository, wordCloudAPI string) *Analyzer {
	return &Analyzer{
		repo:         repo,
		wordCloudAPI: wordCloudAPI,
	}
}

func (a *Analyzer) Analyze(fileID string) (*AnalysisResult, error) {
	// Проверяем существующий анализ
	existing, err := a.repo.GetAnalysisByFileID(fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing analysis: %v", err)
	}
	if existing != nil {
		return existing, nil
	}

	// Получаем содержимое файла
	content, err := a.repo.GetFileContent(fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file content: %v", err)
	}

	// Базовый анализ текста
	paragraphs := strings.Count(content, "\n\n") + 1
	words := len(strings.Fields(content))
	characters := len([]rune(content))

	// Проверка на плагиат
	plagiarism, err := a.checkPlagiarism(content)
	if err != nil {
		log.Printf("Plagiarism check failed: %v", err)
	}

	// Генерация облака слов
	wordCloudURL, err := a.generateWordCloud(content)
	if err != nil {
		log.Printf("Word cloud generation failed: %v", err)
	}

	// Сохраняем результат
	result := AnalysisResult{
		ID:           uuid.New().String(),
		FileID:       fileID,
		Paragraphs:   paragraphs,
		Words:        words,
		Characters:   characters,
		Plagiarism:   plagiarism,
		WordCloudURL: wordCloudURL,
	}

	if err := a.repo.SaveAnalysis(result); err != nil {
		return nil, fmt.Errorf("failed to save analysis: %v", err)
	}

	return &result, nil
}

func (a *Analyzer) checkPlagiarism(content string) (float64, error) {
	similarFiles, err := a.repo.FindSimilarFiles(content)
	if err != nil {
		return 0, err
	}

	if len(similarFiles) > 0 {
		return similarFiles[0].Similarity, nil
	}
	return 0, nil
}

func (a *Analyzer) generateWordCloud(content string) (string, error) {
	// Подготавливаем запрос к Word Cloud API
	requestBody := fmt.Sprintf(`{
		"text": "%s",
		"width": 800,
		"height": 600,
		"format": "png",
		"removeStopwords": true,
		"caseSensitive": false,
		"maxNumWords": 100
	}`, strings.ReplaceAll(content, `"`, `\"`))

	// Отправляем запрос
	resp, err := http.Post(a.wordCloudAPI, "application/json", bytes.NewBufferString(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to request word cloud: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("word cloud API returned status %d", resp.StatusCode)
	}

	// Сохраняем изображение
	imageID := uuid.New().String()
	if err := a.repo.SaveWordCloud(imageID, resp.Body); err != nil {
		return "", fmt.Errorf("failed to save word cloud: %v", err)
	}

	return fmt.Sprintf("/wordcloud/%s", imageID), nil
}
