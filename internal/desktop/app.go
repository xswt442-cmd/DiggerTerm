package desktop

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

func Run(assets fs.FS, trayIcon []byte) {
	runGUI(assets, trayIcon)
}

func runGUI(assets fs.FS, trayIcon []byte) {
	elevated, err := guiIsElevated()
	if err != nil {
		log.Printf("inspect GUI privilege: %v", err)
		return
	}
	broker, primary, err := acquireGUIInstance(os.Args)
	if err != nil {
		log.Printf("GUI launch: %v", err)
		return
	}
	if !primary {
		return
	}
	if broker != nil {
		defer broker.close()
	}
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
	app.RegisterService(application.NewService(NewThemeService()))
	app.RegisterService(application.NewService(NewKeybindingService()))
	app.RegisterService(application.NewService(NewQuickSelectService()))
	app.RegisterService(application.NewService(&ClipboardService{app: app}))

	windowTitle := "DiggerTerm"
	if elevated {
		windowTitle += " (Administrator)"
	}
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            windowTitle,
		Width:            1120,
		Height:           720,
		MinWidth:         760,
		MinHeight:        480,
		BackgroundColour: application.NewRGB(17, 19, 24),
		URL:              "/",
	})
	if broker != nil {
		go func() {
			for args := range broker.launches {
				if err := sessions.enqueueLaunch(args); err != nil {
					log.Printf("GUI directory launch: %v", err)
				}
				window.Show()
				window.Restore()
				window.Focus()
			}
		}()
	}

	tray := app.SystemTray.New()
	tray.SetIcon(trayIcon)
	trayTooltip := func(label string) string {
		if elevated {
			return label + " (Administrator)"
		}
		return label
	}
	tray.SetTooltip(trayTooltip(trayLabelsFor("en").tooltip))
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
	stopAllItem := menu.Add(trayLabelsFor("en").stopAll)
	currentLocale := "en"
	var showStopAllConfirmation func(sessionIDs []string)
	showStopAllConfirmation = func(sessionIDs []string) {
		labels := trayLabelsFor(currentLocale)
		confirmLabel, cancelLabel := stopAllConfirmationButtonLabels(currentLocale, runtime.GOOS)
		question := app.Dialog.Question().
			AttachToWindow(window).
			SetTitle(labels.stopAllTitle).
			SetMessage(fmt.Sprintf(labels.stopAllMessage, len(sessionIDs)))
		cancel := question.AddButton(cancelLabel)
		question.SetDefaultButton(cancel).SetCancelButton(cancel)
		cancel.OnClick(func() {
			_ = runStopAllAfterConfirmation(false, sessionIDs, sessions.stopAllSessionsAndDaemon)
		})
		confirm := question.AddButton(confirmLabel)
		confirm.OnClick(func() {
			if !sessions.tryBeginBulkStop() {
				window.Show()
				window.Focus()
				app.Event.Emit("terminal:quit-blocked", trayLabelsFor(currentLocale).stopAllBusy)
				return
			}
			window.SetEnabled(false)
			quitItem.SetEnabled(false)
			stopItem.SetEnabled(false)
			stopAllItem.SetEnabled(false)
			go func() {
				err := runStopAllAfterConfirmation(true, sessionIDs, sessions.stopAllSessionsAndDaemon)
				sessions.finishBulkStop()
				if err == nil {
					application.InvokeSync(app.Quit)
					return
				}
				var changed *managedSessionStateChangedError
				if errors.As(err, &changed) {
					application.InvokeSync(func() {
						window.SetEnabled(true)
						quitItem.SetEnabled(true)
						stopItem.SetEnabled(true)
						stopAllItem.SetEnabled(true)
						showStopAllConfirmation(changed.currentIDs)
					})
					return
				}
				application.InvokeSync(func() {
					window.SetEnabled(true)
					quitItem.SetEnabled(true)
					stopItem.SetEnabled(true)
					stopAllItem.SetEnabled(true)
					window.Show()
					window.Focus()
					app.Event.Emit("terminal:quit-blocked", err.Error())
				})
			}()
		})
		question.Show()
	}
	stopAllItem.OnClick(func(_ *application.Context) {
		sessionIDs, err := sessions.runningManagedSessionIDs()
		if err != nil {
			window.Show()
			window.Focus()
			app.Event.Emit("terminal:quit-blocked", err.Error())
			return
		}
		window.Show()
		window.Focus()
		showStopAllConfirmation(sessionIDs)
	})
	tray.SetMenu(menu)
	// MenuItem.SetLabel updates the native Windows/Linux/macOS menu in place.
	// Do the update on Wails' main thread; tooltip dispatches there itself.
	sessions.setUILanguage = func(locale string) {
		labels := trayLabelsFor(locale)
		application.InvokeSync(func() {
			currentLocale = locale
			showItem.SetLabel(labels.show)
			quitItem.SetLabel(labels.quit)
			stopItem.SetLabel(labels.stop)
			stopAllItem.SetLabel(labels.stopAll)
		})
		tray.SetTooltip(trayTooltip(labels.tooltip))
	}

	if err := app.Run(); err != nil {
		log.Printf("DiggerTerm GUI: %v", err)
	}
}
