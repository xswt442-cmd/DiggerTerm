//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

func adminLaunchParameters(args []string) (string, error) {
	if len(args) != 0 && (len(args) != 2 || args[0] != "--directory") {
		return "", fmt.Errorf("invalid administrator launch arguments")
	}
	if len(args) == 2 {
		if !filepath.IsAbs(args[1]) {
			return "", fmt.Errorf("startup directory must be absolute")
		}
		info, err := os.Stat(args[1])
		if err != nil {
			return "", err
		}
		if !info.IsDir() {
			return "", fmt.Errorf("startup path is not a directory")
		}
	}
	var parts []string
	for _, arg := range args {
		parts = append(parts, windows.EscapeArg(arg))
	}
	return strings.Join(parts, " "), nil
}

func requestAdminElevation(args []string) error {
	arguments, err := adminLaunchParameters(args)
	if err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	verb, _ := windows.UTF16PtrFromString("runas")
	target, err := windows.UTF16PtrFromString(exe)
	if err != nil {
		return err
	}
	params, err := windows.UTF16PtrFromString(arguments)
	if err != nil {
		return err
	}
	return windows.ShellExecute(0, verb, target, params, nil, windows.SW_SHOWNORMAL)
}
