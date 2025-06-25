package main

import (
	"context"
	"embed"
	"fmt"
	"os"

	"github.com/javi11/postie/internal/backend"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/build
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

func main() {
	// Create an instance of the app structure
	app := backend.NewApp()

	// Variable to store the context from OnStartup
	var appCtx context.Context

	// Create the menu
	appMenu := menu.NewMenu()

	// File menu
	fileMenu := appMenu.AddSubmenu("File")
	fileMenu.AddText("Settings", keys.Control("comma"), func(_ *menu.CallbackData) {
		app.NavigateToSettings()
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("Quit", keys.Control("q"), func(_ *menu.CallbackData) {
		runtime.Quit(context.Background())
	})

	// Edit menu
	editMenu := appMenu.AddSubmenu("Edit")
	editMenu.AddText("Undo", keys.CmdOrCtrl("z"), func(_ *menu.CallbackData) {
		if appCtx != nil {
			runtime.EventsEmit(appCtx, "menu-undo")
		}
	})
	editMenu.AddText("Redo", keys.CmdOrCtrl("y"), func(_ *menu.CallbackData) {
		if appCtx != nil {
			runtime.EventsEmit(appCtx, "menu-redo")
		}
	})
	editMenu.AddSeparator()
	editMenu.AddText("Cut", keys.CmdOrCtrl("x"), func(_ *menu.CallbackData) {
		if appCtx != nil {
			runtime.EventsEmit(appCtx, "menu-cut")
		}
	})
	editMenu.AddText("Copy", keys.CmdOrCtrl("c"), func(_ *menu.CallbackData) {
		if appCtx != nil {
			runtime.EventsEmit(appCtx, "menu-copy")
		}
	})
	editMenu.AddText("Paste", keys.CmdOrCtrl("v"), func(_ *menu.CallbackData) {
		if appCtx != nil {
			runtime.EventsEmit(appCtx, "menu-paste")
		}
	})
	editMenu.AddSeparator()
	editMenu.AddText("Select All", keys.CmdOrCtrl("a"), func(_ *menu.CallbackData) {
		if appCtx != nil {
			runtime.EventsEmit(appCtx, "menu-select-all")
		}
	})

	// View menu
	viewMenu := appMenu.AddSubmenu("View")
	viewMenu.AddText("Dashboard", keys.Control("1"), func(_ *menu.CallbackData) {
		app.NavigateToDashboard()
	})
	viewMenu.AddText("Settings", keys.Control("2"), func(_ *menu.CallbackData) {
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
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "postie",
		},
		EnableDefaultContextMenu: true,
		Menu:                     appMenu,
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: true,
		},
		OnStartup: func(ctx context.Context) {
			appCtx = ctx
			app.Startup(ctx)

			// Set up file drop handler
			runtime.OnFileDrop(ctx, func(x, y int, paths []string) {
				if err := app.HandleDroppedFiles(paths); err != nil {
					fmt.Printf("Error handling dropped files: %v\n", err)
					// Emit error event to frontend for user notification
					runtime.EventsEmit(ctx, "file-drop-error", err.Error())
				} else {
					// Emit success event to frontend
					runtime.EventsEmit(ctx, "file-drop-success", len(paths))
				}
			})
		},
		LogLevelProduction: logger.DEBUG,
		Bind: []interface{}{
			app,
		},
		Mac: &mac.Options{
			About: &mac.AboutInfo{
				Title:   "Postie",
				Message: "© 2025 Postie",
				Icon:    icon,
			},
			Appearance:          mac.NSAppearanceNameAccessibilityHighContrastVibrantDark,
			WindowIsTranslucent: true,
		},
	})

	if err != nil {
		fmt.Printf("Error starting GUI: %v\n", err)
		os.Exit(1)
	}
}
