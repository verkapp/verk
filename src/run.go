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

	app.window = wapp.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "main",
		Title:            "Verk",
		Width:            1240,
		Height:           840,
		MinWidth:         980,
		MinHeight:        680,
		BackgroundColour: application.NewRGB(250, 249, 248),
		URL:              "/",
		// macOS: the title bar is transparent and the content runs under it,
		// so the traffic lights sit on top of the sidebar. The frontend keeps
		// that strip free and draggable (see .titlebar in app.component).
		Mac: application.MacWindow{
			Appearance: application.NSAppearanceNameAqua,
			TitleBar:   application.MacTitleBarHidden,
		},
		// Windows: the native title bar and menu bar take the app's colours
		// instead of the system grey.
		Windows: application.WindowsWindow{
			Theme:       application.Light,
			CustomTheme: windowsChrome(),
		},
	})

	wapp.Menu.Set(app.applicationMenu())
	app.setupWindow()
	app.setupTray()

	if err := wapp.Run(); err != nil {
		log.Fatal(err)
	}
}

// windowsChrome mirrors the frontend tokens in styles.css: --bg for the title
// and menu bars, --text and --muted for labels, --line for the border and
// --accent-soft for the hovered menu entry.
func windowsChrome() application.ThemeSettings {
	bg := application.NewRGBPtr(0xfb, 0xfa, 0xf8)
	line := application.NewRGBPtr(0xea, 0xe7, 0xe1)
	text := application.NewRGBPtr(0x26, 0x24, 0x2a)
	muted := application.NewRGBPtr(0x8f, 0x8b, 0x92)
	hover := application.NewRGBPtr(0xec, 0xe8, 0xff)
	return application.ThemeSettings{
		LightModeActive:   &application.WindowTheme{BorderColour: line, TitleBarColour: bg, TitleTextColour: text},
		LightModeInactive: &application.WindowTheme{BorderColour: line, TitleBarColour: bg, TitleTextColour: muted},
		LightModeMenuBar: &application.MenuBarTheme{
			Default:  &application.TextTheme{Text: text, Background: bg},
			Hover:    &application.TextTheme{Text: text, Background: hover},
			Selected: &application.TextTheme{Text: text, Background: hover},
		},
	}
}
