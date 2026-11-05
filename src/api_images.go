package src

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// AddAttachment opens the native picker, copies the chosen files into the
// application data directory and returns the refreshed task. Cancelling is not
// an error: the task is simply returned unchanged.
func (a *App) AddAttachment(taskID string) (Task, error) {
	if err := a.ready(); err != nil {
		return Task{}, err
	}
	if _, err := a.store.getTask(taskID); err != nil {
		return Task{}, err
	}
	paths, err := wruntime.OpenMultipleFilesDialog(a.ctx, wruntime.OpenDialogOptions{
		Title: "Choisir des pièces jointes",
		Filters: []wruntime.FileFilter{
			{DisplayName: "Tous les fichiers", Pattern: "*.*"},
		},
	})
	if err != nil {
		return Task{}, err
	}
	if len(paths) == 0 {
		return a.store.getTask(taskID)
	}
	for _, path := range paths {
		if _, err := a.store.importAttachment(taskID, path); err != nil {
			return Task{}, err
		}
	}
	a.requestSync()
	return a.store.getTask(taskID)
}

// AddAttachmentPaths stores files copied from the file manager, when the
// clipboard exposes their local paths instead of file blobs.
func (a *App) AddAttachmentPaths(taskID string, paths []string) (Task, error) {
	if err := a.ready(); err != nil {
		return Task{}, err
	}
	if _, err := a.store.getTask(taskID); err != nil {
		return Task{}, err
	}
	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			continue
		}
		if _, err := a.store.importAttachment(taskID, path); err != nil {
			return Task{}, err
		}
	}
	if len(paths) > 0 {
		a.requestSync()
	}
	return a.store.getTask(taskID)
}

// AddAttachments stores files pasted from the clipboard. Data is a data URL or
// raw base64 payload, so the frontend can forward any clipboard document.
func (a *App) AddAttachments(taskID string, files []AttachmentInput) (Task, error) {
	if err := a.ready(); err != nil {
		return Task{}, err
	}
	if _, err := a.store.getTask(taskID); err != nil {
		return Task{}, err
	}
	for _, file := range files {
		data, err := decodeAttachmentData(file.Data)
		if err != nil {
			return Task{}, fmt.Errorf("pièce jointe %q : %w", file.Name, err)
		}
		if _, err := a.store.importAttachmentBytes(taskID, file.Name, data); err != nil {
			return Task{}, err
		}
	}
	if len(files) > 0 {
		a.requestSync()
	}
	return a.store.getTask(taskID)
}

func decodeAttachmentData(data string) ([]byte, error) {
	data = strings.TrimSpace(data)
	if strings.HasPrefix(data, "data:") {
		comma := strings.Index(data, ",")
		if comma < 0 {
			return nil, errors.New("données de pièce jointe illisibles")
		}
		data = data[comma+1:]
	}
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err == nil {
		return decoded, nil
	}
	return base64.RawStdEncoding.DecodeString(data)
}

// DeleteAttachment removes a stored file from a task.
func (a *App) DeleteAttachment(id string) (Task, error) {
	if err := a.ready(); err != nil {
		return Task{}, err
	}
	task, err := a.store.deleteAttachment(id)
	if err == nil {
		a.requestSync()
	}
	return task, err
}

// OpenAttachment opens a document with the system application. Images use the
// in-app lightbox instead.
func (a *App) OpenAttachment(id string) error {
	if err := a.ready(); err != nil {
		return err
	}
	path, err := a.store.attachmentPath(id)
	if err != nil {
		return err
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
		hideWindow(cmd)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}

// AddImage and DeleteImage are kept for older frontend builds.
func (a *App) AddImage(taskID string) (Task, error) { return a.AddAttachment(taskID) }

func (a *App) DeleteImage(id string) (Task, error) { return a.DeleteAttachment(id) }
