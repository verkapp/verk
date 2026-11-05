package src

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

// importAttachment copies a user-selected file into the application data
// directory. Images keep a preview, other documents are stored and opened
// with the system application.
func (s *store) importAttachment(taskID, sourcePath string) (Task, error) {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return Task{}, fmt.Errorf("fichier introuvable: %w", err)
	}
	if info.IsDir() {
		return Task{}, errors.New("ce fichier est un dossier")
	}
	if info.Size() > maxAttachmentBytes {
		return Task{}, errors.New("fichier trop lourd (25 Mo maximum)")
	}
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return Task{}, err
	}
	return s.importAttachmentBytes(taskID, filepath.Base(sourcePath), data)
}

// importAttachmentBytes stores a file received from the clipboard.
func (s *store) importAttachmentBytes(taskID, name string, data []byte) (Task, error) {
	if len(data) == 0 {
		return Task{}, errors.New("fichier vide")
	}
	if int64(len(data)) > maxAttachmentBytes {
		return Task{}, errors.New("fichier trop lourd (25 Mo maximum)")
	}

	head := data
	if len(head) > 512 {
		head = head[:512]
	}
	detected := http.DetectContentType(head)

	name, ext := attachmentName(name)
	if ext == "" {
		ext = extensionForMime(detected)
		if name == "" {
			name = "pièce-jointe"
		}
		name = clip(name+ext, 200)
	}
	if name == "" {
		name = clip("pièce-jointe"+ext, 200)
	}

	fileName := newID() + ext
	path := filepath.Join(s.imageDir, fileName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return Task{}, err
	}

	now := time.Now().UnixMilli()
	if _, err := s.db.Exec(`INSERT INTO images (id, task_id, name, file, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		newID(), taskID, name, fileName, now, now); err != nil {
		_ = os.Remove(path)
		return Task{}, err
	}
	return s.getTask(taskID)
}

// attachmentName strips any pasted path and extracts a safe extension.
func attachmentName(raw string) (string, string) {
	name := filepath.Base(strings.ReplaceAll(strings.TrimSpace(raw), "\\", "/"))
	if name == "." || name == "/" {
		name = ""
	}
	name = clip(name, 200)
	ext := safeAttachmentExtension(filepath.Ext(name))
	return name, ext
}

func safeAttachmentExtension(ext string) string {
	if len(ext) < 2 || len(ext) > 16 {
		return ""
	}
	for _, r := range ext[1:] {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return ""
		}
	}
	return strings.ToLower(ext)
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
	case "image/svg+xml":
		return ".svg"
	case "application/pdf":
		return ".pdf"
	case "application/zip":
		return ".zip"
	case "application/json":
		return ".json"
	case "application/msword":
		return ".doc"
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return ".docx"
	case "application/vnd.ms-excel":
		return ".xls"
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return ".xlsx"
	case "application/vnd.ms-powerpoint":
		return ".ppt"
	case "application/vnd.openxmlformats-officedocument.presentationml.presentation":
		return ".pptx"
	case "text/plain":
		return ".txt"
	case "text/csv":
		return ".csv"
	case "text/html":
		return ".html"
	case "text/markdown":
		return ".md"
	default:
		return ".bin"
	}
}

func (s *store) deleteAttachment(id string) (Task, error) {
	var taskID, file string
	if err := s.db.QueryRow(`SELECT task_id, file FROM images WHERE id = ?`, id).Scan(&taskID, &file); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Task{}, errNotFound
		}
		return Task{}, err
	}
	now := time.Now().UnixMilli()
	tx, err := s.db.Begin()
	if err != nil {
		return Task{}, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM images WHERE id = ?`, id); err != nil {
		return Task{}, err
	}
	if err := recordTombstone(tx, "image", id, taskID, now); err != nil {
		return Task{}, err
	}
	if err := tx.Commit(); err != nil {
		return Task{}, err
	}
	_ = os.Remove(filepath.Join(s.imageDir, file))
	return s.getTask(taskID)
}

func (s *store) attachmentFiles(taskID string) ([]string, error) {
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

func (s *store) attachmentPath(id string) (string, error) {
	var file string
	if err := s.db.QueryRow(`SELECT file FROM images WHERE id = ?`, id).Scan(&file); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errNotFound
		}
		return "", err
	}
	return filepath.Join(s.imageDir, file), nil
}

// attachmentMeta returns the MIME type and byte size of a stored attachment.
func (s *store) attachmentMeta(name, file string) (string, int64) {
	path := filepath.Join(s.imageDir, file)
	var size int64
	if info, err := os.Stat(path); err == nil {
		size = info.Size()
	}

	detected := detectFileMime(path)
	byExtension := mime.TypeByExtension(strings.ToLower(filepath.Ext(name)))
	if byExtension == "" {
		byExtension = mime.TypeByExtension(strings.ToLower(filepath.Ext(file)))
	}

	mimeType := byExtension
	// A real image header wins over a misleading or missing extension.
	if mimeType == "" || strings.HasPrefix(detected, "image/") {
		mimeType = detected
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	return mimeType, size
}

func detectFileMime(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return "application/octet-stream"
	}
	defer func() { _ = file.Close() }()
	head := make([]byte, 512)
	n, _ := file.Read(head)
	if n == 0 {
		return "application/octet-stream"
	}
	return http.DetectContentType(head[:n])
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
