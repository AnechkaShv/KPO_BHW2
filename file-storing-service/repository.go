package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

type FileMetadata struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Hash     string `json:"hash"`
	Location string `json:"location"`
}

type FileContent struct {
	Location string `json:"location"`
	Content  string `json:"content"`
}

type Repository interface {
	GetFileByHash(hash string) (*FileMetadata, error)
	SaveFile(metadata FileMetadata, content string) (string, error)
	GetFile(id string) (*FileMetadata, error)
	GetFileContent(location string) (string, error)
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
		panic(err)
	}

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	// Initialize tables
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS file_metadata (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			hash TEXT NOT NULL UNIQUE,
			location TEXT NOT NULL UNIQUE
		)
	`)
	if err != nil {
		panic(err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS file_content (
			location TEXT PRIMARY KEY,
			content TEXT NOT NULL
		)
	`)
	if err != nil {
		panic(err)
	}

	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) GetFileByHash(hash string) (*FileMetadata, error) {
	var file FileMetadata
	err := r.db.QueryRow(
		"SELECT id, name, hash, location FROM file_metadata WHERE hash = $1",
		hash,
	).Scan(&file.ID, &file.Name, &file.Hash, &file.Location)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &file, nil
}

func (r *PostgresRepository) SaveFile(metadata FileMetadata, content string) (string, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return "", err
	}

	_, err = tx.Exec(
		"INSERT INTO file_metadata (id, name, hash, location) VALUES ($1, $2, $3, $4)",
		metadata.ID, metadata.Name, metadata.Hash, metadata.Location,
	)
	if err != nil {
		tx.Rollback()
		return "", err
	}

	_, err = tx.Exec(
		"INSERT INTO file_content (location, content) VALUES ($1, $2)",
		metadata.Location, content,
	)
	if err != nil {
		tx.Rollback()
		return "", err
	}

	err = tx.Commit()
	if err != nil {
		return "", err
	}

	return metadata.ID, nil
}

func (r *PostgresRepository) GetFile(id string) (*FileMetadata, error) {
	var file FileMetadata
	err := r.db.QueryRow(
		"SELECT id, name, hash, location FROM file_metadata WHERE id = $1",
		id,
	).Scan(&file.ID, &file.Name, &file.Hash, &file.Location)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &file, nil
}

func (r *PostgresRepository) GetFileContent(location string) (string, error) {
	var content string
	err := r.db.QueryRow(
		"SELECT content FROM file_content WHERE location = $1",
		location,
	).Scan(&content)

	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}

	return content, nil
}
