package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
)

type AnalysisResult struct {
	ID           string  `json:"id"`
	FileID       string  `json:"file_id"`
	Paragraphs   int     `json:"paragraphs"`
	Words        int     `json:"words"`
	Characters   int     `json:"characters"`
	Plagiarism   float64 `json:"plagiarism"`
	WordCloudURL string  `json:"word_cloud_url"`
}

type SimilarFile struct {
	FileID     string  `json:"file_id"`
	Similarity float64 `json:"similarity"`
}

type Repository interface {
	GetFileContent(fileID string) (string, error)
	FindSimilarFiles(content string) ([]SimilarFile, error)
	SaveAnalysis(result AnalysisResult) error
	SaveWordCloud(id string, image io.Reader) error
	GetWordCloud(id string) (io.Reader, error)
	GetAnalysisByFileID(fileID string) (*AnalysisResult, error)
}

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository() *PostgresRepository {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	// Убедимся, что расширение pg_trgm установлено
	_, err = db.Exec("CREATE EXTENSION IF NOT EXISTS pg_trgm")
	if err != nil {
		log.Fatal(err)
	}

	// Создаем таблицы
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS analysis_results (
			id TEXT PRIMARY KEY,
			file_id TEXT NOT NULL UNIQUE,
			paragraphs INTEGER NOT NULL,
			words INTEGER NOT NULL,
			characters INTEGER NOT NULL,
			plagiarism FLOAT NOT NULL,
			word_cloud_url TEXT NOT NULL
		)
	`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS word_clouds (
			id TEXT PRIMARY KEY,
			image BYTEA NOT NULL
		)
	`)
	if err != nil {
		log.Fatal(err)
	}

	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) FindSimilarFiles(content string) ([]SimilarFile, error) {
	rows, err := r.db.Query(`
		SELECT fm.id, 
		       similarity(fc.content, $1) as similarity
		FROM file_content fc
		JOIN file_metadata fm ON fc.location = fm.location
		WHERE similarity(fc.content, $1) > 0.7
		ORDER BY similarity DESC
		LIMIT 5
	`, content)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SimilarFile
	for rows.Next() {
		var sf SimilarFile
		if err := rows.Scan(&sf.FileID, &sf.Similarity); err != nil {
			return nil, err
		}
		results = append(results, sf)
	}
	return results, nil
}

func (r *PostgresRepository) SaveAnalysis(result AnalysisResult) error {
	_, err := r.db.Exec(`
		INSERT INTO analysis_results 
		(id, file_id, paragraphs, words, characters, plagiarism, word_cloud_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, result.ID, result.FileID, result.Paragraphs, result.Words,
		result.Characters, result.Plagiarism, result.WordCloudURL)
	return err
}

func (r *PostgresRepository) SaveWordCloud(id string, image io.Reader) error {
	data, err := io.ReadAll(image)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(`
		INSERT INTO word_clouds (id, image)
		VALUES ($1, $2)
	`, id, data)
	return err
}

func (r *PostgresRepository) GetWordCloud(id string) (io.Reader, error) {
	var data []byte
	err := r.db.QueryRow(`
		SELECT image FROM word_clouds WHERE id = $1
	`, id).Scan(&data)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func (r *PostgresRepository) GetAnalysisByFileID(fileID string) (*AnalysisResult, error) {
	var result AnalysisResult
	err := r.db.QueryRow(`
		SELECT id, file_id, paragraphs, words, characters, 
		       plagiarism, word_cloud_url
		FROM analysis_results
		WHERE file_id = $1
	`, fileID).Scan(
		&result.ID,
		&result.FileID,
		&result.Paragraphs,
		&result.Words,
		&result.Characters,
		&result.Plagiarism,
		&result.WordCloudURL,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &result, nil
}

func (r *PostgresRepository) GetFileContent(fileID string) (string, error) {
	// 1. Получаем URL File Storing Service из переменных окружения
	fileStoringURL := os.Getenv("FILE_STORING_SERVICE_URL")
	if fileStoringURL == "" {
		return "", fmt.Errorf("FILE_STORING_SERVICE_URL not set")
	}

	// 2. Делаем запрос к File Storing Service
	client := &http.Client{Timeout: 10 * time.Second}

	// Сначала получаем метаданные файла
	resp, err := client.Get(fmt.Sprintf("%s/files/%s", fileStoringURL, fileID))
	if err != nil {
		return "", fmt.Errorf("failed to get file metadata: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("file not found")
	}

	var metadata struct {
		Location string `json:"location"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return "", fmt.Errorf("failed to decode metadata: %v", err)
	}

	// Затем получаем содержимое файла
	resp, err = client.Get(fmt.Sprintf("%s/files/content/%s", fileStoringURL, metadata.Location))
	if err != nil {
		return "", fmt.Errorf("failed to get file content: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("file content not found")
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read content: %v", err)
	}

	return string(content), nil
}
