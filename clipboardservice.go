package main

import (
	"errors"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// ClipboardService checks the native result, which the runtime JS bridge omits.
// Clipboard contents never travel through the session daemon.
type ClipboardService struct {
	app *application.App
}

func (s *ClipboardService) WriteText(text string) error {
	if strings.ContainsRune(text, '\x00') {
		return errors.New("clipboard text cannot contain NUL")
	}
	if !s.app.Clipboard.SetText(text) {
		return writeClipboardFallback(text)
	}
	return nil
}

func (s *ClipboardService) ReadText() (string, error) {
	text, ok := s.app.Clipboard.Text()
	if !ok {
		return readClipboardFallback()
	}
	return text, nil
}
