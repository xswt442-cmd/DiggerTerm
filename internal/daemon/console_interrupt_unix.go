//go:build !windows

package daemon

func prepareDaemonConsoleInterrupts() error { return nil }
