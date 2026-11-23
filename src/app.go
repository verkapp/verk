package src

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

// App is the service bound to the Angular frontend.
type App struct {
	ctx     context.Context
	store   *store
	initErr error

	wapp     *application.App
	window   *application.WebviewWindow
	quitting atomic.Bool

	// Git synchronisation runtime state. The configuration itself lives in
	// the local settings table so each machine keeps its own remote.
	gitAvailable bool

	notifier        *notifications.NotificationService
	notificationErr error
	reminder        *reminderState

	syncRunMu sync.Mutex

	syncStateMu    sync.Mutex
	syncTimer      *time.Timer
	syncSyncing    bool
	syncGeneration int64
	syncPending    int
	syncBatchStart time.Time

	updates    updater
	updateStop chan struct{}
}

// NewApp opens the local database before the Wails window is created.
func NewApp() *App {
	app := &App{gitAvailable: gitInstalled(), updateStop: make(chan struct{})}
	if st, err := openStore(); err != nil {
		app.initErr = err
	} else {
		app.store = st
	}
	return app
}

// ServiceStartup is called by Wails v3 before the window is shown.
func (a *App) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	a.ctx = ctx
	if a.initErr != nil {
		log.Printf("initialisation: %v", a.initErr)
		// Dialogs need the main loop: show it once the app has started.
		a.wapp.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
			a.wapp.Dialog.Error().
				SetTitle("Todo").
				SetMessage("Impossible d'ouvrir la base de données.\n\n" + a.initErr.Error()).
				Show()
		})
		return nil
	}
	// Pull remote changes without delaying the window: the frontend loads the
	// local state first, then refreshes when the sync generation changes.
	go a.autoSync()
	go a.autoUpdateLoop()
	a.startNotifications(ctx)
	return nil
}

// ServiceShutdown runs when the application quits.
func (a *App) ServiceShutdown() error {
	a.quitting.Store(true)
	close(a.updateStop)
	a.stopNotifications()
	a.stopSyncTimer()
	if a.store != nil {
		a.installUpdateOnQuit()
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
	return nil
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
