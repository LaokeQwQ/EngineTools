package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:            "Engine DJ Unicode Fix Tool",
		Width:            520,
		Height:           680,
		MinWidth:         480,
		MinHeight:        600,
		AssetServer:      &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 0x1a, G: 0x1a, B: 0x2e, A: 255},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent:              false,
			WindowIsTranslucent:                false,
			DisableWindowIcon:                  false,
			WebviewBrowserPath:                 "",
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}