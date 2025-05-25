package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/google/uuid"
)

const (
	maxTextLengthForWordCloud = 10000
	minWordsForWordCloud      = 1
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
	content, err := a.repo.GetFileContent(fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file content: %v", err)
	}

	paragraphs := CountParagraphs(content)
	words := CountWords(content)
	characters := len([]rune(content))

	_, similarFiles, err := a.calculatePlagiarism(content, fileID)
	if err != nil {
		log.Printf("Plagiarism calculation warning: %v", err)
	}

	wordCloudID := ""
	if words >= minWordsForWordCloud {
		id, err := a.generateWordCloud(content)
		if err != nil {
			log.Printf("Word cloud generation warning: %v", err)
		} else {
			wordCloudID = id
		}
	}

	result := AnalysisResult{
		ID:           uuid.New().String(),
		FileID:       fileID,
		Paragraphs:   paragraphs,
		Words:        words,
		Characters:   characters,
		SimilarFiles: similarFiles,
		WordCloudID:  wordCloudID,
	}

	if err := a.repo.SaveAnalysis(result); err != nil {
		return nil, fmt.Errorf("failed to save analysis result: %v", err)
	}

	return &result, nil
}

func (a *Analyzer) calculatePlagiarism(content string, fileID string) (float64, []SimilarFile, error) {
	files, err := a.repo.GetAllFilesExcept(fileID)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get files for comparison: %v", err)
	}

	currentWords := strings.Fields(cleanText(content))
	if len(currentWords) == 0 {
		return 0, nil, nil
	}

	var similarFiles []SimilarFile
	totalUniqueWords := make(map[string]bool)
	plagiarizedWords := make(map[string]bool)

	for _, file := range files {
		fileWords := strings.Fields(cleanText(file.Content))
		fileWordSet := make(map[string]bool)

		for _, word := range fileWords {
			fileWordSet[word] = true
			totalUniqueWords[word] = true
		}

		matches := 0
		for _, word := range currentWords {
			if fileWordSet[word] {
				plagiarizedWords[word] = true
				matches++
			}
		}

		if len(fileWords) > 0 {
			similarity := float64(matches) / float64(len(currentWords)) * 100
			if similarity > 5 { // Порог в 5%
				similarFiles = append(similarFiles, SimilarFile{
					FileID:     file.ID,
					Name:       file.Name,
					Similarity: similarity,
				})
			}
		}
	}

	var plagiarismRate float64
	if len(totalUniqueWords) > 0 {
		plagiarismRate = float64(len(plagiarizedWords)) / float64(len(currentWords)) * 100
	}

	sort.Slice(similarFiles, func(i, j int) bool {
		return similarFiles[i].Similarity > similarFiles[j].Similarity
	})

	return plagiarismRate, similarFiles, nil
}

func (a *Analyzer) generateWordCloud(content string) (string, error) {
	cleanedContent := cleanText(content)
	if len(strings.Fields(cleanedContent)) < minWordsForWordCloud {
		return "", fmt.Errorf("not enough meaningful words after cleaning")
	}

	cloudID := uuid.New().String()

	wordCloudURL := fmt.Sprintf("https://quickchart.io/wordcloud?text=%s&width=800&height=600&format=png&padding=2",
		url.QueryEscape(cleanedContent))

	resp, err := http.Get(wordCloudURL)
	if err != nil {
		return "", fmt.Errorf("failed to download word cloud: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("word cloud API returned status %d", resp.StatusCode)
	}

	imgData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read word cloud image: %v", err)
	}

	if err := a.repo.SaveWordCloud(cloudID, imgData); err != nil {
		return "", fmt.Errorf("failed to save word cloud to DB: %v", err)
	}

	return cloudID, nil
}

func cleanText(text string) string {
	text = strings.ToLower(text)

	reg := regexp.MustCompile(`[^\w\s'-]`)
	text = reg.ReplaceAllString(text, "")

	text = strings.Join(strings.Fields(text), " ")

	return text
}

func CountWords(text string) int {
	words := strings.Fields(text)
	return len(words)
}

func CountParagraphs(text string) int {
	paragraphs := strings.Split(text, "\n\n")
	nonEmptyParagraphs := 0
	for _, p := range paragraphs {
		if strings.TrimSpace(p) != "" {
			nonEmptyParagraphs++
		}
	}
	return nonEmptyParagraphs
}
