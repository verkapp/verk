package src

import (
	_ "embed"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed trayicons/template.png
var trayTemplateIcon []byte

//go:embed trayicons/color.png
var trayColorIcon []byte

// setupTray adds the menu bar (macOS) or notification area (Windows, Linux)
// icon. On macOS a click opens the menu; elsewhere a left click shows the
// window and a right click opens the menu.
func (a *App) setupTray() {
	tray := a.wapp.SystemTray.New()
	if runtime.GOOS == "darwin" {
		tray.SetTemplateIcon(trayTemplateIcon)
	} else {
		tray.SetIcon(trayColorIcon)
	}
	tray.SetTooltip("Todo")

	menu := application.NewMenu()
	menu.Add("Ouvrir Todo").OnClick(func(*application.Context) { a.showWindow() })
	menu.Add("Nouvelle tâche").OnClick(func(*application.Context) { a.sendAction(actionNewTask) })
	menu.AddSeparator()
	menu.Add("Réduire dans la barre").OnClick(func(*application.Context) { a.hideToTray() })
	menu.AddSeparator()
	menu.Add("Quitter Todo").OnClick(func(*application.Context) { a.quitApp() })
	tray.SetMenu(menu)

	if runtime.GOOS != "darwin" {
		tray.OnClick(a.showWindow)
	}
}

// setupWindow wires the close button and the Dock icon. Closing either hides
// the window in the tray (when the setting is on) or quits, as before.
func (a *App) setupWindow() {
	a.window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		if a.quitting.Load() {
			return
		}
		e.Cancel()
		if a.closeToTray() {
			a.window.Hide()
			return
		}
		go a.quitApp()
	})
	// Clicking the Dock icon while the window is hidden brings it back.
	a.wapp.Event.OnApplicationEvent(events.Mac.ApplicationShouldHandleReopen, func(*application.ApplicationEvent) {
		a.showWindow()
	})
}

func (a *App) showWindow() {
	if a.window == nil {
		return
	}
	a.window.Show()
	a.window.UnMinimise()
	a.window.Focus()
}

// hideToTray hides the window; the tray icon brings it back.
func (a *App) hideToTray() {
	if a.window != nil {
		a.window.Hide()
	}
}

func (a *App) quitApp() {
	if a.wapp == nil {
		return
	}
	a.quitting.Store(true)
	a.wapp.Quit()
}

// closeToTray is read on every close, so the setting applies immediately.
func (a *App) closeToTray() bool {
	if a.store == nil {
		return false
	}
	settings, err := a.store.getSettings()
	return err == nil && settings.CloseToTray
}

// emit sends an event to the frontend once the application is running.
func (a *App) emit(name string, data any) {
	if a.wapp == nil || a.ctx == nil {
		return
	}
	a.wapp.Event.Emit(name, data)
}
