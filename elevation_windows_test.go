//go:build windows

package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"golang.org/x/sys/windows"
)

func TestAdminLaunchParameters(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "folder & space")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	params, err := adminLaunchParameters([]string{"--directory", dir})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := windows.DecomposeCommandLine("diggerterm.exe " + params)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(parsed, []string{"diggerterm.exe", "--directory", dir}) {
		t.Fatalf("arguments changed: %q", parsed)
	}
	for _, args := range [][]string{{"--daemon"}, {"--directory", "relative"}, {"--directory", dir, "extra"}} {
		if _, err := adminLaunchParameters(args); err == nil {
			t.Fatalf("accepted invalid arguments %q", args)
		}
	}
}
