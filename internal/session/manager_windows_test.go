package session

import (
	"encoding/base64"
	"regexp"
	"testing"
	"time"
)

// Exercises the real Windows ConPTY backend, not a fake in-memory terminal.
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
