//go:build !windows

package session

import (
	"os"
	"os/exec"
	"path/filepath"
)

func bashCommandSpec() (commandSpec, bool) {
	path, ok := unixBashPath()
	if !ok {
		return commandSpec{}, false
	}
	return commandSpec{path: path, args: []string{"-i"}}, true
}

func unixBashPath() (string, bool) {
	for _, path := range []string{"/bin/bash", "/usr/bin/bash"} {
		if executableFile(path) {
			return path, true
		}
	}
	if path, err := exec.LookPath("bash"); err == nil && executableFile(path) {
		absolute, absErr := filepath.Abs(path)
		if absErr == nil {
			return absolute, true
		}
	}
	return "", false
}

func executableFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0111 != 0
}
