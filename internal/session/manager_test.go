package session

import (
	"errors"
	"strings"
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
	if _, err := m.Rename("missing", "Valid name"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Rename error = %v, want ErrNotFound", err)
	}
	if _, err := m.Create("unknown", 80, 24); err == nil {
		t.Fatal("unknown profile should be rejected")
	}
}

func TestRenameInputValidation(t *testing.T) {
	m := NewManager(nil, nil)
	for _, name := range []string{"", "   ", "bad\nname", "tab\tname", "spoof\u202Eexe", strings.Repeat("x", 65)} {
		if _, err := m.Rename("missing", name); err == nil || errors.Is(err, ErrNotFound) {
			t.Errorf("should reject unsafe rename %q before lookup, err=%v", name, err)
		}
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

func TestListIncludesCurrentOutputOffsetWithoutMutatingSessionInfo(t *testing.T) {
	m := NewManager(nil, nil)
	session := &managed{info: Info{ID: "session-1", Status: "running"}, offset: 42}
	m.sessions[session.info.ID] = session
	m.order = append(m.order, session.info.ID)

	listed := m.List()
	if len(listed) != 1 || listed[0].OutputOffset != 42 {
		t.Fatalf("List() = %#v, want current offset 42", listed)
	}
	if session.info.OutputOffset != 0 {
		t.Fatalf("List mutated stored session metadata: %#v", session.info)
	}
}
