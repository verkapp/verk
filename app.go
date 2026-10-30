package main

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"runtime"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx     context.Context
	store   *store
	initErr error
}

func NewApp() *App {
	app := &App{}
	if st, err := openStore(); err != nil {
		app.initErr = err
	} else {
		app.store = st
	}
	return app
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if a.initErr != nil {
		log.Printf("initialisation: %v", a.initErr)
		_, _ = wruntime.MessageDialog(ctx, wruntime.MessageDialogOptions{
			Type:    wruntime.ErrorDialog,
			Title:   "Todo",
			Message: "Impossible d'ouvrir la base de données.\n\n" + a.initErr.Error(),
		})
	}
}

func (a *App) shutdown(ctx context.Context) {
	if a.store != nil {
		_ = a.store.close()
	}
}

func (a *App) ready() error {
	if a.store == nil {
		if a.initErr != nil {
			return fmt.Errorf("base de données indisponible : %w", a.initErr)
		}
		return fmt.Errorf("l'application n'est pas encore prête")
	}
	return nil
}

func (a *App) ListTasks() ([]Task, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.store.listTasks()
}

func (a *App) CreateTask(input TaskInput) (Task, error) {
	if err := a.ready(); err != nil {
		return Task{}, err
	}
	return a.store.createTask(input)
}

func (a *App) UpdateTask(id string, input TaskInput) (Task, error) {
	if err := a.ready(); err != nil {
		return Task{}, err
	}
	return a.store.updateTask(id, input)
}

func (a *App) SetTaskDone(id string, done bool) (Task, error) {
	if err := a.ready(); err != nil {
		return Task{}, err
	}
	return a.store.setDone(id, done)
}

func (a *App) DeleteTask(id string) error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.store.deleteTask(id)
}

func (a *App) AddSubtask(taskID, title string) (Task, error) {
	if err := a.ready(); err != nil {
		return Task{}, err
	}
	return a.store.addSubtask(taskID, title)
}

func (a *App) AddSubtasks(taskID string, titles []string) (Task, error) {
	if err := a.ready(); err != nil {
		return Task{}, err
	}
	return a.store.addSubtasks(taskID, titles)
}

func (a *App) UpdateSubtask(id, title string) (Task, error) {
	if err := a.ready(); err != nil {
		return Task{}, err
	}
	return a.store.updateSubtask(id, title)
}

func (a *App) SetSubtaskDone(id string, done bool) (Task, error) {
	if err := a.ready(); err != nil {
		return Task{}, err
	}
	return a.store.setSubtaskDone(id, done)
}

func (a *App) DeleteSubtask(id string) (Task, error) {
	if err := a.ready(); err != nil {
		return Task{}, err
	}
	return a.store.deleteSubtask(id)
}

// AddImage opens the native file picker, copies the chosen image into the
// application data directory and returns the refreshed task. Cancelling is not
// an error: the task is simply returned unchanged.
func (a *App) AddImage(taskID string) (Task, error) {
	if err := a.ready(); err != nil {
		return Task{}, err
	}
	if _, err := a.store.getTask(taskID); err != nil {
		return Task{}, err
	}
	path, err := wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{
		Title: "Choisir une image",
		Filters: []wruntime.FileFilter{
			{DisplayName: "Images", Pattern: "*.png;*.jpg;*.jpeg;*.gif;*.webp;*.bmp"},
		},
	})
	if err != nil {
		return Task{}, err
	}
	if path == "" {
		return a.store.getTask(taskID)
	}
	return a.store.importImage(taskID, path)
}

func (a *App) DeleteImage(id string) (Task, error) {
	if err := a.ready(); err != nil {
		return Task{}, err
	}
	return a.store.deleteImage(id)
}

func (a *App) GetSettings() (Settings, error) {
	if err := a.ready(); err != nil {
		return Settings{}, err
	}
	return a.store.getSettings()
}

func (a *App) SaveSettings(settings Settings) error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.store.saveSettings(settings)
}

func (a *App) ClaudeStatus() ClaudeStatus {
	if err := a.ready(); err != nil {
		return ClaudeStatus{Error: err.Error()}
	}
	return a.claudeStatus()
}

func (a *App) BreakDownTask(taskID string) ([]string, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.breakDownTask(taskID)
}

func (a *App) CoachNudge() (string, error) {
	if err := a.ready(); err != nil {
		return "", err
	}
	return a.coachNudge()
}

// OpenDataFolder reveals the SQLite database and images in the file manager.
func (a *App) OpenDataFolder() error {
	if err := a.ready(); err != nil {
		return err
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", a.store.dataDir)
	case "windows":
		cmd = exec.Command("explorer", a.store.dataDir)
	default:
		cmd = exec.Command("xdg-open", a.store.dataDir)
	}
	return cmd.Start()
}

func (a *App) AppInfo() map[string]string {
	info := map[string]string{"version": "1.0.0", "database": "todo.db"}
	if a.store != nil {
		info["dataDir"] = a.store.dataDir
		info["databasePath"] = filepath.Join(a.store.dataDir, "todo.db")
	}
	return info
}
