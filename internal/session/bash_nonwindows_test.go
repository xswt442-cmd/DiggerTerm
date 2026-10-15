//go:build !windows

package session

import (
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUnixBashProfileUsesExplicitExecutable(t *testing.T) {
	path, available := unixBashPath()
	spec, supported := profileCommand("bash")
	if available != supported {
		t.Fatalf("profile availability %v differs from detected Bash %q", supported, path)
	}
	if !available {
		for _, profile := range Profiles() {
			if profile.ID == "bash" {
				t.Fatal("listed Bash without an executable")
			}
		}
		return
	}
	resolved, err := exec.LookPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(spec.path) != filepath.Clean(resolved) {
		t.Fatalf("Bash command does not use detected executable: %+v", spec)
	}
	if !filepath.IsAbs(spec.path) {
		t.Fatalf("Bash command path must be absolute: %q", spec.path)
	}
	if len(spec.args) != 1 || spec.args[0] != "-i" {
		t.Fatalf("unexpected Bash arguments: %v", spec.args)
	}
	if _, err := os.Stat(spec.path); err != nil {
		t.Fatalf("Bash executable is unavailable: %v", err)
	}
}

func TestUnixBashPTYUsesConfiguredDirectory(t *testing.T) {
	if _, ok := unixBashPath(); !ok {
		t.Skip("Bash is not installed")
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
	root, err := filepath.Abs(root)
	if err != nil {
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
	if err := m.Write(info.ID, "marker='DIGGERTERM_BASH_EXPANDED'; printf '%s|%s\\n' \"$marker\" \"$PWD\"\r"); err != nil {
		t.Fatal(err)
	}
	want := "DIGGERTERM_BASH_EXPANDED|" + root
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		replay, err := m.Snapshot(info.ID)
		if err != nil {
			t.Fatal(err)
		}
		data, err := base64.StdEncoding.DecodeString(replay.Data)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), want) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	replay, err := m.Snapshot(info.ID)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := base64.StdEncoding.DecodeString(replay.Data)
	t.Fatalf("Bash did not expand the marker and report configured cwd %q (replay bytes=%d offset=%d)", root, len(data), replay.Offset)
}
