package session

import (
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// Exercises the real Windows ConPTY backend, not a fake in-memory terminal.
func TestRenameSurvivesManagerListAndStatus(t *testing.T) {
	m := NewManager(nil, nil)
	info, err := m.Create("cmd", 80, 24)
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
	renamed, err := m.Rename(info.ID, "  项目终端 / Dev  ")
	if err != nil {
		t.Fatal(err)
	}
	if renamed.Name != "项目终端 / Dev" || renamed.ID != info.ID || renamed.PID != info.PID {
		t.Fatalf("wrong rename result: %+v", renamed)
	}
	if got := m.List()[0].Name; got != renamed.Name {
		t.Fatalf("lost daemon-owned rename: %q", got)
	}
	if _, err := m.Rename("missing", "External PID"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("external processes must not be renamable: %v", err)
	}
}

func TestLocalShellWorkingDirectory(t *testing.T) {
	m := NewManager(nil, nil)
	root := filepath.Join(t.TempDir(), "workspace with spaces")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	for _, profile := range []string{"cmd", "powershell"} {
		t.Run(profile, func(t *testing.T) {
			info, err := m.CreateLocal(profile, root, 90, 26)
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
			command := "echo %CD%\r\n"
			if profile == "powershell" {
				command = "(Get-Location).Path\r\n"
			}
			if err := m.Write(info.ID, command); err != nil {
				t.Fatal(err)
			}
			deadline := time.Now().Add(12 * time.Second)
			for time.Now().Before(deadline) {
				output, err := m.Snapshot(info.ID)
				if err != nil {
					t.Fatal(err)
				}
				raw, err := base64.StdEncoding.DecodeString(output.Data)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(strings.ToLower(string(raw)), strings.ToLower(root)) {
					return
				}
				time.Sleep(65 * time.Millisecond)
			}
			t.Fatalf("%s did not start in configured directory %q", profile, root)
		})
	}
	for _, tc := range []struct{ profile, directory string }{
		{"cmd", "relative"},
		{"powershell", filepath.Join(root, "does-not-exist")},
		{"cmd", filepath.Join(root, "file.txt")},
		{"wsl", root},
		{"ssh", root},
	} {
		if tc.directory == filepath.Join(root, "file.txt") {
			if err := os.WriteFile(tc.directory, []byte("data"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := m.CreateLocal(tc.profile, tc.directory, 80, 24); err == nil {
			t.Errorf("accepted invalid cwd: %q %q", tc.profile, tc.directory)
		}
	}
}

func TestWindowsConPTYLifecycle(t *testing.T) {
	m := NewManager(nil, nil)
	info, err := m.Create("cmd", 80, 25)
	if err != nil {
		t.Fatalf("Create cmd ConPTY: %v", err)
	}
	if info.PID <= 0 {
		t.Fatalf("no process ID: %#v", info)
	}
	defer func() {
		current := m.List()
		for _, s := range current {
			if s.ID == info.ID && s.Status == "running" {
				_ = m.Terminate(info.ID)
			}
		}
	}()

	if err := m.Resize(info.ID, 100, 30); err != nil {
		t.Fatalf("Resize: %v", err)
	}
	if err := m.Write(info.ID, "echo DIGGERTERM_CONPTY_%RANDOM%\r\n"); err != nil {
		t.Fatalf("Write: %v", err)
	}

	const sentinel = `DIGGERTERM_CONPTY_[0-9]{1,6}`
	deadline := time.Now().Add(15 * time.Second)
	seen := false
	for time.Now().Before(deadline) {
		reply, err := m.Snapshot(info.ID)
		if err != nil {
			t.Fatalf("Snapshot: %v", err)
		}
		data, err := base64.StdEncoding.DecodeString(reply.Data)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if regexp.MustCompile(sentinel).Match(data) {
			seen = true
			break
		}
		time.Sleep(30 * time.Millisecond)
	}
	if !seen {
		t.Fatal("did not receive command output from ConPTY")
	}
	if err := m.Terminate(info.ID); err != nil {
		t.Fatalf("Terminate: %v", err)
	}
	for time.Now().Before(deadline.Add(3 * time.Second)) {
		items := m.List()
		if len(items) == 1 && items[0].Status == "exited" {
			if err := m.Forget(info.ID); err != nil {
				t.Fatalf("Forget finished session: %v", err)
			}
			if len(m.List()) != 0 {
				t.Fatal("expected no sessions after Forget")
			}
			return
		}
		time.Sleep(30 * time.Millisecond)
	}
	t.Fatal("session failed to reach exited state")
}
