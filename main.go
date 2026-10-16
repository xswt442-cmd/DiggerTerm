package main

import (
	"embed"
	"fmt"
	"log"
	"os"

	"github.com/xswt442-cmd/DiggerTerm/internal/daemon"
	"github.com/xswt442-cmd/DiggerTerm/internal/desktop"
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
	desktop.Run(assets, trayIcon)
}
