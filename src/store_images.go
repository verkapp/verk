package src

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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
	defer func() { _ = file.Close() }()

	head := make([]byte, 512)
	n, _ := file.Read(head)
	if n == 0 {
		return Task{}, errors.New("fichier vide")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return Task{}, err
	}
	detected := http.DetectContentType(head[:n])
	if !strings.HasPrefix(detected, "image/") {
		return Task{}, errors.New("ce fichier n'est pas une image")
	}

	ext := strings.ToLower(filepath.Ext(sourcePath))
	if ext == "" {
		ext = extensionForMime(detected)
	}
	fileName := newID() + ext
	dst, err := os.Create(filepath.Join(s.imageDir, fileName))
	if err != nil {
		return Task{}, err
	}
	if _, err := io.Copy(dst, file); err != nil {
		_ = dst.Close()
		_ = os.Remove(filepath.Join(s.imageDir, fileName))
		return Task{}, err
	}
	if err := dst.Close(); err != nil {
		_ = os.Remove(filepath.Join(s.imageDir, fileName))
		return Task{}, err
	}

	if _, err := s.db.Exec(`INSERT INTO images (id, task_id, name, file, created_at) VALUES (?, ?, ?, ?, ?)`,
		newID(), taskID, clip(filepath.Base(sourcePath), 200), fileName, time.Now().UnixMilli()); err != nil {
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
