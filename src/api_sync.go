package src

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
)

// SyncStatus reports whether git sync is configured and its latest result.
func (a *App) SyncStatus() SyncStatus {
	if a.store == nil {
		return SyncStatus{Branch: "main", GitAvailable: a.gitAvailable}
	}
	return a.currentSyncStatus()
}

// SetSyncConfig validates the remote with the system git credentials, stores
// the per-machine configuration and starts a first sync in the background.
func (a *App) SetSyncConfig(remote string, enabled bool) error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.configureSync(remote, enabled)
}

// SyncNow forces a pull + merge + push, used by the manual button.
func (a *App) SyncNow() (SyncStatus, error) {
	if err := a.ready(); err != nil {
		return a.currentSyncStatus(), err
	}
	cfg, err := a.store.syncConfig()
	if err != nil {
		return a.currentSyncStatus(), err
	}
	if cfg.Remote == "" {
		return a.currentSyncStatus(), errors.New("aucun dépôt git n'est configuré")
	}
	ctx, cancel := context.WithTimeout(context.Background(), syncRunTimeout)
	defer cancel()
	_, err = a.syncNow(ctx, true)
	return a.currentSyncStatus(), err
}

// OpenSyncFolder reveals the app-managed checkout for troubleshooting.
func (a *App) OpenSyncFolder() error {
	if err := a.ready(); err != nil {
		return err
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", a.syncDirectory())
	case "windows":
		cmd = exec.Command("explorer", a.syncDirectory())
	default:
		cmd = exec.Command("xdg-open", a.syncDirectory())
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("ouverture du dossier de synchronisation : %w", err)
	}
	return nil
}
