package src

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

// Run starts the desktop application using the embedded Angular bundle.
func Run(assets embed.FS) {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:            "Todo",
		Width:            1240,
		Height:           840,
		MinWidth:         980,
		MinHeight:        680,
		DisableResize:    false,
		BackgroundColour: &options.RGBA{R: 250, G: 249, B: 248, A: 255},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Menu:              app.applicationMenu(),
		HideWindowOnClose: app.closeToTray(),
		OnStartup:         app.startup,
		OnShutdown:        app.shutdown,
		Bind: []interface{}{
			app,
		},
		Mac: &mac.Options{
			Appearance: mac.NSAppearanceNameAqua,
			About: &mac.AboutInfo{
				Title:   "Todo",
				Message: "Une liste de tâches douce, avec un coach Claude Code.",
			},
		},
		Windows: &windows.Options{
			Theme: windows.Light,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
