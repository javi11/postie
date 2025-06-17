package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/javi11/postie/internal/backend"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/build
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := backend.NewApp()

	// Create the menu
	appMenu := menu.NewMenu()

	// File menu
	fileMenu := appMenu.AddSubmenu("File")
	fileMenu.AddText("Settings", keys.CmdOrCtrl("comma"), func(_ *menu.CallbackData) {
		app.NavigateToSettings()
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("Quit", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
		runtime.Quit(nil)
	})

	// View menu
	viewMenu := appMenu.AddSubmenu("View")
	viewMenu.AddText("Dashboard", keys.CmdOrCtrl("1"), func(_ *menu.CallbackData) {
		app.NavigateToDashboard()
	})
	viewMenu.AddText("Settings", keys.CmdOrCtrl("2"), func(_ *menu.CallbackData) {
		app.NavigateToSettings()
	})

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "Postie UI",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		DisableResize:    false,
		Menu:             appMenu,
		OnStartup:        app.Startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		fmt.Printf("Error starting GUI: %v\n", err)
		os.Exit(1)
	}
}
