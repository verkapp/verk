package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
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
	if _, err := db.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA foreign_keys = ON;
		PRAGMA busy_timeout = 5000;
		CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			done INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL,
			due_at INTEGER,
			completed_at INTEGER
		);
		CREATE TABLE IF NOT EXISTS subtasks (
			id TEXT PRIMARY KEY,
			task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			done INTEGER NOT NULL DEFAULT 0,
			position INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE IF NOT EXISTS images (
			id TEXT PRIMARY KEY,
			task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			file TEXT NOT NULL,
			created_at INTEGER NOT NULL
		);
		CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_subtasks_task ON subtasks(task_id);
		CREATE INDEX IF NOT EXISTS idx_images_task ON images(task_id);
	`); err != nil {
		db.Close()
		return nil, fmt.Errorf("initialisation de la base: %w", err)
	}
	return &store{db: db, dataDir: dir, imageDir: imageDir}, nil
}

func (s *store) close() error { return s.db.Close() }

func (s *store) listTasks() ([]Task, error) {
	rows, err := s.db.Query(`SELECT id, title, description, done, created_at, due_at, completed_at FROM tasks`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := []Task{}
	index := map[string]int{}
	for rows.Next() {
		var t Task
		var done int
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &done, &t.CreatedAt, &t.DueAt, &t.CompletedAt); err != nil {
			return nil, err
		}
		t.Done = done == 1
		t.Subtasks = []Subtask{}
		t.Images = []TaskImage{}
		index[t.ID] = len(tasks)
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	subRows, err := s.db.Query(`SELECT id, task_id, title, done, position FROM subtasks ORDER BY position ASC, rowid ASC`)
	if err != nil {
		return nil, err
	}
	defer subRows.Close()
	for subRows.Next() {
		var sub Subtask
		var taskID string
		var done int
		if err := subRows.Scan(&sub.ID, &taskID, &sub.Title, &done, &sub.Position); err != nil {
			return nil, err
		}
		sub.Done = done == 1
		if i, ok := index[taskID]; ok {
			tasks[i].Subtasks = append(tasks[i].Subtasks, sub)
		}
	}

	imgRows, err := s.db.Query(`SELECT id, task_id, name, file, created_at FROM images ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer imgRows.Close()
	for imgRows.Next() {
		var id, taskID, name, file string
		var createdAt int64
		if err := imgRows.Scan(&id, &taskID, &name, &file, &createdAt); err != nil {
			return nil, err
		}
		if i, ok := index[taskID]; ok {
			tasks[i].Images = append(tasks[i].Images, TaskImage{
				ID:        id,
				Name:      name,
				DataURL:   s.imageDataURL(file),
				CreatedAt: createdAt,
			})
		}
	}

	sortTasks(tasks)
	return tasks, nil
}

func sortTasks(tasks []Task) {
	sort.SliceStable(tasks, func(i, j int) bool {
		a, b := tasks[i], tasks[j]
		if a.Done != b.Done {
			return !a.Done
		}
		if a.Done {
			var ac, bc int64
			if a.CompletedAt != nil {
				ac = *a.CompletedAt
			}
			if b.CompletedAt != nil {
				bc = *b.CompletedAt
			}
			return ac > bc
		}
		if (a.DueAt == nil) != (b.DueAt == nil) {
			return a.DueAt != nil
		}
		if a.DueAt != nil && *a.DueAt != *b.DueAt {
			return *a.DueAt < *b.DueAt
		}
		return a.CreatedAt > b.CreatedAt
	})
}

func (s *store) getTask(id string) (Task, error) {
	all, err := s.listTasks()
	if err != nil {
		return Task{}, err
	}
	for _, t := range all {
		if t.ID == id {
			return t, nil
		}
	}
	return Task{}, errNotFound
}

func (s *store) createTask(input TaskInput) (Task, error) {
	title := cleanTitle(input.Title)
	if title == "" {
		return Task{}, errors.New("le titre est obligatoire")
	}
	id := newID()
	now := time.Now().UnixMilli()
	_, err := s.db.Exec(`INSERT INTO tasks (id, title, description, done, created_at, due_at) VALUES (?, ?, ?, 0, ?, ?)`,
		id, title, strings.TrimSpace(input.Description), now, normalizeDue(input.DueAt))
	if err != nil {
		return Task{}, err
	}
	return s.getTask(id)
}

func (s *store) updateTask(id string, input TaskInput) (Task, error) {
	title := cleanTitle(input.Title)
	if title == "" {
		return Task{}, errors.New("le titre est obligatoire")
	}
	res, err := s.db.Exec(`UPDATE tasks SET title = ?, description = ?, due_at = ? WHERE id = ?`,
		title, strings.TrimSpace(input.Description), normalizeDue(input.DueAt), id)
	if err != nil {
		return Task{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Task{}, errNotFound
	}
	return s.getTask(id)
}

func (s *store) setDone(id string, done bool) (Task, error) {
	var completedAt *int64
	if done {
		now := time.Now().UnixMilli()
		completedAt = &now
	}
	res, err := s.db.Exec(`UPDATE tasks SET done = ?, completed_at = ? WHERE id = ?`, boolInt(done), completedAt, id)
	if err != nil {
		return Task{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Task{}, errNotFound
	}
	return s.getTask(id)
}

func (s *store) deleteTask(id string) error {
	files, err := s.imageFiles(id)
	if err != nil {
		return err
	}
	res, err := s.db.Exec(`DELETE FROM tasks WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errNotFound
	}
	for _, f := range files {
		_ = os.Remove(filepath.Join(s.imageDir, f))
	}
	return nil
}

func (s *store) addSubtask(taskID, title string) (Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Task{}, errors.New("la sous-tâche est vide")
	}
	var pos int
	_ = s.db.QueryRow(`SELECT COALESCE(MAX(position), -1) + 1 FROM subtasks WHERE task_id = ?`, taskID).Scan(&pos)
	if _, err := s.db.Exec(`INSERT INTO subtasks (id, task_id, title, done, position) VALUES (?, ?, ?, 0, ?)`,
		newID(), taskID, clip(title, 240), pos); err != nil {
		return Task{}, err
	}
	return s.getTask(taskID)
}

func (s *store) addSubtasks(taskID string, titles []string) (Task, error) {
	for _, title := range titles {
		if strings.TrimSpace(title) == "" {
			continue
		}
		if _, err := s.addSubtask(taskID, title); err != nil {
			return Task{}, err
		}
	}
	return s.getTask(taskID)
}

func (s *store) updateSubtask(id, title string) (Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Task{}, errors.New("la sous-tâche est vide")
	}
	var taskID string
	if err := s.db.QueryRow(`SELECT task_id FROM subtasks WHERE id = ?`, id).Scan(&taskID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Task{}, errNotFound
		}
		return Task{}, err
	}
	if _, err := s.db.Exec(`UPDATE subtasks SET title = ? WHERE id = ?`, clip(title, 240), id); err != nil {
		return Task{}, err
	}
	return s.getTask(taskID)
}

func (s *store) setSubtaskDone(id string, done bool) (Task, error) {
	var taskID string
	if err := s.db.QueryRow(`SELECT task_id FROM subtasks WHERE id = ?`, id).Scan(&taskID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Task{}, errNotFound
		}
		return Task{}, err
	}
	if _, err := s.db.Exec(`UPDATE subtasks SET done = ? WHERE id = ?`, boolInt(done), id); err != nil {
		return Task{}, err
	}
	return s.getTask(taskID)
}

func (s *store) deleteSubtask(id string) (Task, error) {
	var taskID string
	if err := s.db.QueryRow(`SELECT task_id FROM subtasks WHERE id = ?`, id).Scan(&taskID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Task{}, errNotFound
		}
		return Task{}, err
	}
	if _, err := s.db.Exec(`DELETE FROM subtasks WHERE id = ?`, id); err != nil {
		return Task{}, err
	}
	return s.getTask(taskID)
}

// importImage copies a user-selected file into the app data directory.
func (s *store) importImage(taskID, sourcePath string) (Task, error) {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return Task{}, fmt.Errorf("fichier introuvable: %w", err)
	}
	if info.IsDir() {
		return Task{}, errors.New("ce fichier est un dossier")
	}
	if info.Size() > maxImageBytes {
		return Task{}, errors.New("image trop lourde (15 Mo maximum)")
	}
	file, err := os.Open(sourcePath)
	if err != nil {
		return Task{}, err
	}
	defer file.Close()
	head := make([]byte, 512)
	n, _ := file.Read(head)
	if n == 0 {
		return Task{}, errors.New("fichier vide")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return Task{}, err
	}
	if !strings.HasPrefix(http.DetectContentType(head[:n]), "image/") {
		return Task{}, errors.New("ce fichier n'est pas une image")
	}
	detected := http.DetectContentType(head[:n])
	ext := strings.ToLower(filepath.Ext(sourcePath))
	if ext == "" {
		ext = extensionForMime(detected)
	}
	name := filepath.Base(sourcePath)
	fileName := newID() + ext
	dst, err := os.Create(filepath.Join(s.imageDir, fileName))
	if err != nil {
		return Task{}, err
	}
	if _, err := io.Copy(dst, file); err != nil {
		dst.Close()
		_ = os.Remove(filepath.Join(s.imageDir, fileName))
		return Task{}, err
	}
	dst.Close()
	if _, err := s.db.Exec(`INSERT INTO images (id, task_id, name, file, created_at) VALUES (?, ?, ?, ?, ?)`,
		newID(), taskID, clip(name, 200), fileName, time.Now().UnixMilli()); err != nil {
		_ = os.Remove(filepath.Join(s.imageDir, fileName))
		return Task{}, err
	}
	return s.getTask(taskID)
}

func extensionForMime(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/bmp":
		return ".bmp"
	default:
		return ".png"
	}
}

func (s *store) deleteImage(id string) (Task, error) {
	var taskID, file string
	if err := s.db.QueryRow(`SELECT task_id, file FROM images WHERE id = ?`, id).Scan(&taskID, &file); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Task{}, errNotFound
		}
		return Task{}, err
	}
	if _, err := s.db.Exec(`DELETE FROM images WHERE id = ?`, id); err != nil {
		return Task{}, err
	}
	_ = os.Remove(filepath.Join(s.imageDir, file))
	return s.getTask(taskID)
}

func (s *store) imageFiles(taskID string) ([]string, error) {
	rows, err := s.db.Query(`SELECT file FROM images WHERE task_id = ?`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var files []string
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

func (s *store) imageDataURL(file string) string {
	data, err := os.ReadFile(filepath.Join(s.imageDir, file))
	if err != nil {
		return ""
	}
	mimeType := mime.TypeByExtension(strings.ToLower(filepath.Ext(file)))
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
	}
	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data)
}

func (s *store) getSettings() (Settings, error) {
	out := Settings{Accent: "lavande"}
	rows, err := s.db.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return out, err
		}
		switch k {
		case "accent":
			if v != "" {
				out.Accent = v
			}
		case "coach_on_start":
			out.CoachOnStart = v == "1"
		case "claude_path":
			out.ClaudePath = v
		}
	}
	return out, rows.Err()
}

func (s *store) saveSettings(in Settings) error {
	if in.Accent == "" {
		in.Accent = "lavande"
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	values := map[string]string{
		"accent":         in.Accent,
		"coach_on_start": boolIntString(in.CoachOnStart),
		"claude_path":    strings.TrimSpace(in.ClaudePath),
	}
	for k, v := range values {
		if _, err := tx.Exec(`INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, k, v); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func cleanTitle(s string) string {
	return clip(strings.Join(strings.Fields(s), " "), 240)
}

func clip(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}

func normalizeDue(value *int64) *int64 {
	if value == nil || *value <= 0 {
		return nil
	}
	return value
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func boolIntString(b bool) string {
	if b {
		return "1"
	}
	return "0"
}
