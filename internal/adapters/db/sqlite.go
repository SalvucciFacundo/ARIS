package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// SQLiteDB wraps the database connection for ARIS.
type SQLiteDB struct {
	db *sql.DB
}

// NewDefaultSQLiteDB initializes the SQLite database at ~/.aris/aris.db.
func NewDefaultSQLiteDB() (*SQLiteDB, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve user home dir: %w", err)
	}
	dbDir := filepath.Join(home, ".aris")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("create aris config dir: %w", err)
	}
	return NewSQLiteDB(filepath.Join(dbDir, "aris.db"))
}

// NewSQLiteDB creates or connects to a SQLite database at the specified path.
func NewSQLiteDB(dbPath string) (*SQLiteDB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db at %s: %w", dbPath, err)
	}

	// Optimize for single-process agent usage
	if _, err := db.Exec("PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set pragmas: %w", err)
	}

	s := &SQLiteDB{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return s, nil
}

// DB returns the underlying database handle.
func (s *SQLiteDB) DB() *sql.DB {
	return s.db
}

// Close closes the underlying database connection.
func (s *SQLiteDB) Close() error {
	return s.db.Close()
}

func (s *SQLiteDB) migrate() error {
	schema := `
	-- Knowledge Graph Facts Table (GAIA 3-Scope Model)
	CREATE TABLE IF NOT EXISTS knowledge_facts (
		id TEXT PRIMARY KEY,
		topic TEXT NOT NULL,
		concept TEXT NOT NULL,
		fact TEXT NOT NULL,
		source_agent TEXT NOT NULL,
		labels TEXT NOT NULL DEFAULT '[]',
		project TEXT NOT NULL DEFAULT '',
		scope TEXT NOT NULL DEFAULT 'user',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_knowledge_topic ON knowledge_facts(topic);
	CREATE INDEX IF NOT EXISTS idx_knowledge_scope ON knowledge_facts(scope);
	CREATE INDEX IF NOT EXISTS idx_knowledge_created ON knowledge_facts(created_at);

	-- FTS5 Full-Text Search Virtual Table
	CREATE VIRTUAL TABLE IF NOT EXISTS knowledge_facts_fts USING fts5(
		topic,
		concept,
		fact,
		labels,
		content='knowledge_facts',
		content_rowid='rowid'
	);

	-- Generation History Table
	CREATE TABLE IF NOT EXISTS generations (
		id TEXT PRIMARY KEY,
		prompt_raw TEXT NOT NULL,
		prompt_enhanced TEXT NOT NULL,
		negative_prompt TEXT NOT NULL DEFAULT '',
		backend TEXT NOT NULL,
		model TEXT NOT NULL,
		width INTEGER NOT NULL,
		height INTEGER NOT NULL,
		steps INTEGER NOT NULL DEFAULT 20,
		cfg_scale REAL NOT NULL DEFAULT 7.0,
		seed INTEGER NOT NULL DEFAULT 0,
		image_path TEXT NOT NULL,
		thumb_path TEXT NOT NULL DEFAULT '',
		duration_ms INTEGER NOT NULL DEFAULT 0,
		rating INTEGER NOT NULL DEFAULT 0,
		feedback TEXT NOT NULL DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_generations_created ON generations(created_at DESC);
	`
	_, err := s.db.Exec(schema)
	return err
}
