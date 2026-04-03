package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps an *sql.DB with helper methods for the agent manager.
type DB struct {
	*sql.DB
}

// PaginatedResult is a generic paginated response.
type PaginatedResult struct {
	Data       []map[string]interface{} `json:"data"`
	NextCursor interface{}              `json:"next_cursor"`
	HasMore    bool                     `json:"has_more"`
}

// Now returns the current time in SQLite-compatible format.
func Now() string {
	return time.Now().UTC().Format("2006-01-02 15:04:05")
}

// Open creates or opens the SQLite database, runs migrations, and returns a DB.
func Open(dbPath string) (*DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	sqlDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// Enable WAL and foreign keys
	for _, pragma := range []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA foreign_keys = ON",
		"PRAGMA busy_timeout = 5000",
	} {
		if _, err := sqlDB.Exec(pragma); err != nil {
			return nil, fmt.Errorf("pragma %s: %w", pragma, err)
		}
	}

	d := &DB{sqlDB}
	if err := d.createTables(); err != nil {
		return nil, err
	}
	if err := d.runMigrations(); err != nil {
		return nil, err
	}
	if err := d.createTriggers(); err != nil {
		return nil, err
	}
	d.backfillComputedFields()
	d.migrateFilesToDisk()

	return d, nil
}

// WALCheckpoint runs a passive WAL checkpoint.
func (d *DB) WALCheckpoint() error {
	_, err := d.Exec("PRAGMA wal_checkpoint(PASSIVE)")
	return err
}

// WALCheckpointTruncate runs a truncating WAL checkpoint for shutdown.
func (d *DB) WALCheckpointTruncate() error {
	_, err := d.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	return err
}

// ---- Helpers ----

// scanRows converts sql.Rows into a slice of maps.
func scanRows(rows *sql.Rows) ([]map[string]interface{}, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var result []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make(map[string]interface{})
		for i, col := range cols {
			v := values[i]
			// Convert []byte to string for JSON serialization
			if b, ok := v.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = v
			}
		}
		result = append(result, row)
	}
	if result == nil {
		result = []map[string]interface{}{}
	}
	return result, nil
}

// scanRow converts a single sql.Row result into a map.
func (d *DB) scanRowMap(query string, args ...interface{}) (map[string]interface{}, error) {
	rows, err := d.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	return results[0], nil
}
