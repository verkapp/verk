package src

import (
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
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
func (a *App) applicationMenu() *application.Menu {
	isMac := runtime.GOOS == "darwin"
	bar := application.NewMenu()
	if isMac {
		bar.AddRole(application.AppMenu)
	}

	file := bar.AddSubmenu("Fichier")
	file.Add("Nouvelle tâche").SetAccelerator("CmdOrCtrl+N").OnClick(a.menuAction(actionNewTask))
	file.Add("Rafraîchir").SetAccelerator("CmdOrCtrl+R").OnClick(a.menuAction(actionRefresh))
	file.AddSeparator()
	file.Add("Réduire dans la barre").SetAccelerator("CmdOrCtrl+Shift+M").OnClick(func(*application.Context) {
		a.hideToTray()
	})
	if !isMac {
		file.AddSeparator()
		file.Add("Quitter").SetAccelerator("CmdOrCtrl+Q").OnClick(func(*application.Context) { a.quitApp() })
	}

	editMenu(bar.AddSubmenu("Édition"), isMac)

	view := bar.AddSubmenu("Affichage")
	view.Add("Aujourd’hui").SetAccelerator("CmdOrCtrl+1").OnClick(a.menuAction(actionToday))
	view.Add("Toutes les tâches").SetAccelerator("CmdOrCtrl+2").OnClick(a.menuAction(actionAll))
	view.Add("Coach").SetAccelerator("CmdOrCtrl+3").OnClick(a.menuAction(actionCoach))
	view.Add("Historique").SetAccelerator("CmdOrCtrl+4").OnClick(a.menuAction(actionHistory))
	view.AddSeparator()
	view.Add("Réglages").SetAccelerator("CmdOrCtrl+,").OnClick(a.menuAction(actionSettings))

	if isMac {
		bar.AddRole(application.WindowMenu)
	}
	return bar
}

func (a *App) menuAction(action string) func(*application.Context) {
	return func(*application.Context) { a.sendAction(action) }
}

// sendAction brings the window back if needed and forwards the action.
func (a *App) sendAction(action string) {
	a.showWindow()
	a.emit("menu:action", action)
}

// roleLabel is a native menu role with its French label; an empty label
// stands for a separator.
type roleLabel struct {
	role  application.Role
	label string
}

// editMenu rebuilds Wails' Edit role with French labels: the roles keep
// their native behaviour (clipboard, undo in the webview), only the text
// changes. Wails' own EditMenu role is labelled in English.
func editMenu(edit *application.Menu, isMac bool) {
	items := []roleLabel{
		{application.Undo, "Annuler"},
		{application.Redo, "Rétablir"},
		{},
		{application.Cut, "Couper"},
		{application.Copy, "Copier"},
		{application.Paste, "Coller"},
	}
	if isMac {
		items = append(items, roleLabel{application.PasteAndMatchStyle, "Coller et adapter le style"})
	}
	items = append(items,
		roleLabel{application.Delete, "Supprimer"},
		roleLabel{},
		roleLabel{application.SelectAll, "Tout sélectionner"},
	)
	for _, item := range items {
		if item.label == "" {
			edit.AddSeparator()
			continue
		}
		edit.AddRole(item.role)
		if added := edit.FindByRole(item.role); added != nil {
			added.SetLabel(item.label)
		}
	}
}
