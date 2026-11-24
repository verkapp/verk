//go:build windows

package src

import (
	"log"
	"reflect"
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/w32"
)

func systemPrefersDark() bool {
	return w32.IsCurrentlyDarkMode()
}

// watchSystemTheme follows the Windows light/dark setting while the app
// theme is « Système ». The window is never created in Wails' own system
// mode, so this handler is the only one reacting to the change.
func (a *App) watchSystemTheme() {
	a.wapp.Event.OnApplicationEvent(events.Windows.SystemThemeChanged, func(*application.ApplicationEvent) {
		if a.store == nil {
			return
		}
		settings, err := a.store.getSettings()
		if err != nil || settings.Theme != themeSystem {
			return
		}
		a.refreshNativeTheme()
	})
}

// applyNativeTheme repaints the title bar, the window border, the menu bar
// and the drop-down menus without recreating the window. Wails 3.1 only
// themes a window when it creates it, so this goes through the Win32 helpers
// that Wails exposes in pkg/w32.
func (a *App) applyNativeTheme(dark bool) {
	window := a.window
	application.InvokeSync(func() {
		ptr := window.NativeWindow()
		if ptr == nil {
			return
		}
		hwnd := uintptr(ptr)
		palette := chromeFor(dark)

		// Drop-down menus follow the process-wide app mode, not the window.
		if w32.SetPreferredAppMode != nil {
			mode := w32.PreferredAppModeForceLight
			if dark {
				mode = w32.PreferredAppModeForceDark
			}
			w32.SetPreferredAppMode(mode)
			if w32.FlushMenuThemes != nil {
				w32.FlushMenuThemes()
			}
		}

		// Title bar: dark caption buttons, then the app's own colours.
		w32.SetTheme(hwnd, dark)
		w32.SetTitleBarColour(hwnd, palette.bg)
		w32.SetTitleTextColour(hwnd, palette.text)
		w32.SetBorderColour(hwnd, palette.line)

		// Menu bar: Wails paints it with brushes built once from the theme
		// it keeps on the window, so that theme is replaced as a whole.
		theme := &w32.MenuBarTheme{
			TitleBarBackground:     &palette.bg,
			TitleBarText:           &palette.text,
			MenuBarBackground:      &palette.bg,
			MenuHoverBackground:    &palette.hover,
			MenuHoverText:          &palette.text,
			MenuSelectedBackground: &palette.hover,
			MenuSelectedText:       &palette.text,
		}
		theme.Init()
		if setMenuBarTheme(window, theme) {
			if menu := w32.GetMenu(w32.HWND(hwnd)); menu != 0 {
				theme.SetMenuBackground(menu)
			}
			w32.DrawMenuBar(w32.HWND(hwnd))
		}
		w32.RedrawWindow(w32.HWND(hwnd), nil, 0, w32.RDW_FRAME|w32.RDW_INVALIDATE|w32.RDW_UPDATENOW|w32.RDW_ALLCHILDREN)
	})
}

// setMenuBarTheme swaps the menu bar theme Wails keeps in the private
// windowsWebviewWindow.menubarTheme field (Wails 3.1). If a later Wails
// release renames or retypes the field, nothing is written: the menu bar then
// keeps the colours of the launch theme while the title bar still follows.
func setMenuBarTheme(window *application.WebviewWindow, theme *w32.MenuBarTheme) bool {
	impl := reflect.ValueOf(window).Elem().FieldByName("impl")
	if !impl.IsValid() || impl.Kind() != reflect.Interface || impl.IsNil() {
		return reportMenuBarThemeMiss()
	}
	native := impl.Elem()
	if native.Kind() != reflect.Pointer || native.IsNil() || native.Elem().Kind() != reflect.Struct {
		return reportMenuBarThemeMiss()
	}
	field := native.Elem().FieldByName("menubarTheme")
	if !field.IsValid() || field.Type() != reflect.TypeOf(theme) || !field.CanAddr() {
		return reportMenuBarThemeMiss()
	}
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(reflect.ValueOf(theme))
	return true
}

func reportMenuBarThemeMiss() bool {
	log.Printf("thème: la barre de menus Wails n'a pas pu être mise à jour, elle suivra au prochain lancement")
	return false
}

func chromeFor(dark bool) chromePalette {
	if dark {
		return darkChrome
	}
	return lightChrome
}
