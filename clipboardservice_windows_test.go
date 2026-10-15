//go:build windows && !server

package main

import (
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// This opt-in test replaces the current system clipboard with test-only text.
// Run it only in an interactive Windows desktop session:
// Set DIGGERTERM_TEST_NATIVE_CLIPBOARD=1, then run:
// go test -run TestNativeClipboardIntegration -timeout 30s
func TestNativeClipboardIntegration(t *testing.T) {
	if os.Getenv("DIGGERTERM_TEST_NATIVE_CLIPBOARD") != "1" {
		t.Skip("set DIGGERTERM_TEST_NATIVE_CLIPBOARD=1 to replace the system clipboard with test markers")
	}
	powershell, err := exec.LookPath("powershell.exe")
	if err != nil {
		t.Fatal("powershell.exe is required for the cross-process clipboard check")
	}

	const written = "DiggerTerm 原生剪贴板测试\nwrapped line: 窗口与终端\n" +
		"long marker: 0123456789 abcdefghijklmnopqrstuvwxyz 这是仅用于测试的文本\n"
	const pasted = "DiggerTerm paste marker: 粘贴测试"

	app := application.New(application.Options{Name: "DiggerTerm native clipboard integration test"})
	service := &ClipboardService{app: app}
	result := make(chan error, 1)
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		go func() {
			if err := service.WriteText(written); err != nil {
				result <- err
				app.Quit()
				return
			}

			check := exec.Command(powershell, "-NoProfile", "-NonInteractive", "-Command",
				"if ((Get-Clipboard -Raw) -ceq $env:DIGGERTERM_EXPECTED_CLIPBOARD) { exit 0 }; exit 1")
			check.Env = append(os.Environ(), "DIGGERTERM_EXPECTED_CLIPBOARD="+written)
			if err := check.Run(); err != nil {
				result <- err
				app.Quit()
				return
			}

			set := exec.Command(powershell, "-NoProfile", "-NonInteractive", "-Command",
				"Set-Clipboard -Value $env:DIGGERTERM_TEST_PASTE")
			set.Env = append(os.Environ(), "DIGGERTERM_TEST_PASTE="+pasted)
			if err := set.Run(); err != nil {
				result <- err
				app.Quit()
				return
			}
			got, err := service.ReadText()
			if err == nil && got != pasted {
				err = errors.New("native clipboard read did not match the test marker")
			}
			result <- err
			app.Quit()
		}()
	})

	runDone := make(chan error, 1)
	go func() { runDone <- app.Run() }()
	t.Cleanup(func() {
		app.Quit()
		select {
		case err := <-runDone:
			if err != nil {
				t.Errorf("Wails application stopped with error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("Wails clipboard test application did not stop")
		}
	})

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("native clipboard integration failed: %v", err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("native clipboard integration timed out")
	}
}
