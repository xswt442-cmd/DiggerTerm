package session

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestSSHBrowseCommandQuotesAndRejectsUnsafePaths(t *testing.T) {
	command, err := sshBrowseCommand("/srv/O'Brien; $(touch x)")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(command, "'/srv/O'\\''Brien; $(touch x)'") {
		t.Fatalf("path was not escaped: %q", command)
	}
	if !strings.Contains(command, "printf '%s\\0'") {
		t.Fatalf("expected NUL delimiter: %q", command)
	}
	for _, path := range []string{"relative", "a\nb", "/a\x1bb", strings.Repeat("x", 2050)} {
		if _, err := sshBrowseCommand(path); err == nil {
			t.Errorf("accepted unsafe directory %q", path)
		}
	}
	if _, err := sshBrowseCommand(""); err != nil {
		t.Fatalf("home directory: %v", err)
	}
}

func TestParseSSHDirectoryListingHandlesHiddenAndSpecialNames(t *testing.T) {
	payload := []byte("/home/dev\x00.zhidden\x00space name\x00O'Brien\x00projects\x00")
	got, err := parseSSHDirectoryListing(payload)
	if err != nil {
		t.Fatal(err)
	}
	if got.Path != "/home/dev" {
		t.Fatalf("path = %q", got.Path)
	}
	expected := []string{".zhidden", "O'Brien", "projects", "space name"}
	if strings.Join(got.Directories, "|") != strings.Join(expected, "|") {
		t.Fatalf("directories = %v, want %v", got.Directories, expected)
	}
	if got.Truncated {
		t.Fatal("unexpected truncation")
	}
}

func TestParseSSHDirectoryListingRejectsMalformedAndCapsEntries(t *testing.T) {
	for _, raw := range [][]byte{nil, []byte("incomplete"), []byte("relative\x00child\x00")} {
		if _, err := parseSSHDirectoryListing(raw); err == nil {
			t.Errorf("accepted %q", raw)
		}
	}
	b := bytes.NewBufferString("/root\x00")
	for i := 0; i < maxBrowseEntries+10; i++ {
		b.WriteString("folder\x00")
	}
	got, err := parseSSHDirectoryListing(b.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if !got.Truncated || len(got.Directories) != maxBrowseEntries {
		t.Fatalf("expected bounded result: truncated=%v length=%d", got.Truncated, len(got.Directories))
	}
}

func TestBrowseSSHRejectsUnmanagedAndWrongProfile(t *testing.T) {
	m := NewManager(nil, nil)
	if _, err := m.BrowseSSHDirectories(context.Background(), "external", "/"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown session: %v", err)
	}
	m.mu.Lock()
	m.sessions["local"] = &managed{info: Info{ID: "local", Profile: "powershell", Status: "running"}}
	m.mu.Unlock()
	if _, err := m.BrowseSSHDirectories(context.Background(), "local", "/"); err == nil {
		t.Fatal("accepted non-SSH session")
	}
}

func TestBoundedBrowseOutputCapsWithoutAllocatingExcess(t *testing.T) {
	writer := &boundedBrowseOutput{limit: 8}
	if _, err := writer.Write([]byte("12345")); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("6789")); err == nil {
		t.Fatal("expected limit error")
	}
	if !writer.exceeded || writer.buf.Len() != 5 {
		t.Fatalf("writer state %v: %d", writer.exceeded, writer.buf.Len())
	}
}
