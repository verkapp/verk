package src

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the object bound to the Angular frontend.
type App struct {
	ctx     context.Context
	store   *store
	initErr error

	// Git synchronisation runtime state. The configuration itself lives in
	// the local settings table so each machine keeps its own remote.
	gitAvailable bool

	notificationErr error
	reminder        *reminderState

	syncRunMu sync.Mutex

	syncStateMu    sync.Mutex
	syncTimer      *time.Timer
	syncSyncing    bool
	syncGeneration int64
}

// NewApp opens the local database before the Wails window is created.
func NewApp() *App {
	app := &App{gitAvailable: gitInstalled()}
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
		return
	}
	// Pull remote changes without delaying the window: the frontend loads the
	// local state first, then refreshes when the sync generation changes.
	go a.autoSync()
	a.startNotifications()
}

func (a *App) shutdown(ctx context.Context) {
	a.stopNotifications()
	a.stopSyncTimer()
	if a.store != nil {
		if cfg, ok := a.syncConfigured(); ok && cfg.Enabled {
			// Flush pending edits on quit, but never wait for an already
			// running sync that could take longer than the timeout.
			if a.syncRunMu.TryLock() {
				runCtx, cancel := context.WithTimeout(context.Background(), syncQuitTimeout)
				a.setSyncing(true)
				_, _ = a.performSync(runCtx, cfg)
				a.setSyncing(false)
				cancel()
				a.syncRunMu.Unlock()
			}
		}
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
