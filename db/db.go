package db

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// DB represents the database connection
type DB struct {
	*sql.DB
}

// MobileRecord represents a record in the database
type MobileRecord struct {
	ID        int64
	Mobile    string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewDB creates a new database connection
func NewDB() (*DB, error) {
	// Get database connection string from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	// Open database connection
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %v", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error connecting to the database: %v", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &DB{db}, nil
}

// InitDB initializes the database schema
func (db *DB) InitDB() error {
	// Create mobile_records table
	query := `
	CREATE TABLE IF NOT EXISTS mobile_records (
		id SERIAL PRIMARY KEY,
		mobile VARCHAR(10) UNIQUE NOT NULL,
		name VARCHAR(255) NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("error creating table: %v", err)
	}

	return nil
}

// SaveMobileRecord saves a mobile record to the database
func (db *DB) SaveMobileRecord(mobile, name string) error {
	query := `
	INSERT INTO mobile_records (mobile, name)
	VALUES ($1, $2)
	ON CONFLICT (mobile) 
	DO UPDATE SET 
		name = $2,
		updated_at = CURRENT_TIMESTAMP
	RETURNING id;`

	var id int64
	err := db.QueryRow(query, mobile, name).Scan(&id)
	if err != nil {
		return fmt.Errorf("error saving mobile record: %v", err)
	}

	return nil
}

// GetMobileRecord retrieves a mobile record from the database
func (db *DB) GetMobileRecord(mobile string) (*MobileRecord, error) {
	query := `
	SELECT id, mobile, name, created_at, updated_at
	FROM mobile_records
	WHERE mobile = $1;`

	record := &MobileRecord{}
	err := db.QueryRow(query, mobile).Scan(
		&record.ID,
		&record.Mobile,
		&record.Name,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error getting mobile record: %v", err)
	}

	return record, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}
