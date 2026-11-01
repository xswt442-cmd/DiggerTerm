//go:build windows

package desktop

import "github.com/xswt442-cmd/DiggerTerm/internal/winsecurity"

func guiIsElevated() (bool, error) { return winsecurity.Elevated() }
