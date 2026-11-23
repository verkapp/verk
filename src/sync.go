package src

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	syncDebounce    = 8 * time.Second
	syncMaxBatch    = 45 * time.Second
	syncRunTimeout  = 90 * time.Second
	syncQuitTimeout = 12 * time.Second
	syncAttempts    = 3
)

func (a *App) syncDirectory() string {
	if a.store == nil {
		return ""
	}
	return filepath.Join(a.store.dataDir, syncDirName)
}

func (a *App) syncConfigured() (syncConfig, bool) {
	if a.store == nil {
		return syncConfig{}, false
	}
	cfg, err := a.store.syncConfig()
	if err != nil {
		return syncConfig{}, false
	}
	return cfg, cfg.Remote != ""
}

// requestSync schedules a background push after a local change. Rapid edits
// (typing a title, ticking several subtasks) are coalesced into one commit:
// the timer restarts on each change, but a batch never waits longer than
// syncMaxBatch so continuous activity still reaches the remote.
func (a *App) requestSync() {
	if a.store == nil {
		return
	}
	cfg, ok := a.syncConfigured()
	if !ok || !cfg.Enabled {
		return
	}
	a.syncStateMu.Lock()
	now := time.Now()
	if a.syncPending == 0 {
		a.syncBatchStart = now
	}
	a.syncPending++
	if a.syncTimer != nil {
		a.syncTimer.Stop()
	}
	delay := syncDebounce
	if !a.syncBatchStart.IsZero() {
		if remaining := syncMaxBatch - now.Sub(a.syncBatchStart); remaining < delay {
			delay = remaining
		}
	}
	if delay < 0 {
		delay = 0
	}
	a.syncTimer = time.AfterFunc(delay, a.autoSync)
	a.syncStateMu.Unlock()
}

// takeSyncBatch resets the pending action count and returns it for the commit
// message.
func (a *App) takeSyncBatch() int {
	a.syncStateMu.Lock()
	defer a.syncStateMu.Unlock()
	pending := a.syncPending
	a.syncPending = 0
	a.syncBatchStart = time.Time{}
	if a.syncTimer != nil {
		a.syncTimer.Stop()
		a.syncTimer = nil
	}
	return pending
}

func (a *App) stopSyncTimer() {
	a.syncStateMu.Lock()
	if a.syncTimer != nil {
		a.syncTimer.Stop()
		a.syncTimer = nil
	}
	a.syncPending = 0
	a.syncBatchStart = time.Time{}
	a.syncStateMu.Unlock()
}

func (a *App) autoSync() {
	cfg, ok := a.syncConfigured()
	if !ok || !cfg.Enabled {
		a.takeSyncBatch()
		return
	}
	pending := a.takeSyncBatch()
	ctx, cancel := context.WithTimeout(context.Background(), syncRunTimeout)
	defer cancel()
	_, _ = a.syncNowWithMessage(ctx, false, batchCommitMessage(pending))
}

// syncNow serialises all sync runs. Neither the automatic nor the manual path
// can overlap with another run.
func (a *App) syncNow(ctx context.Context, force bool) (bool, error) {
	return a.syncNowWithMessage(ctx, force, "")
}

func (a *App) syncNowWithMessage(ctx context.Context, force bool, message string) (bool, error) {
	a.syncRunMu.Lock()
	defer a.syncRunMu.Unlock()

	cfg, err := a.store.syncConfig()
	if err != nil {
		return false, err
	}
	if cfg.Remote == "" || (!force && !cfg.Enabled) {
		return false, nil
	}
	if !a.gitAvailable {
		return false, errors.New("git est introuvable sur cette machine")
	}

	a.setSyncing(true)
	// Deferred calls run in reverse order: the event goes out once syncing
	// is false again, so the frontend reads the final status.
	defer a.emitSyncStatus()
	defer a.setSyncing(false)

	changed, syncErr := a.performSyncWithMessage(ctx, cfg, message)
	now := time.Now().UnixMilli()
	if syncErr != nil {
		errorText := syncErr.Error()
		lastAt, _, metaErr := a.store.syncMeta()
		if metaErr == nil {
			_ = a.store.saveSyncMeta(lastAt, errorText)
		}
		return false, syncErr
	}
	_ = a.store.saveSyncMeta(now, "")
	if changed {
		a.syncStateMu.Lock()
		a.syncGeneration++
		a.syncStateMu.Unlock()
	}
	return changed, nil
}

func (a *App) performSync(ctx context.Context, cfg syncConfig) (bool, error) {
	return a.performSyncWithMessage(ctx, cfg, "")
}

func (a *App) performSyncWithMessage(ctx context.Context, cfg syncConfig, message string) (bool, error) {
	dir := a.syncDirectory()
	branch := cfg.Branch
	if branch == "" {
		branch = "main"
	}
	if err := ensureSyncRepo(ctx, dir, cfg.Remote, branch); err != nil {
		return false, fmt.Errorf("préparation du dépôt : %w", err)
	}
	imagesDir := filepath.Join(dir, "images")
	remoteRef := "refs/remotes/origin/" + branch
	changed := false

	var lastPushErr error
	for attempt := 0; attempt < syncAttempts; attempt++ {
		if _, err := runGit(ctx, dir, "fetch", "--prune", "origin"); err != nil {
			return changed, fmt.Errorf("récupération impossible : %w", err)
		}

		if gitRefExists(ctx, dir, remoteRef) {
			if _, err := runGit(ctx, dir, "reset", "--hard", remoteRef); err != nil {
				return changed, fmt.Errorf("préparation du dépôt : %w", err)
			}
			data, err := os.ReadFile(filepath.Join(dir, "todo.json"))
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return changed, err
			}
			remoteSnapshot, err := decodeSnapshot(data)
			if err != nil {
				return changed, err
			}
			merged, mergedChanged, err := a.store.mergeSnapshot(remoteSnapshot)
			if err != nil {
				return changed, err
			}
			if mergedChanged {
				changed = true
			}
			// Bring down the image files referenced by the merged dataset.
			if err := copyMissingImages(merged.Images, imagesDir, a.store.imageDir); err != nil {
				return changed, err
			}
		}

		localSnapshot, err := a.store.snapshot()
		if err != nil {
			return changed, err
		}
		if err := writeSnapshotFile(filepath.Join(dir, "todo.json"), localSnapshot); err != nil {
			return changed, err
		}
		if err := copyMissingImages(localSnapshot.Images, a.store.imageDir, imagesDir); err != nil {
			return changed, err
		}

		if _, err := runGit(ctx, dir, "add", "-A"); err != nil {
			return changed, fmt.Errorf("index git : %w", err)
		}
		status, err := runGit(ctx, dir, "status", "--porcelain")
		if err != nil {
			return changed, fmt.Errorf("état git : %w", err)
		}
		if strings.TrimSpace(status) != "" {
			commitMessage := message
			if commitMessage == "" {
				commitMessage = fmt.Sprintf("sync: %s %s", hostname(), time.Now().UTC().Format(time.RFC3339))
			}
			if _, err := runGit(ctx, dir, "commit", "-m", commitMessage); err != nil {
				return changed, fmt.Errorf("commit impossible : %w", err)
			}
		}

		if _, err := runGit(ctx, dir, "push", "origin", "HEAD:"+branch); err != nil {
			message := err.Error()
			retryable := strings.Contains(message, "rejected") ||
				strings.Contains(message, "non-fast-forward") ||
				strings.Contains(message, "fetch first")
			if !retryable {
				return changed, fmt.Errorf("envoi impossible : %w", err)
			}
			lastPushErr = err
			continue
		}
		return changed, nil
	}
	return changed, fmt.Errorf("envoi impossible : %w", lastPushErr)
}

func (a *App) setSyncing(value bool) {
	a.syncStateMu.Lock()
	a.syncSyncing = value
	a.syncStateMu.Unlock()
}

// emitSyncStatus pushes the sync status to the frontend, which reloads the
// tasks right away when a pull changed them instead of waiting for its poll.
func (a *App) emitSyncStatus() {
	if a.ctx == nil {
		return
	}
	wruntime.EventsEmit(a.ctx, "sync:status", a.currentSyncStatus())
}

func (a *App) currentSyncStatus() SyncStatus {
	status := SyncStatus{Branch: "main", GitAvailable: a.gitAvailable}
	if a.store != nil {
		cfg, err := a.store.syncConfig()
		if err == nil {
			status.Configured = cfg.Remote != ""
			status.Remote = cfg.Remote
			status.Enabled = cfg.Enabled
			status.Branch = cfg.Branch
		}
		if lastAt, lastErr, err := a.store.syncMeta(); err == nil {
			status.LastSyncAt = lastAt
			status.LastError = lastErr
		}
	}
	a.syncStateMu.Lock()
	status.Syncing = a.syncSyncing
	status.Generation = a.syncGeneration
	a.syncStateMu.Unlock()
	if status.Configured && !a.gitAvailable {
		status.LastError = "git est introuvable sur cette machine"
	}
	return status
}

func (a *App) configureSync(remote string, enabled bool) error {
	remote = strings.TrimSpace(remote)
	cfg, err := a.store.syncConfig()
	if err != nil {
		return err
	}
	if remote == "" {
		cfg.Remote = ""
		cfg.Enabled = false
		if err := a.store.saveSyncConfig(cfg); err != nil {
			return err
		}
		a.stopSyncTimer()
		_ = a.store.saveSyncMeta(0, "")
		return nil
	}
	if !a.gitAvailable {
		return errors.New("git est introuvable sur cette machine : installe git pour synchroniser tes tâches")
	}
	branch := cfg.Branch
	if branch == "" || remote != cfg.Remote {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		branch, err = detectRemoteBranch(ctx, a.store.dataDir, remote)
		cancel()
		if err != nil {
			return fmt.Errorf("dépôt git inaccessible : %w", err)
		}
	}
	// Switching to another repository starts from a clean checkout so files
	// from the previous remote cannot leak into the new one.
	if cfg.Remote != "" && cfg.Remote != remote {
		if err := os.RemoveAll(a.syncDirectory()); err != nil {
			return fmt.Errorf("préparation du dépôt : %w", err)
		}
	}
	cfg = syncConfig{Remote: remote, Enabled: enabled, Branch: branch}
	if err := a.store.saveSyncConfig(cfg); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := ensureSyncRepo(ctx, a.syncDirectory(), remote, branch); err != nil {
		return fmt.Errorf("préparation du dépôt : %w", err)
	}
	if enabled {
		a.requestSync()
	} else {
		a.stopSyncTimer()
	}
	return nil
}

func writeSnapshotFile(path string, snap snapshot) error {
	data, err := json.MarshalIndent(snap.body(), "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func hostname() string {
	name, err := os.Hostname()
	if err != nil || name == "" {
		return "todo"
	}
	return name
}

// batchCommitMessage summarises the actions coalesced by the debounce timer.
func batchCommitMessage(changes int) string {
	if changes <= 0 {
		return ""
	}
	label := "modification"
	if changes > 1 {
		label = "modifications"
	}
	return fmt.Sprintf("sync: %d %s — %s", changes, label, hostname())
}
