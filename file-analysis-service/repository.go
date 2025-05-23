package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type SimilarFile struct {
	FileID     string  `json:"file_id"`
	Name       string  `json:"name"`
	Similarity float64 `json:"similarity"`
}

type AnalysisResult struct {
	ID             string        `json:"id"`
	FileID         string        `json:"file_id"`
	Paragraphs     int           `json:"paragraphs"`
	Words          int           `json:"words"`
	Characters     int           `json:"characters"`
	PlagiarismRate float64       `json:"plagiarism_rate"`
	SimilarFiles   []SimilarFile `json:"similar_files"`
	WordCloudURL   string        `json:"word_cloud_url"`
}

type FileMetadata struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Hash     string `json:"hash"`
	Location string `json:"location"`
}

type Repository interface {
	GetFileContent(fileID string) (string, error)
	FindSimilarFiles(content, currentFileID string) ([]SimilarFile, error)
	SaveAnalysis(result AnalysisResult) error
	GetAnalysisByFileID(fileID string) (*AnalysisResult, error)
	GetFileMetadata(fileID string) (*FileMetadata, error)
	SaveWordCloud(id string, image []byte) error
	GetWordCloud(id string) ([]byte, error)
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

	_, err = db.Exec(`
    CREATE TABLE IF NOT EXISTS analysis_results (
        id TEXT PRIMARY KEY,
        file_id TEXT NOT NULL UNIQUE,
        paragraphs INTEGER NOT NULL,
        words INTEGER NOT NULL,
        characters INTEGER NOT NULL,
        similar_files JSONB NOT NULL,
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

func (r *PostgresRepository) GetFileMetadata(fileID string) (*FileMetadata, error) {
	// Используем File Storing Service для получения метаданных
	fileStoringURL := os.Getenv("FILE_STORING_SERVICE_URL")
	if fileStoringURL == "" {
		return nil, fmt.Errorf("FILE_STORING_SERVICE_URL not set")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(fmt.Sprintf("%s/files/%s", fileStoringURL, fileID))
	if err != nil {
		return nil, fmt.Errorf("failed to get file metadata: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("file not found")
	}

	var metadata FileMetadata
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %v", err)
	}

	return &metadata, nil
}

func (r *PostgresRepository) FindSimilarFiles(content, currentFileID string) ([]SimilarFile, error) {
	// Нормализуем текст для сравнения
	normalizedContent := normalizeText(content)

	rows, err := r.db.Query(`
        WITH normalized AS (
            SELECT 
                file_metadata.id, 
                file_metadata.location, 
                $2 AS norm_content
            FROM file_content
            JOIN file_metadata ON file_content.location = file_metadata.location
            WHERE file_metadata.id != $1
            AND file_content.content IS NOT NULL
        )
        SELECT 
            id, 
            similarity($2, norm_content) AS similarity
        FROM normalized
        WHERE similarity($2, norm_content) > 0.3
        ORDER BY similarity DESC
        LIMIT 5
    `, currentFileID, normalizedContent)

	if err != nil {
		return nil, fmt.Errorf("query error: %v", err)
	}
	defer rows.Close()

	var results []SimilarFile
	for rows.Next() {
		var sf SimilarFile
		if err := rows.Scan(&sf.FileID, &sf.Similarity); err != nil {
			return nil, fmt.Errorf("scan error: %v", err)
		}
		results = append(results, sf)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %v", err)
	}

	return results, nil
}

func normalizeText(text string) string {
	text = strings.ToLower(text)

	reg := regexp.MustCompile(`[^\w\s]`)
	text = reg.ReplaceAllString(text, " ")

	text = strings.Join(strings.Fields(text), " ")

	return text
}
func (r *PostgresRepository) SaveAnalysis(result AnalysisResult) error {
	similarFilesJSON, err := json.Marshal(result.SimilarFiles)
	if err != nil {
		return fmt.Errorf("failed to marshal similar files: %v", err)
	}

	_, err = r.db.Exec(`
        INSERT INTO analysis_results 
        (id, file_id, paragraphs, words, characters, similar_files, word_cloud_url)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `, result.ID, result.FileID, result.Paragraphs, result.Words,
		result.Characters, similarFilesJSON, result.WordCloudURL)
	return err
}

func (r *PostgresRepository) SaveWordCloud(id string, image []byte) error {
	_, err := r.db.Exec(
		"INSERT INTO word_clouds (id, image) VALUES ($1, $2)",
		id, image,
	)
	return err
}

func (r *PostgresRepository) GetWordCloud(id string) ([]byte, error) {
	var image []byte
	err := r.db.QueryRow(
		"SELECT image FROM word_clouds WHERE id = $1",
		id,
	).Scan(&image)
	return image, err
}

func (r *PostgresRepository) GetAnalysisByFileID(fileID string) (*AnalysisResult, error) {
	var (
		result           AnalysisResult
		similarFilesJSON []byte
	)

	err := r.db.QueryRow(`
        SELECT id, file_id, paragraphs, words, characters, 
               similar_files, word_cloud_url
        FROM analysis_results
        WHERE file_id = $1
    `, fileID).Scan(
		&result.ID,
		&result.FileID,
		&result.Paragraphs,
		&result.Words,
		&result.Characters,
		&similarFilesJSON,
		&result.WordCloudURL,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(similarFilesJSON, &result.SimilarFiles); err != nil {
		return nil, fmt.Errorf("failed to unmarshal similar files: %v", err)
	}

	return &result, nil
}

func (r *PostgresRepository) GetFileContent(fileID string) (string, error) {
	fileStoringURL := os.Getenv("FILE_STORING_SERVICE_URL")
	if fileStoringURL == "" {
		return "", fmt.Errorf("FILE_STORING_SERVICE_URL not set")
	}

	client := &http.Client{Timeout: 10 * time.Second}

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
