package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKeybindingServiceReadAbsentAndSaveValidatedConfig(t *testing.T) {
	home := t.TempDir()
	service := &KeybindingService{homeDirectory: func() (string, error) { return home, nil }}
	content, err := service.ReadKeybindings()
	if err != nil || content != "" {
		t.Fatalf("ReadKeybindings() = %q, %v; want empty config without error", content, err)
	}
	path, err := service.KeybindingsPath()
	if err != nil || path != filepath.Join(home, "user", "DiggerTerm", "keybindings.json") {
		t.Fatalf("KeybindingsPath() = %q, %v", path, err)
	}
	config := `{"version":1,"bindings":[{"action":"splitRight","scope":"workspace","keys":["Ctrl+Shift+F8"]},{"action":"toggleCopyMode","scope":"terminal","keys":[]}]}`
	if err := service.SaveKeybindings(config); err != nil {
		t.Fatalf("SaveKeybindings(): %v", err)
	}
	content, err = service.ReadKeybindings()
	if err != nil || content != config {
		t.Fatalf("ReadKeybindings() = %q, %v", content, err)
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("saved file is not regular: info=%v err=%v", info, err)
	}
	if err := service.SaveKeybindings(`{"version":1,"bindings":[]}`); err != nil {
		t.Fatalf("replace keybindings file atomically: %v", err)
	}
	content, err = service.ReadKeybindings()
	if err != nil || content != `{"version":1,"bindings":[]}` {
		t.Fatalf("read replacement = %q, %v", content, err)
	}
}

func TestKeybindingServiceRejectsInvalidAndUnsafeConfig(t *testing.T) {
	invalid := []struct {
		name    string
		content string
		want    string
	}{
		{"unsupported version", `{"version":2,"bindings":[]}`, "version must be 1"},
		{"missing bindings", `{"version":1}`, "bindings must be an array"},
		{"null bindings", `{"version":1,"bindings":null}`, "bindings must be an array"},
		{"unknown action", `{"version":1,"bindings":[{"action":"run","scope":"workspace","keys":["Ctrl+Shift+R"]}]}`, "unknown action"},
		{"wrong scope", `{"version":1,"bindings":[{"action":"splitRight","scope":"terminal","keys":["Ctrl+Shift+F8"]}]}`, "must use scope workspace"},
		{"shell control", `{"version":1,"bindings":[{"action":"splitRight","scope":"workspace","keys":["Ctrl+C"]}]}`, "reserved"},
		{"reserved operating system key", `{"version":1,"bindings":[{"action":"closePane","scope":"workspace","keys":["Alt+F4"]}]}`, "reserved by the operating system"},
		{"ambiguous AltGr key", `{"version":1,"bindings":[{"action":"splitRight","scope":"workspace","keys":["Ctrl+Alt+Right"]}]}`, "AltGr"},
		{"duplicate action", `{"version":1,"bindings":[{"action":"splitRight","scope":"workspace","keys":[]},{"action":"splitRight","scope":"workspace","keys":[]}]}`, "duplicate action"},
		{"duplicate shortcut", `{"version":1,"bindings":[{"action":"splitRight","scope":"workspace","keys":["Ctrl+Shift+F8"]},{"action":"splitDown","scope":"workspace","keys":["Shift+Control+F8"]}]}`, "bound more than once"},
		{"default conflict", `{"version":1,"bindings":[{"action":"splitRight","scope":"workspace","keys":["Ctrl+Shift+C"]}]}`, "conflicts"},
		{"unknown field", `{"version":1,"bindings":[],"command":"echo unsafe"}`, "unknown field"},
		{"unknown binding field", `{"version":1,"bindings":[{"action":"splitRight","scope":"workspace","keys":[],"command":"echo unsafe"}]}`, "unknown field"},
	}
	for _, testCase := range invalid {
		t.Run(testCase.name, func(t *testing.T) {
			if err := validateKeybindings([]byte(testCase.content)); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("validateKeybindings() error = %v, want substring %q", err, testCase.want)
			}
		})
	}
	service := &KeybindingService{homeDirectory: func() (string, error) { return t.TempDir(), nil }}
	if err := service.SaveKeybindings(`{"version":1,"bindings":[{"action":"closePane","scope":"workspace","keys":["Alt+F4"]}]}`); err == nil {
		t.Fatal("SaveKeybindings accepted unsafe config")
	}
}

func TestValidateKeybindingsAllowsSimultaneousSwap(t *testing.T) {
	config := `{"version":1,"bindings":[{"action":"splitRight","scope":"workspace","keys":["Ctrl+Shift+ArrowDown"]},{"action":"splitDown","scope":"workspace","keys":["Ctrl+Shift+ArrowRight"]},{"action":"openSettings","scope":"workspace","keys":["Ctrl+Shift+,"]}]}`
	if err := validateKeybindings([]byte(config)); err != nil {
		t.Fatal(err)
	}
}

func TestKeybindingServiceRejectsSymlinkedConfigDirectory(t *testing.T) {
	if os.Getenv("GOOS") == "windows" {
		t.Skip("directory symlink creation may require elevated Windows privileges")
	}
	home := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(home, "user")); err != nil {
		t.Skipf("create symlink: %v", err)
	}
	service := &KeybindingService{homeDirectory: func() (string, error) { return home, nil }}
	if err := service.SaveKeybindings(`{"version":1,"bindings":[]}`); err == nil {
		t.Fatal("SaveKeybindings followed a symlinked user directory")
	}
	if _, err := os.Stat(filepath.Join(outside, "DiggerTerm")); !os.IsNotExist(err) {
		t.Fatalf("wrote outside the expected directory: %v", err)
	}
}

func TestValidateKeybindingsRejectsOversizedDocument(t *testing.T) {
	if err := validateKeybindings([]byte(`{"version":1,"bindings":[]}` + strings.Repeat(" ", maxKeybindingsFileSize))); err == nil {
		t.Fatal("SaveKeybindings accepted oversized config")
	}
}
