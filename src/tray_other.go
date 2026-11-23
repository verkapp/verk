//go:build !darwin

package src

import (
	_ "embed"
	goruntime "runtime"

	"fyne.io/systray"
)

//go:embed trayicons/color.ico
var trayIconICO []byte

//go:embed trayicons/color.png
var trayIconPNG []byte

// startTray runs the tray on its own locked OS thread: on Windows the
// message loop must run on the thread that created the tray window, and on
// Linux the tray only talks to DBus, independently of the GTK loop.
func (a *App) startTray() {
	go func() {
		goruntime.LockOSThread()
		systray.Run(func() { a.buildTray() }, nil)
	}()
}

func (a *App) buildTray() {
	if goruntime.GOOS == "windows" {
		systray.SetIcon(trayIconICO)
	} else {
		systray.SetIcon(trayIconPNG)
	}
	systray.SetTooltip("Todo")
	systray.SetOnTapped(func() { a.trayAction(trayOpen) })

	items := map[int]*systray.MenuItem{}
	items[trayOpen] = systray.AddMenuItem("Ouvrir Todo", "")
	items[trayNewTask] = systray.AddMenuItem("Nouvelle tâche", "")
	systray.AddSeparator()
	items[trayHide] = systray.AddMenuItem("Réduire dans la barre", "")
	systray.AddSeparator()
	items[trayQuit] = systray.AddMenuItem("Quitter Todo", "")

	for id, item := range items {
		go func(id int, clicked <-chan struct{}) {
			for range clicked {
				a.trayAction(id)
			}
		}(id, item.ClickedCh)
	}
}

func (a *App) stopTray() {
	systray.Quit()
}

func activateApp() {}
