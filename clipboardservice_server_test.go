//go:build server

package main

import (
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestUnsupportedNativeClipboardReportsFailure(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	_ = listener.Close()
	t.Setenv("WAILS_SERVER_HOST", "127.0.0.1")
	t.Setenv("WAILS_SERVER_PORT", port)
	app := application.New(application.Options{Name: "DiggerTerm clipboard test"})
	done := make(chan error, 1)
	go func() { done <- app.Run() }()
	t.Cleanup(func() {
		app.Quit()
		select {
		case err := <-done:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(5 * time.Second):
			t.Error("clipboard test runtime did not stop")
		}
	})
	client := &http.Client{Timeout: 200 * time.Millisecond}
	deadline := time.Now().Add(5 * time.Second)
	for {
		response, err := client.Get("http://127.0.0.1:" + port + "/health")
		if err == nil {
			_ = response.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("clipboard test runtime did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	service := &ClipboardService{app: app}
	if err := service.WriteText("DIGGERTERM_CLIPBOARD_TEST"); err == nil {
		t.Fatal("unsupported native clipboard must report failure")
	}
	if _, err := service.ReadText(); err == nil {
		t.Fatal("unsupported native clipboard read must report failure")
	}
}
