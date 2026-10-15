package session

import (
	"errors"
	"testing"
	"time"
)

func TestSessionIDsAreScopedToManagerInstance(t *testing.T) {
	profiles := Profiles()
	if len(profiles) == 0 {
		t.Fatal("no local shell profile is available")
	}
	first := NewManager(nil, nil)
	second := NewManager(nil, nil)
	created := make([]struct {
		manager *Manager
		info    Info
	}, 0, 2)
	t.Cleanup(func() {
		for _, item := range created {
			_ = item.manager.Terminate(item.info.ID)
		}
		for _, item := range created {
			waitForManagerExit(t, item.manager, item.info.ID)
		}
	})
	for _, manager := range []*Manager{first, second} {
		info, err := manager.Create(profiles[0].ID, 80, 24)
		if err != nil {
			t.Fatal(err)
		}
		created = append(created, struct {
			manager *Manager
			info    Info
		}{manager, info})
	}
	if created[0].info.ID == created[1].info.ID {
		t.Fatalf("different manager instances reused session ID %q", created[0].info.ID)
	}
	if _, err := second.Snapshot(created[0].info.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("stale ID snapshot error = %v, want ErrNotFound", err)
	}
	if err := second.Write(created[0].info.ID, "x"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("stale ID write error = %v, want ErrNotFound", err)
	}
}

func waitForManagerExit(t *testing.T, manager *Manager, id string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, info := range manager.List() {
			if info.ID == id && info.Status == "exited" {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Errorf("session %s did not exit during cleanup", id)
}
