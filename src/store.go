package src

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

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
		_ = db.Close()
		return nil, fmt.Errorf("initialisation de la base: %w", err)
	}
	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migration de la base: %w", err)
	}
	return &store{db: db, dataDir: dir, imageDir: imageDir}, nil
}

// migrate upgrades databases created by earlier versions by adding the
// columns they lack.
func migrate(db *sql.DB) error {
	columns := map[string][]struct{ name, ddl string }{
		"tasks": {
			{"profile_id", "TEXT NOT NULL DEFAULT ''"},
			{"position", "INTEGER NOT NULL DEFAULT 0"},
			{"started_at", "INTEGER"},
			{"time_spent", "INTEGER NOT NULL DEFAULT 0"},
			{"updated_at", "INTEGER NOT NULL DEFAULT 0"},
		},
		"subtasks": {
			{"updated_at", "INTEGER NOT NULL DEFAULT 0"},
		},
		"images": {
			{"updated_at", "INTEGER NOT NULL DEFAULT 0"},
		},
	}
	positionAdded := false
	for table, wanted := range columns {
		have, err := tableColumns(db, table)
		if err != nil {
			return err
		}
		for _, c := range wanted {
			if have[c.name] {
				continue
			}
			if _, err := db.Exec(`ALTER TABLE ` + table + ` ADD COLUMN ` + c.name + ` ` + c.ddl); err != nil {
				return err
			}
			if table == "tasks" && c.name == "position" {
				positionAdded = true
			}
		}
	}
	// Existing rows get a sensible timestamp so their first sync does not
	// overwrite newer data coming from another machine.
	backfills := []string{
		`UPDATE tasks SET updated_at = COALESCE(completed_at, created_at, 0) WHERE updated_at = 0`,
		`UPDATE subtasks SET updated_at = COALESCE((SELECT created_at FROM tasks WHERE tasks.id = subtasks.task_id), 0) WHERE updated_at = 0`,
		`UPDATE images SET updated_at = created_at WHERE updated_at = 0`,
	}
	for _, query := range backfills {
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}
	// Order existing tasks exactly like the previous automatic sort did, so
	// enabling manual reordering does not shuffle the user's list.
	if positionAdded {
		if err := backfillTaskPositions(db); err != nil {
			return err
		}
	}
	return ensureProfiles(db)
}

// ensureProfiles seeds the two starter profiles and makes sure every task
// belongs to one. It is idempotent and runs on every startup.
func ensureProfiles(db *sql.DB) error {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM profiles`).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		now := time.Now().UnixMilli()
		starters := []Profile{
			{ID: newID(), Name: "Perso", Color: profileColors[0], Position: 0, CreatedAt: now},
			{ID: newID(), Name: "Travail", Color: profileColors[1], Position: 1, CreatedAt: now},
		}
		for _, p := range starters {
			if _, err := db.Exec(`INSERT INTO profiles (id, name, color, position, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
				p.ID, p.Name, p.Color, p.Position, p.CreatedAt, now); err != nil {
				return err
			}
		}
	}
	var fallback string
	if err := db.QueryRow(`SELECT id FROM profiles ORDER BY position ASC, created_at ASC, id ASC LIMIT 1`).Scan(&fallback); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	_, err := db.Exec(`UPDATE tasks SET profile_id = ? WHERE profile_id = '' OR profile_id NOT IN (SELECT id FROM profiles)`, fallback)
	return err
}

// backfillTaskPositions assigns a position to every task using the legacy
// automatic order. sortTasks relies on due dates and creation time when all
// positions are equal, which is exactly what we need here.
func backfillTaskPositions(db *sql.DB) error {
	rows, err := db.Query(`SELECT id, done, created_at, due_at, completed_at FROM tasks`)
	if err != nil {
		return err
	}
	tasks := []Task{}
	for rows.Next() {
		var t Task
		var done int
		if err := rows.Scan(&t.ID, &done, &t.CreatedAt, &t.DueAt, &t.CompletedAt); err != nil {
			_ = rows.Close()
			return err
		}
		t.Done = done == 1
		tasks = append(tasks, t)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if err := rows.Err(); err != nil {
		return err
	}
	sortTasks(tasks)
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for index, task := range tasks {
		if _, err := tx.Exec(`UPDATE tasks SET position = ? WHERE id = ?`, index, task.ID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func tableColumns(db *sql.DB, table string) (map[string]bool, error) {
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	have := map[string]bool{}
	for rows.Next() {
		var cid, notNull, pk int
		var name, typ string
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
			return nil, err
		}
		have[name] = true
	}
	return have, rows.Err()
}

func (s *store) close() error { return s.db.Close() }
