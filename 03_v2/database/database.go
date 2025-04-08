package database

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"strings"

	_ "github.com/lib/pq" // Import the PostgreSQL driver
)

// DB toàn cục để truy cập cơ sở dữ liệu
var DB *sql.DB

// ConnectDB thiết lập kết nối đến cơ sở dữ liệu PostgreSQL
func ConnectDB(databaseURL string) error {
	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		return fmt.Errorf("failed to parse database URL: %w", err)
	}

	dbName := strings.TrimPrefix(parsedURL.Path, "/")
	serverURL := strings.Replace(databaseURL, "/"+dbName, "/", 1)

	if err := createDatabaseIfNotExists(serverURL, dbName); err != nil {
		log.Printf("Warning: Failed to create database '%s': %v. Ensure PostgreSQL server is running and the user has CREATE DATABASE privileges.", dbName, err)
		log.Println("Attempting to connect to the existing database.")
	}

	DB, err = sql.Open("postgres", databaseURL)
	if err != nil {
		return fmt.Errorf("failed to open database '%s': %w", dbName, err)
	}

	if err = DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database '%s': %w", dbName, err)
	}

	log.Printf("Connected to PostgreSQL database '%s'", dbName)

	// Tạo bảng nếu chưa tồn tại
	if err := createTablesIfNotExists(); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	return nil
}

// CloseDB đóng kết nối cơ sở dữ liệu
func CloseDB() error {
	if DB != nil {
		err := DB.Close()
		if err != nil {
			return fmt.Errorf("failed to close database: %w", err)
		}
		log.Println("Disconnected from PostgreSQL")
	}
	return nil
}

func createDatabaseIfNotExists(serverURL, dbName string) error {
	db, err := sql.Open("postgres", serverURL)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer db.Close()

	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT datname FROM pg_database WHERE datname = $1)", dbName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check database existence: %w", err)
	}

	if !exists {
		_, err = db.Exec("CREATE DATABASE " + dbName)
		if err != nil {
			return fmt.Errorf("failed to create database '%s': %w", dbName, err)
		}
		log.Printf("Database '%s' created successfully", dbName)
	} else {
		log.Printf("Database '%s' already exists", dbName)
	}

	return nil
}

func createTablesIfNotExists() error {
	// SQL lệnh tạo bảng dialog
	dialogTableSQL := `
	CREATE TABLE IF NOT EXISTS dialog (
		id BIGSERIAL PRIMARY KEY,
		lang VARCHAR(2) NOT NULL,
		content TEXT NOT NULL
	);`

	// SQL lệnh tạo bảng word
	wordTableSQL := `
	CREATE TABLE IF NOT EXISTS word (
		id BIGSERIAL PRIMARY KEY,
		lang VARCHAR(2) NOT NULL,
		content TEXT NOT NULL,
		translate TEXT NOT NULL
	);`

	// SQL lệnh tạo bảng word_dialog
	wordDialogTableSQL := `
	CREATE TABLE IF NOT EXISTS word_dialog (
		dialog_id BIGINT REFERENCES dialog(id) ON DELETE CASCADE,
		word_id BIGINT REFERENCES word(id) ON DELETE CASCADE,
		PRIMARY KEY (dialog_id, word_id)
	);`

	_, err := DB.Exec(dialogTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create dialog table: %w", err)
	}
	log.Println("Dialog table created or already exists")

	_, err = DB.Exec(wordTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create word table: %w", err)
	}
	log.Println("Word table created or already exists")

	_, err = DB.Exec(wordDialogTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create word_dialog table: %w", err)
	}
	log.Println("Word_dialog table created or already exists")

	return nil
}
