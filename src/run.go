package src

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed trayicons/appicon.png
var appIcon []byte

// Run starts the desktop application using the embedded Angular bundle.
func Run(assets embed.FS) {
	app := NewApp()

	wapp := application.New(application.Options{
		Name:        "Todo",
		Description: "Une liste de tâches douce, avec un coach Claude Code.",
		Icon:        appIcon,
		Services: []application.Service{
			application.NewService(app),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			// The window can be hidden in the tray: the app must keep running.
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
		Windows: application.WindowsOptions{
			DisableQuitOnLastWindowClosed: true,
		},
	})
	app.wapp = wapp

	app.window = wapp.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "main",
		Title:            "Todo",
		Width:            1240,
		Height:           840,
		MinWidth:         980,
		MinHeight:        680,
		BackgroundColour: application.NewRGB(250, 249, 248),
		URL:              "/",
		Mac: application.MacWindow{
			Appearance: application.NSAppearanceNameAqua,
		},
		Windows: application.WindowsWindow{
			Theme: application.Light,
		},
	})

	wapp.Menu.Set(app.applicationMenu())
	app.setupWindow()
	app.setupTray()

	if err := wapp.Run(); err != nil {
		log.Fatal(err)
	}
}
