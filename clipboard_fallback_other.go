//go:build !windows || server

package main

import "errors"

func writeClipboardFallback(string) error {
	return errors.New("system clipboard write failed")
}

func readClipboardFallback() (string, error) {
	return "", errors.New("system clipboard read failed")
}
