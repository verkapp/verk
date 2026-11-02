package src

import (
	"os/exec"
	"path/filepath"
	"runtime"
)

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
	info := map[string]string{"version": Version, "database": "todo.db"}
	if a.store != nil {
		info["dataDir"] = a.store.dataDir
		info["databasePath"] = filepath.Join(a.store.dataDir, "todo.db")
	}
	return info
}
