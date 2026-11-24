package src

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

// chromePalette mirrors the frontend tokens in styles.css for the native
// window chrome: --bg for the title and menu bars, --text and --muted for
// labels, --line for the border and the soft lavender accent for the hovered
// menu entry.
type chromePalette struct {
	bg, line, text, muted, hover uint32
}

func colour(r, g, b uint8) uint32 {
	return *application.NewRGBPtr(r, g, b)
}

var (
	lightChrome = chromePalette{
		bg:    colour(0xfb, 0xfa, 0xf8),
		line:  colour(0xea, 0xe7, 0xe1),
		text:  colour(0x26, 0x24, 0x2a),
		muted: colour(0x8f, 0x8b, 0x92),
		hover: colour(0xec, 0xe8, 0xff),
	}
	darkChrome = chromePalette{
		bg:    colour(0x18, 0x17, 0x1b),
		line:  colour(0x34, 0x32, 0x3a),
		text:  colour(0xec, 0xeb, 0xef),
		muted: colour(0x91, 0x8e, 0x98),
		hover: colour(0x3a, 0x35, 0x58),
	}
)

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

// resolveDark turns the setting into the theme to paint, asking the OS when
// the setting follows it.
func resolveDark(theme string) bool {
	switch theme {
	case themeDark:
		return true
	case themeLight:
		return false
	default:
		return systemPrefersDark()
	}
}

func windowsTheme(dark bool) application.Theme {
	if dark {
		return application.Dark
	}
	return application.Light
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
// webview paints.
func windowBackground(dark bool) application.RGBA {
	if dark {
		return application.NewRGB(0x18, 0x17, 0x1b)
	}
	return application.NewRGB(0xfb, 0xfa, 0xf8)
}

// windowsChrome is the theme Wails applies when it creates the window.
// Later changes go through applyNativeTheme.
func windowsChrome() application.ThemeSettings {
	window := func(p chromePalette, active bool) *application.WindowTheme {
		text := p.text
		if !active {
			text = p.muted
		}
		return &application.WindowTheme{BorderColour: &p.line, TitleBarColour: &p.bg, TitleTextColour: &text}
	}
	menu := func(p chromePalette) *application.MenuBarTheme {
		return &application.MenuBarTheme{
			Default:  &application.TextTheme{Text: &p.text, Background: &p.bg},
			Hover:    &application.TextTheme{Text: &p.text, Background: &p.hover},
			Selected: &application.TextTheme{Text: &p.text, Background: &p.hover},
		}
	}
	return application.ThemeSettings{
		LightModeActive:   window(lightChrome, true),
		LightModeInactive: window(lightChrome, false),
		LightModeMenuBar:  menu(lightChrome),
		DarkModeActive:    window(darkChrome, true),
		DarkModeInactive:  window(darkChrome, false),
		DarkModeMenuBar:   menu(darkChrome),
	}
}

// refreshNativeTheme repaints the native chrome after the theme setting
// changed. It is a no-op on platforms whose chrome follows only at launch.
func (a *App) refreshNativeTheme() {
	if a.store == nil || a.window == nil {
		return
	}
	settings, err := a.store.getSettings()
	if err != nil {
		return
	}
	dark := resolveDark(settings.Theme)
	a.window.SetBackgroundColour(windowBackground(dark))
	a.applyNativeTheme(dark)
}
