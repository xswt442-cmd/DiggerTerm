//go:build windows && !server

package desktop

import (
	"context"
	"fmt"

	"golang.design/x/clipboard"
)

func writeClipboardFallback(text string) error {
	if err := clipboard.Init(); err != nil {
		return fmt.Errorf("initialize Windows clipboard: %w", err)
	}
	if _, err := clipboard.Write(context.Background(), clipboard.FmtText, []byte(text)); err != nil {
		return fmt.Errorf("system clipboard write failed: %w", err)
	}
	return nil
}

func readClipboardFallback() (string, error) {
	if err := clipboard.Init(); err != nil {
		return "", fmt.Errorf("initialize Windows clipboard: %w", err)
	}
	data, err := clipboard.Read(context.Background(), clipboard.FmtText)
	if err != nil {
		return "", fmt.Errorf("system clipboard read failed: %w", err)
	}
	return string(data), nil
}
