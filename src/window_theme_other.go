//go:build !windows

package src

// systemPrefersDark is only needed to pick the Windows chrome: elsewhere the
// window keeps the system appearance by itself (DefaultAppearance on macOS).
func systemPrefersDark() bool {
	return false
}

// watchSystemTheme has nothing to follow outside Windows.
func (a *App) watchSystemTheme() {}

// applyNativeTheme: on macOS the title bar is hidden and the web content
// draws it; the appearance of native menus follows on the next launch.
func (a *App) applyNativeTheme(bool) {}
