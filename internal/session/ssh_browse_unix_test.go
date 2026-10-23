//go:build !windows

package session

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSSHBrowsePOSIXShellRoundTrip(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"folder", "spaces here", "O'Brien", ".hidden"} {
		if err := os.Mkdir(filepath.Join(root, name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "ordinary-file"), []byte("content"), 0600); err != nil {
		t.Fatal(err)
	}
	script, err := sshBrowseCommand(root)
	if err != nil {
		t.Fatal(err)
	}
	result, err := exec.Command("sh", "-c", script).Output()
	if err != nil {
		t.Fatal(err)
	}
	listing, err := parseSSHDirectoryListing(result)
	if err != nil {
		t.Fatal(err)
	}
	if listing.Path != root || len(listing.Directories) != 4 {
		t.Fatalf("listing=%+v", listing)
	}
}
