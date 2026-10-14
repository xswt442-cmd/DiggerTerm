//go:build linux

package session

import (
	"encoding/base64"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// Runs real GNU nano under the same owned Linux PTY used by the GUI.
// This is an end-to-end test of nano's native control-byte semantics,
// not a replacement for xterm/browser keyboard-event tests.
func TestGNUNanoNativeControlKeys(t *testing.T) {
	if _, err := exec.LookPath("nano"); err != nil {
		t.Skip("GNU nano is not installed")
	}
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("SHELL", "/bin/sh") // deterministic interactive shell, no user prompt plugins
	m := NewManager(nil, nil)
	temp, err := os.CreateTemp("", "diggerterm-nano-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	file := temp.Name()
	_ = temp.Close()
	_ = os.Remove(file)
	defer os.Remove(file)
	info, err := m.Create("shell", 90, 28)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Write(info.ID, "nano -I "+file+"\n"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		for _, item := range m.List() {
			if item.ID == info.ID && item.Status == "running" {
				_ = m.Terminate(info.ID)
			}
		}
	}()
	wait := func(what string, timeout time.Duration, contains func(string) bool) {
		t.Helper()
		deadline := time.Now().Add(timeout)
		for time.Now().Before(deadline) {
			snap, err := m.Snapshot(info.ID)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := base64.StdEncoding.DecodeString(snap.Data)
			if err != nil {
				t.Fatal(err)
			}
			if contains(string(raw)) {
				return
			}
			time.Sleep(80 * time.Millisecond)
		}
		snap, _ := m.Snapshot(info.ID)
		raw, _ := base64.StdEncoding.DecodeString(snap.Data)
		if len(raw) > 1300 {
			raw = raw[len(raw)-1300:]
		}
		t.Fatalf("nano did not %s; snapshot tail: %q", what, raw)
	}
	wait("start", 8*time.Second, func(output string) bool {
		return strings.Contains(output, "GNU nano")
	})
	if err := m.Write(info.ID, "DIGGERTERM_NANO_NATIVE_KEYS"); err != nil {
		t.Fatal(err)
	}
	// xterm cursor-left/right escape sequences must still reach nano.
	if err := m.Write(info.ID, "\x1b[D\x1b[C"); err != nil {
		t.Fatal(err)
	}
	// ^O (write out) followed by Enter confirms the file name.
	if err := m.Write(info.ID, "\x0f"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(350 * time.Millisecond)
	if err := m.Write(info.ID, "\r"); err != nil {
		t.Fatal(err)
	}
	wait("save with Ctrl+O", 8*time.Second, func(string) bool {
		data, err := os.ReadFile(file)
		return err == nil && strings.Contains(string(data), "DIGGERTERM_NANO_NATIVE_KEYS")
	})
	// ^K cut, ^U uncut; save again. ^W search, ^V page down, ^X exit.
	for _, input := range []string{"\x0b", "\x15", "\x0f"} {
		if err := m.Write(info.ID, input); err != nil {
			t.Fatal(err)
		}
		time.Sleep(130 * time.Millisecond)
	}
	if err := m.Write(info.ID, "\r"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(250 * time.Millisecond)
	if err := m.Write(info.ID, "\x17"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	if err := m.Write(info.ID, "NANO_NATIVE_KEYS\r"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(250 * time.Millisecond)
	if err := m.Write(info.ID, "\x16"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	// Nano uses ^C for cursor position; unlike a shell it must not quit nano.
	if err := m.Write(info.ID, "\x03"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(150 * time.Millisecond)
	if err := m.Write(info.ID, "\x18"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	if err := m.Write(info.ID, "echo DIGGERTERM_NANO_EXITED\n"); err != nil {
		t.Fatal(err)
	}
	wait("exit with Ctrl+X and return to the shell", 8*time.Second, func(s string) bool {
		return strings.Count(s, "DIGGERTERM_NANO_EXITED") >= 2
	})
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "DIGGERTERM_NANO_NATIVE_KEYS") {
		t.Fatalf("nano did not preserve cut/uncut text after save: %q", data)
	}
}
