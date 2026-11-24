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

	// Wails only applies a window theme at creation: a theme change in the
	// settings reaches the native chrome on the next launch, while the web
	// content switches immediately.
	theme := app.startupTheme()
	app.window = wapp.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "main",
		Title:            "Verk",
		Width:            1240,
		Height:           840,
		MinWidth:         980,
		MinHeight:        680,
		BackgroundColour: windowBackground(theme),
		URL:              "/",
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
			Theme:       windowsTheme(theme),
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

// startupTheme reads the appearance setting before the window is created.
func (a *App) startupTheme() string {
	if a.store == nil {
		return themeSystem
	}
	settings, err := a.store.getSettings()
	if err != nil {
		return themeSystem
	}
	return settings.Theme
}

func windowsTheme(theme string) application.Theme {
	switch theme {
	case themeLight:
		return application.Light
	case themeDark:
		return application.Dark
	default:
		return application.SystemDefault
	}
}

func macAppearance(theme string) application.MacAppearanceType {
	switch theme {
	case themeLight:
		return application.NSAppearanceNameAqua
	case themeDark:
		return application.NSAppearanceNameDarkAqua
	default:
		return application.DefaultAppearance
	}
}

// windowBackground matches --bg so the window does not flash before the
// webview paints. With the system theme the light colour is the safe guess.
func windowBackground(theme string) application.RGBA {
	if theme == themeDark {
		return application.NewRGB(0x18, 0x17, 0x1b)
	}
	return application.NewRGB(250, 249, 248)
}

// windowsChrome mirrors the frontend tokens in styles.css, for both themes:
// --bg for the title and menu bars, --text and --muted for labels, --line for
// the border and --accent-soft (lavender) for the hovered menu entry.
func windowsChrome() application.ThemeSettings {
	bg := application.NewRGBPtr(0xfb, 0xfa, 0xf8)
	line := application.NewRGBPtr(0xea, 0xe7, 0xe1)
	text := application.NewRGBPtr(0x26, 0x24, 0x2a)
	muted := application.NewRGBPtr(0x8f, 0x8b, 0x92)
	hover := application.NewRGBPtr(0xec, 0xe8, 0xff)

	darkBg := application.NewRGBPtr(0x18, 0x17, 0x1b)
	darkLine := application.NewRGBPtr(0x34, 0x32, 0x3a)
	darkText := application.NewRGBPtr(0xec, 0xeb, 0xef)
	darkMuted := application.NewRGBPtr(0x91, 0x8e, 0x98)
	darkHover := application.NewRGBPtr(0x3a, 0x35, 0x58)
	return application.ThemeSettings{
		LightModeActive:   &application.WindowTheme{BorderColour: line, TitleBarColour: bg, TitleTextColour: text},
		LightModeInactive: &application.WindowTheme{BorderColour: line, TitleBarColour: bg, TitleTextColour: muted},
		LightModeMenuBar: &application.MenuBarTheme{
			Default:  &application.TextTheme{Text: text, Background: bg},
			Hover:    &application.TextTheme{Text: text, Background: hover},
			Selected: &application.TextTheme{Text: text, Background: hover},
		},
		DarkModeActive:   &application.WindowTheme{BorderColour: darkLine, TitleBarColour: darkBg, TitleTextColour: darkText},
		DarkModeInactive: &application.WindowTheme{BorderColour: darkLine, TitleBarColour: darkBg, TitleTextColour: darkMuted},
		DarkModeMenuBar: &application.MenuBarTheme{
			Default:  &application.TextTheme{Text: darkText, Background: darkBg},
			Hover:    &application.TextTheme{Text: darkText, Background: darkHover},
			Selected: &application.TextTheme{Text: darkText, Background: darkHover},
		},
	}
}
