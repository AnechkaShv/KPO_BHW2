package main

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
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
	existingAnalysis, err := a.repo.GetAnalysisByFileID(fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing analysis: %v", err)
	}
	if existingAnalysis != nil {
		return existingAnalysis, nil
	}

	content, err := a.repo.GetFileContent(fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file content: %v", err)
	}

	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("file content is empty")
	}

	// Подсчет статистики
	paragraphs := countParagraphs(content)
	words := countWords(content)
	characters := len([]rune(content))

	// Проверка на плагиат (выполняется всегда)
	plagiarismRate, similarFiles, err := a.checkPlagiarism(content, fileID)
	if err != nil {
		log.Printf("Plagiarism check warning: %v", err)
	}

	wordCloudURL := ""
	if len(strings.Fields(content)) >= 1 {
		url, err := a.generateWordCloud(content)
		if err != nil {
			log.Printf("Word cloud generation failed: %v", err)
		} else {
			wordCloudURL = url
		}
	} else {
		log.Printf("Text too short for word cloud generation")
	}

	// Создаем результат анализа
	result := AnalysisResult{
		ID:             uuid.New().String(),
		FileID:         fileID,
		Paragraphs:     paragraphs,
		Words:          words,
		Characters:     characters,
		PlagiarismRate: plagiarismRate,
		SimilarFiles:   similarFiles,
		WordCloudURL:   wordCloudURL,
	}

	if err := a.repo.SaveAnalysis(result); err != nil {
		return nil, fmt.Errorf("failed to save analysis result: %v", err)
	}

	return &result, nil
}

func (a *Analyzer) checkPlagiarism(content, currentFileID string) (float64, []SimilarFile, error) {
	similarFiles, err := a.repo.FindSimilarFiles(content, currentFileID)
	if err != nil {
		return 0, nil, err
	}

	if len(similarFiles) > 0 {
		return similarFiles[0].Similarity, similarFiles, nil
	}
	return 0, nil, nil
}

func (a *Analyzer) generateWordCloud(content string) (string, error) {
	cleanedContent := cleanText(content)

	wordCloudURL := fmt.Sprintf("https://quickchart.io/wordcloud?text=%s&width=800&height=600&format=png",
		url.QueryEscape(cleanedContent))

	return wordCloudURL, nil
}

func cleanText(text string) string {
	// Keep basic punctuation that might be part of words
	replacer := strings.NewReplacer(
		"\n", " ", "\r", " ", "\t", " ",
		"(", " ", ")", " ", "[", " ", "]", " ",
		"{", " ", "}", " ", "\"", " ", "'", " ",
	)
	cleaned := replacer.Replace(text)

	// Remove standalone punctuation but keep words with apostrophes
	reg := regexp.MustCompile(`(^|\s)[^\w']+(\s|$)|[^\w'\s]`)
	cleaned = reg.ReplaceAllString(cleaned, " ")

	// Remove multiple spaces
	cleaned = strings.Join(strings.Fields(cleaned), " ")

	return cleaned
}

func countWords(text string) int {
	words := strings.Fields(text)
	return len(words)
}

func countParagraphs(text string) int {
	paragraphs := strings.Split(text, "\n\n")
	nonEmptyParagraphs := 0
	for _, p := range paragraphs {
		if strings.TrimSpace(p) != "" {
			nonEmptyParagraphs++
		}
	}
	return nonEmptyParagraphs
}
