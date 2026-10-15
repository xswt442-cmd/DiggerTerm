//go:build windows

package session

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGitBashDiscoveryRequiresInstallationMarkers(t *testing.T) {
	empty := t.TempDir()
	if path, ok := gitBashInRoots([]string{empty}); ok {
		t.Fatalf("accepted unmarked directory as Git for Windows: %q", path)
	}
	if path, ok := gitBashInRoots([]string{"Git"}); ok {
		t.Fatalf("accepted a relative path as a Git for Windows root: %q", path)
	}
	if path, ok := gitBashInRoots([]string{filepath.Join(os.Getenv("SystemRoot"), "System32")}); ok {
		t.Fatalf("accepted System32 shell as Git Bash: %q", path)
	}
}

func TestGitBashProfileUsesDetectedAbsoluteExecutable(t *testing.T) {
	path, available := windowsGitBashPath()
	spec, supported := profileCommand("bash")
	if available != supported {
		t.Fatalf("profile availability %v differs from detected executable %q", supported, path)
	}
	if !available {
		for _, profile := range Profiles() {
			if profile.ID == "bash" {
				t.Fatal("listed Git Bash without a validated installation")
			}
		}
		return
	}
	if !filepath.IsAbs(path) || spec.path != path {
		t.Fatalf("Git Bash executable must be the detected absolute path: path=%q spec=%+v", path, spec)
	}
	if strings.ToLower(filepath.Base(filepath.Dir(path))) == "system32" || strings.EqualFold(path, `C:\Windows\System32\bash.exe`) {
		t.Fatalf("selected Windows subsystem shell instead of Git Bash: %q", path)
	}
	if !strings.EqualFold(filepath.Base(path), "bash.exe") || strings.Join(spec.args, " ") != "--login -i" {
		t.Fatalf("unexpected Git Bash command: %+v", spec)
	}
	if len(spec.env) != 1 || spec.env[0] != "CHERE_INVOKING=1" {
		t.Fatalf("Git Bash must preserve the requested working directory: env=%v", spec.env)
	}
}

func TestGitBashConPTYRunsInConfiguredDirectory(t *testing.T) {
	if _, ok := windowsGitBashPath(); !ok {
		t.Skip("Git for Windows is not installed")
	}
	tempRoot := t.TempDir()
	home := filepath.Join(tempRoot, "isolated home")
	if err := os.Mkdir(home, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	root := filepath.Join(tempRoot, "bash workspace with spaces")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	m := NewManager(nil, nil)
	info, err := m.CreateLocal("bash", root, 90, 26)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		for _, item := range m.List() {
			if item.ID == info.ID && item.Status == "running" {
				_ = m.Terminate(info.ID)
			}
		}
	}()
	if err := m.Write(info.ID, "marker='DIGGERTERM_BASH_EXPANDED'; printf '%s|%s\\n' \"$marker\" \"$(pwd -W)\"\r\n"); err != nil {
		t.Fatal(err)
	}
	want := strings.ToLower("DIGGERTERM_BASH_EXPANDED|" + strings.ReplaceAll(root, `\`, "/"))
	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) {
		replay, err := m.Snapshot(info.ID)
		if err != nil {
			t.Fatal(err)
		}
		data, err := base64.StdEncoding.DecodeString(replay.Data)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(strings.ToLower(string(data)), want) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	replay, err := m.Snapshot(info.ID)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := base64.StdEncoding.DecodeString(replay.Data)
	t.Fatalf("Git Bash did not expand the marker and report configured cwd %q (replay bytes=%d offset=%d)", root, len(data), replay.Offset)
}
