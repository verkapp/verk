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
	previous, err := a.store.getSettings()
	if err != nil {
		return err
	}
	if err := a.store.saveSettings(settings); err != nil {
		return err
	}
	if normalizeTheme(settings.Theme) != previous.Theme {
		a.refreshNativeTheme()
	}
	return nil
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
	info := map[string]string{"version": Version, "database": databaseName}
	if a.store != nil {
		info["dataDir"] = a.store.dataDir
		info["databasePath"] = filepath.Join(a.store.dataDir, databaseName)
	}
	return info
}
