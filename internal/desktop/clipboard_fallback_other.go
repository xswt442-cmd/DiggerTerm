//go:build !windows || server

package desktop

import "errors"

func writeClipboardFallback(string) error {
	return errors.New("system clipboard write failed")
}

func readClipboardFallback() (string, error) {
	return "", errors.New("system clipboard read failed")
}
