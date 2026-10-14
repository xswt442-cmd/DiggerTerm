package session

import (
	"errors"
	"testing"
)

func TestInvalidSessionOperations(t *testing.T) {
	m := NewManager(nil, nil)
	if _, err := m.Snapshot("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Snapshot error = %v, want ErrNotFound", err)
	}
	if err := m.Write("missing", "x"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Write error = %v, want ErrNotFound", err)
	}
	if err := m.Terminate("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Terminate error = %v, want ErrNotFound", err)
	}
	if err := m.Forget("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Forget error = %v, want ErrNotFound", err)
	}
	if _, err := m.Create("unknown", 80, 24); err == nil {
		t.Fatal("unknown profile should be rejected")
	}
}

func TestProfileAndSizeValidation(t *testing.T) {
	profiles := Profiles()
	if len(profiles) == 0 {
		t.Fatal("expected at least one local profile")
	}
	if _, err := NewManager(nil, nil).Create(profiles[0].ID, 0, 24); err == nil {
		t.Fatal("zero columns should be rejected")
	}
	if validSize(65536, 24) || validSize(80, -1) || !validSize(80, 24) {
		t.Fatal("unexpected terminal size validation result")
	}
}
