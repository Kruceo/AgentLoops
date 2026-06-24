package db

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

// Open opens a SQLite database connection.
func Open(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("enable WAL: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	return db, nil
}

// Migrate runs all pending database migrations.
func Migrate(db *sql.DB) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			task_name TEXT NOT NULL,
			init_message TEXT NOT NULL,
			agent_runner TEXT NOT NULL,
			agent_model TEXT NOT NULL DEFAULT '',
			agent_mode TEXT NOT NULL DEFAULT '',
			work_dir TEXT NOT NULL DEFAULT '',
			enabled INTEGER NOT NULL DEFAULT 1,
			cron_expr TEXT NOT NULL DEFAULT '',
			interval_seconds INTEGER NOT NULL DEFAULT 60,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,

		`CREATE TABLE IF NOT EXISTS runs (
			id TEXT PRIMARY KEY,
			task_id TEXT NOT NULL,
			output TEXT NOT NULL DEFAULT '',
			has_error INTEGER NOT NULL DEFAULT 0,
			started_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			finished_at DATETIME,
			FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
		);`,
	}

	for i, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return fmt.Errorf("migration %d: %w", i, err)
		}
	}

	log.Println("database migrations complete")
	return nil
}

// NewUUID generates a UUID v4-like string using crypto/rand.
// Format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
func NewUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand.Read failed: %v", err))
	}

	// Set version 4
	b[6] = (b[6] & 0x0f) | 0x40
	// Set variant bits
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
