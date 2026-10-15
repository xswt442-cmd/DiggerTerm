//go:build windows

package session

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func bashCommandSpec() (commandSpec, bool) {
	path, ok := windowsGitBashPath()
	if !ok {
		return commandSpec{}, false
	}
	return commandSpec{path: path, args: []string{"--login", "-i"}, env: []string{"CHERE_INVOKING=1"}}, true
}

func windowsGitBashPath() (string, bool) {
	roots := []string{
		filepath.Join(os.Getenv("ProgramFiles"), "Git"),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Git"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Git"),
	}
	roots = append(roots, registeredGitRoots()...)
	if git, err := exec.LookPath("git.exe"); err == nil {
		cmdDir := filepath.Dir(git)
		if strings.EqualFold(filepath.Base(cmdDir), "cmd") {
			roots = append(roots, filepath.Dir(cmdDir))
		}
	}
	return gitBashInRoots(roots)
}

func registeredGitRoots() []string {
	const keyPath = `SOFTWARE\GitForWindows`
	keys := []struct {
		root registry.Key
		path string
	}{
		{registry.CURRENT_USER, keyPath},
		{registry.LOCAL_MACHINE, keyPath},
		{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\GitForWindows`},
	}
	var roots []string
	for _, item := range keys {
		key, err := registry.OpenKey(item.root, item.path, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		value, _, err := key.GetStringValue("InstallPath")
		_ = key.Close()
		if err == nil && value != "" {
			roots = append(roots, value)
		}
	}
	return roots
}

func gitBashInRoots(roots []string) (string, bool) {
	seen := make(map[string]struct{}, len(roots))
	for _, root := range roots {
		if root == "" {
			continue
		}
		if !filepath.IsAbs(root) {
			continue
		}
		root = filepath.Clean(root)
		key := strings.ToLower(root)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if !validGitForWindowsRoot(root) {
			continue
		}
		for _, relative := range []string{filepath.Join("bin", "bash.exe"), filepath.Join("usr", "bin", "bash.exe")} {
			candidate := filepath.Join(root, relative)
			if regularWindowsFile(candidate) {
				return candidate, true
			}
		}
	}
	return "", false
}

func validGitForWindowsRoot(root string) bool {
	if regularWindowsFile(filepath.Join(root, "git-bash.exe")) {
		return true
	}
	return regularWindowsFile(filepath.Join(root, "cmd", "git.exe")) &&
		regularWindowsFile(filepath.Join(root, "usr", "bin", "msys-2.0.dll"))
}

func regularWindowsFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
