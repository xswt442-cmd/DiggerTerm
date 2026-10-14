package main

import (
	"embed"
	"fmt"
	"log"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/xswt442-cmd/DiggerTerm/internal/daemon"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var trayIcon []byte

func main() {
	// A single executable has exactly two modes: a desktop GUI or the one
	// independent daemon. The daemon never initializes Wails or a tray.
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--daemon":
			if err := daemon.Run(); err != nil {
				log.Fatal(err)
			}
			return
		case "--stop-daemon":
			c, err := daemon.Dial(nil)
			if err != nil {
				log.Fatal(err)
			}
			defer c.Close()
			if err := c.Stop(); err != nil {
				log.Fatal(err)
			}
			fmt.Println("DiggerTerm daemon stopped")
			return
		}
	}
	runGUI()
}

func runGUI() {
	app := application.New(application.Options{
		Name:        "DiggerTerm",
		Description: "A lightweight terminal workspace",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})
	sessions := NewSessionService(app)
	defer sessions.close()
	if sessions.initErr != nil {
		log.Printf("session daemon startup: %v", sessions.initErr)
	}
	app.RegisterService(application.NewService(sessions))

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "DiggerTerm",
		Width:            1120,
		Height:           720,
		MinWidth:         760,
		MinHeight:        480,
		BackgroundColour: application.NewRGB(17, 19, 24),
		URL:              "/",
	})

	tray := app.SystemTray.New()
	tray.SetIcon(trayIcon)
	tray.SetTooltip(trayLabelsFor("en").tooltip)
	tray.OnClick(func() {
		window.Show()
		window.Focus()
	})

	// Hiding and quitting GUI ONLY detach the client. All PTYs reside in
	// the separate daemon, which stays alive even if the GUI exits/crashes.
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		window.Hide()
		e.Cancel()
	})
	menu := app.NewMenu()
	showItem := menu.Add(trayLabelsFor("en").show)
	showItem.OnClick(func(_ *application.Context) {
		window.Show()
		window.Focus()
	})
	menu.AddSeparator()
	quitItem := menu.Add(trayLabelsFor("en").quit)
	quitItem.OnClick(func(_ *application.Context) {
		app.Quit()
	})
	stopItem := menu.Add(trayLabelsFor("en").stop)
	stopItem.OnClick(func(_ *application.Context) {
		if err := sessions.stopDaemon(); err != nil {
			window.Show()
			window.Focus()
			app.Event.Emit("terminal:quit-blocked", err.Error())
			return
		}
		app.Quit()
	})
	tray.SetMenu(menu)
	// MenuItem.SetLabel updates the native Windows/Linux/macOS menu in place.
	// Do the update on Wails' main thread; tooltip dispatches there itself.
	sessions.setUILanguage = func(locale string) {
		labels := trayLabelsFor(locale)
		application.InvokeSync(func() {
			showItem.SetLabel(labels.show)
			quitItem.SetLabel(labels.quit)
			stopItem.SetLabel(labels.stop)
		})
		tray.SetTooltip(labels.tooltip)
	}

	if err := app.Run(); err != nil {
		log.Printf("DiggerTerm GUI: %v", err)
	}
}
