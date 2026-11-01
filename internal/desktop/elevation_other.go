//go:build !windows

package desktop

func guiIsElevated() (bool, error) { return false, nil }
