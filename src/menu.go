package src

import (
	"runtime"

	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// Menu actions are sent to the frontend through the menu:action event; the
// Angular shell maps each one to a navigation or a command.
const (
	actionNewTask  = "new-task"
	actionRefresh  = "refresh"
	actionToday    = "view:today"
	actionAll      = "view:all"
	actionCoach    = "view:coach"
	actionHistory  = "view:history"
	actionSettings = "view:settings"
)

// applicationMenu builds the native menu bar: the global bar on macOS, the
// window menu on Windows and Linux.
func (a *App) applicationMenu() *menu.Menu {
	isMac := runtime.GOOS == "darwin"
	bar := menu.NewMenu()
	if isMac {
		bar.Append(menu.AppMenu())
	}

	file := bar.AddSubmenu("Fichier")
	file.AddText("Nouvelle tâche", keys.CmdOrCtrl("n"), a.menuAction(actionNewTask))
	file.AddText("Rafraîchir", keys.CmdOrCtrl("r"), a.menuAction(actionRefresh))
	file.AddSeparator()
	file.AddText("Réduire dans la barre", keys.Combo("m", keys.CmdOrCtrlKey, keys.ShiftKey), func(*menu.CallbackData) {
		a.hideToTray()
	})
	if !isMac {
		file.AddSeparator()
		file.AddText("Quitter", keys.CmdOrCtrl("q"), func(*menu.CallbackData) { a.quitApp() })
	}

	bar.Append(menu.EditMenu())

	view := bar.AddSubmenu("Affichage")
	view.AddText("Aujourd’hui", keys.CmdOrCtrl("1"), a.menuAction(actionToday))
	view.AddText("Toutes les tâches", keys.CmdOrCtrl("2"), a.menuAction(actionAll))
	view.AddText("Coach", keys.CmdOrCtrl("3"), a.menuAction(actionCoach))
	view.AddText("Historique", keys.CmdOrCtrl("4"), a.menuAction(actionHistory))
	view.AddSeparator()
	view.AddText("Réglages", keys.CmdOrCtrl(","), a.menuAction(actionSettings))

	if isMac {
		bar.Append(menu.WindowMenu())
	}
	return bar
}

func (a *App) menuAction(action string) menu.Callback {
	return func(*menu.CallbackData) { a.sendAction(action) }
}

// sendAction brings the window back if needed and forwards the action.
func (a *App) sendAction(action string) {
	if a.ctx == nil {
		return
	}
	a.showWindow()
	wruntime.EventsEmit(a.ctx, "menu:action", action)
}

func (a *App) showWindow() {
	if a.ctx == nil {
		return
	}
	wruntime.WindowShow(a.ctx)
	wruntime.WindowUnminimise(a.ctx)
	activateApp()
}

// hideToTray hides the window; the tray icon brings it back.
func (a *App) hideToTray() {
	if a.ctx == nil {
		return
	}
	wruntime.WindowHide(a.ctx)
}

func (a *App) quitApp() {
	if a.ctx == nil {
		return
	}
	wruntime.Quit(a.ctx)
}

// Tray menu entries, shared by the macOS and the Windows/Linux trays.
const (
	trayOpen = iota + 1
	trayNewTask
	trayHide
	trayQuit
)

func (a *App) trayAction(id int) {
	switch id {
	case trayOpen:
		a.showWindow()
	case trayNewTask:
		a.sendAction(actionNewTask)
	case trayHide:
		a.hideToTray()
	case trayQuit:
		a.quitApp()
	}
}
