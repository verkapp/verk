package src

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var errNotFound = errors.New("élément introuvable")

const maxImageBytes = 15 << 20 // 15 MiB

type store struct {
	db       *sql.DB
	dataDir  string
	imageDir string
}

// dataDirectory resolves the per-user application directory and creates it.
func dataDirectory() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "todo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func openStore() (*store, error) {
	dir, err := dataDirectory()
	if err != nil {
		return nil, fmt.Errorf("dossier de données: %w", err)
	}
	return openStoreAt(dir)
}

// openStoreAt opens (and migrates) the database in a given directory. It is
// separated from openStore so tests can use a temporary directory.
func openStoreAt(dir string) (*store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("dossier de données: %w", err)
	}
	imageDir := filepath.Join(dir, "images")
	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		return nil, fmt.Errorf("dossier images: %w", err)
	}
	dbPath := filepath.Join(dir, "todo.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("ouverture de la base: %w", err)
	}
	// A single connection keeps PRAGMAs (foreign keys) predictable for a
	// desktop workload while avoiding lock contention.
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("initialisation de la base: %w", err)
	}
	return &store{db: db, dataDir: dir, imageDir: imageDir}, nil
}

func (s *store) close() error { return s.db.Close() }
