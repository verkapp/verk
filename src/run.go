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
		Name:        "Verk",
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

	// The theme is resolved to light or dark here, never left to Wails'
	// « system » mode: the app repaints the native chrome itself when the
	// setting or the OS appearance changes (see window_theme.go).
	theme := app.startupTheme()
	dark := resolveDark(theme)
	// Wails v3 attaches the application menu to a window only when the menu
	// exists before the window and the window opts in (UseApplicationMenu).
	wapp.Menu.Set(app.applicationMenu())
	app.window = wapp.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:               "main",
		Title:              "Verk",
		UseApplicationMenu: true,
		Width:              1240,
		Height:             840,
		MinWidth:           980,
		MinHeight:          680,
		BackgroundColour:   windowBackground(dark),
		URL:                "/",
		// macOS: the title bar is transparent and the content runs under it,
		// so the traffic lights sit on top of the sidebar. The frontend keeps
		// that strip free and draggable (see .titlebar in app.component).
		Mac: application.MacWindow{
			Appearance: macAppearance(theme),
			TitleBar:   application.MacTitleBarHidden,
		},
		// Windows: the native title bar and menu bar take the app's colours
		// instead of the system grey.
		Windows: application.WindowsWindow{
			Theme:       windowsTheme(dark),
			CustomTheme: windowsChrome(),
		},
	})

	app.setupWindow()
	app.setupTray()
	app.watchSystemTheme()

	if err := wapp.Run(); err != nil {
		log.Fatal(err)
	}
}
