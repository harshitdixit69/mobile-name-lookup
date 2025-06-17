package db

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
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
	connectionString := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", "upnbsxg4yg4es1ic", "jWLiq8tKZQPtyCoSTGyO", "bakggowhgkephmh0ugod-mysql.services.clever-cloud.com", 3306, "bakggowhgkephmh0ugod")
	// Open database connection
	db, err := sql.Open("mysql", connectionString)
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
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		mobile VARCHAR(10) UNIQUE NOT NULL,
		name VARCHAR(255) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
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
	VALUES (?, ?)
	ON DUPLICATE KEY UPDATE 
		name = VALUES(name),
		updated_at = CURRENT_TIMESTAMP;`

	_, err := db.Exec(query, mobile, name)
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
	WHERE mobile = ?;`

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

// TestConnection tests the database connection
func (db *DB) TestConnection() error {
	// Try to ping the database
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	// Try a simple query
	var result int
	err := db.QueryRow("SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("failed to execute test query: %v", err)
	}

	if result != 1 {
		return fmt.Errorf("unexpected test query result: %d", result)
	}

	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}
