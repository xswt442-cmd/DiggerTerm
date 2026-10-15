package main

import (
	"errors"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// ClipboardService checks the native result, which the runtime JS bridge omits.
// Clipboard contents never travel through the session daemon.
type ClipboardService struct {
	app *application.App
}

func (s *ClipboardService) WriteText(text string) error {
	if !s.app.Clipboard.SetText(text) {
		return errors.New("system clipboard write failed")
	}
	return nil
}

func (s *ClipboardService) ReadText() (string, error) {
	text, ok := s.app.Clipboard.Text()
	if !ok {
		return "", errors.New("system clipboard read failed")
	}
	return text, nil
}
