package database

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// DB wraps sql.DB with additional methods
type DB struct {
	*sql.DB
}

// New creates a new database connection
func New(path string) (*DB, error) {
	// Use DSN with embedded pragmas to ensure all connections get proper settings
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(15000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("database open failed: %w", err)
	}

	// Serialize all database access to eliminate SQLITE_BUSY errors
	db.SetMaxOpenConns(1) // Only one connection at a time
	db.SetMaxIdleConns(1) // Keep connection alive for reuse

	return &DB{db}, nil
}

// InitSchema creates all necessary tables
func (db *DB) InitSchema() error {
	schema := `
    CREATE TABLE IF NOT EXISTS ping_results (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        timestamp DATETIME NOT NULL,
        target TEXT NOT NULL,
        success BOOLEAN NOT NULL,
        rtt_ms REAL,
        error_message TEXT,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );

    CREATE INDEX IF NOT EXISTS idx_timestamp ON ping_results(timestamp);
    CREATE INDEX IF NOT EXISTS idx_target_timestamp ON ping_results(target, timestamp);

    CREATE TABLE IF NOT EXISTS outages (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        target TEXT NOT NULL,
        start_time DATETIME NOT NULL,
        end_time DATETIME,
        duration_seconds INTEGER,
        checks_failed INTEGER
    );

    CREATE TABLE IF NOT EXISTS hourly_stats (
        hour DATETIME NOT NULL,
        target TEXT NOT NULL,
        total_pings INTEGER,
        successful_pings INTEGER,
        avg_rtt_ms REAL,
        max_rtt_ms REAL,
        min_rtt_ms REAL,
        p95_rtt_ms REAL,
        p99_rtt_ms REAL,
        packet_loss_percent REAL,
        PRIMARY KEY (hour, target)
    );

    -- New table for heatmap data (aggregated by hour of day)
    CREATE TABLE IF NOT EXISTS hourly_patterns (
        date DATE NOT NULL,
        hour INTEGER NOT NULL, -- 0-23
        target TEXT NOT NULL,
        total_pings INTEGER,
        failed_pings INTEGER,
        avg_rtt_ms REAL,
        max_rtt_ms REAL,
        failure_rate REAL,
        PRIMARY KEY (date, hour, target)
    );

    CREATE INDEX IF NOT EXISTS idx_hourly_patterns ON hourly_patterns(hour, target);
    CREATE INDEX IF NOT EXISTS idx_hourly_patterns_date ON hourly_patterns(date);
    CREATE INDEX IF NOT EXISTS idx_hourly_patterns_hour_date ON hourly_patterns(hour, date);
    CREATE INDEX IF NOT EXISTS idx_ping_success_timestamp ON ping_results(success, timestamp);
    CREATE INDEX IF NOT EXISTS idx_outages_start_time ON outages(start_time);
    `

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("schema creation failed: %w", err)
	}

	return nil
}
