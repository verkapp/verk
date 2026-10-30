package src

import (
	"context"
	"fmt"
	"log"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the object bound to the Angular frontend.
type App struct {
	ctx     context.Context
	store   *store
	initErr error
}

// NewApp opens the local database before the Wails window is created.
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
