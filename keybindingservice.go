package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf8"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const maxKeybindingsFileSize = 32 * 1024

var keybindingScopes = map[string]string{
	"copySelection":  "terminal",
	"pasteClipboard": "terminal",
	"toggleCopyMode": "terminal",
	"splitRight":     "workspace",
	"splitDown":      "workspace",
	"focusNextPane":  "workspace",
	"closePane":      "workspace",
	"openSettings":   "workspace",
}

type KeybindingService struct {
	homeDirectory func() (string, error)
}

type keybindingsDocument struct {
	Version  int             `json:"version"`
	Bindings json.RawMessage `json:"bindings"`
}

type keybindingRecord struct {
	Action string   `json:"action"`
	Scope  string   `json:"scope"`
	Keys   []string `json:"keys"`
}

func NewKeybindingService() *KeybindingService {
	return &KeybindingService{homeDirectory: os.UserHomeDir}
}

func (s *KeybindingService) KeybindingsPath() (string, error) {
	home, err := s.homeDirectory()
	if err != nil {
		return "", fmt.Errorf("find user home directory: %w", err)
	}
	if !filepath.IsAbs(home) {
		return "", errors.New("user home directory must be absolute")
	}
	return filepath.Join(home, "DiggerTerm", "keybindings.json"), nil
}

func (s *KeybindingService) ReadKeybindings() (string, error) {
	root, missing, err := s.openConfigRoot(false)
	if err != nil {
		return "", err
	}
	if missing {
		return "", nil
	}
	defer root.Close()
	info, err := root.Lstat("keybindings.json")
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("inspect keybindings.json: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("keybindings.json must be a regular file; symlinks are not allowed")
	}
	if info.Size() > maxKeybindingsFileSize {
		return "", fmt.Errorf("keybindings.json exceeds %d KiB", maxKeybindingsFileSize/1024)
	}
	file, err := root.Open("keybindings.json")
	if err != nil {
		return "", fmt.Errorf("open keybindings.json: %w", err)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxKeybindingsFileSize+1))
	if err != nil {
		return "", fmt.Errorf("read keybindings.json: %w", err)
	}
	if len(data) > maxKeybindingsFileSize {
		return "", fmt.Errorf("keybindings.json exceeds %d KiB", maxKeybindingsFileSize/1024)
	}
	if err := validateKeybindings(data); err != nil {
		return "", err
	}
	return string(data), nil
}

// OpenKeybindings opens only the fixed GUI configuration file on explicit request.
func (s *KeybindingService) OpenKeybindings() error {
	root, missing, err := s.openConfigRoot(false)
	if err != nil {
		return err
	}
	if missing {
		return errors.New("save keybindings before opening the file")
	}
	defer root.Close()
	info, err := root.Lstat("keybindings.json")
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("keybindings.json must be a regular file")
	}
	path, err := s.KeybindingsPath()
	if err != nil {
		return err
	}
	return application.Get().Browser.OpenFile(path)
}

func (s *KeybindingService) SaveKeybindings(content string) error {
	data := []byte(content)
	if len(data) > maxKeybindingsFileSize {
		return fmt.Errorf("keybindings.json exceeds %d KiB", maxKeybindingsFileSize/1024)
	}
	if err := validateKeybindings(data); err != nil {
		return err
	}
	root, _, err := s.openConfigRoot(true)
	if err != nil {
		return err
	}
	defer root.Close()
	if info, err := root.Lstat("keybindings.json"); err == nil {
		if !info.Mode().IsRegular() {
			return errors.New("keybindings.json must be a regular file; symlinks are not allowed")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect keybindings.json: %w", err)
	}

	var temporary string
	var file *os.File
	for attempt := 0; attempt < 8; attempt++ {
		var random [12]byte
		if _, err := rand.Read(random[:]); err != nil {
			return fmt.Errorf("create temporary keybindings name: %w", err)
		}
		temporary = ".keybindings-" + hex.EncodeToString(random[:]) + ".tmp"
		file, err = root.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			break
		}
		if !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("create temporary keybindings file: %w", err)
		}
	}
	if file == nil {
		return errors.New("could not allocate a temporary keybindings file")
	}
	defer root.Remove(temporary)
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return fmt.Errorf("write temporary keybindings file: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync temporary keybindings file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close temporary keybindings file: %w", err)
	}
	if err := root.Rename(temporary, "keybindings.json"); err != nil {
		return fmt.Errorf("replace keybindings.json: %w", err)
	}
	return nil
}

func (s *KeybindingService) openConfigRoot(create bool) (*os.Root, bool, error) {
	home, err := s.homeDirectory()
	if err != nil {
		return nil, false, fmt.Errorf("find user home directory: %w", err)
	}
	if !filepath.IsAbs(home) {
		return nil, false, errors.New("user home directory must be absolute")
	}
	root, err := os.OpenRoot(home)
	if err != nil {
		return nil, false, fmt.Errorf("open user home directory: %w", err)
	}
	configRoot, missing, err := openDirectory(root, "DiggerTerm", create)
	_ = root.Close()
	if err != nil || missing {
		return nil, missing, err
	}
	return configRoot, false, nil
}

func openDirectory(parent *os.Root, name string, create bool) (*os.Root, bool, error) {
	info, err := parent.Lstat(name)
	if errors.Is(err, os.ErrNotExist) && create {
		if err = parent.Mkdir(name, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
			return nil, false, fmt.Errorf("create %s directory: %w", name, err)
		}
		info, err = parent.Lstat(name)
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil, true, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("inspect %s directory: %w", name, err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, false, fmt.Errorf("%s must be a directory; symlinks are not allowed", name)
	}
	child, err := parent.OpenRoot(name)
	if err != nil {
		return nil, false, fmt.Errorf("open %s directory: %w", name, err)
	}
	return child, false, nil
}

func validateKeybindings(data []byte) error {
	if len(data) > maxKeybindingsFileSize {
		return fmt.Errorf("keybindings.json exceeds %d KiB", maxKeybindingsFileSize/1024)
	}
	if !utf8.Valid(data) {
		return errors.New("keybindings JSON must be valid UTF-8")
	}
	if !json.Valid(data) {
		return errors.New("invalid keybindings JSON")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var document keybindingsDocument
	if err := decoder.Decode(&document); err != nil {
		return fmt.Errorf("invalid keybindings: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return errors.New("invalid keybindings JSON: trailing data")
	}
	if document.Version != 1 {
		return errors.New("version must be 1")
	}
	bindingsJSON := bytes.TrimSpace(document.Bindings)
	if len(bindingsJSON) == 0 || bindingsJSON[0] != '[' {
		return errors.New("bindings must be an array")
	}
	var bindings []keybindingRecord
	bindingsDecoder := json.NewDecoder(bytes.NewReader(bindingsJSON))
	bindingsDecoder.DisallowUnknownFields()
	if err := bindingsDecoder.Decode(&bindings); err != nil {
		return fmt.Errorf("invalid bindings: %w", err)
	}
	seenActions := make(map[string]bool)
	seenKeys := make(map[string]bool)
	for _, binding := range bindings {
		scope, ok := keybindingScopes[binding.Action]
		if !ok {
			return fmt.Errorf("unknown action %q", binding.Action)
		}
		if seenActions[binding.Action] {
			return fmt.Errorf("duplicate action %s", binding.Action)
		}
		seenActions[binding.Action] = true
		if binding.Scope != scope {
			return fmt.Errorf("%s must use scope %s", binding.Action, scope)
		}
		if binding.Keys == nil {
			return fmt.Errorf("%s.keys must be an array of shortcuts", binding.Action)
		}
		if len(binding.Keys) > 16 {
			return fmt.Errorf("%s may have at most 16 shortcuts", binding.Action)
		}
		for _, raw := range binding.Keys {
			key, err := normalizeKeybindingChord(raw)
			if err != nil {
				return fmt.Errorf("%s: %w", binding.Action, err)
			}
			if seenKeys[key] {
				return fmt.Errorf("shortcut %s is bound more than once", key)
			}
			seenKeys[key] = true
		}
	}
	return validateEffectiveKeybindingConflicts(bindings)
}

func normalizeKeybindingChord(value string) (string, error) {
	parts := strings.Split(value, "+")
	modifiers := map[string]bool{}
	var key string
	for _, raw := range parts {
		part := strings.TrimSpace(raw)
		if part == "" {
			return "", fmt.Errorf("invalid shortcut %q", value)
		}
		var modifier string
		switch strings.ToLower(part) {
		case "ctrl", "control":
			modifier = "Ctrl"
		case "meta", "cmd", "command":
			modifier = "Meta"
		case "alt", "option":
			modifier = "Alt"
		case "shift":
			modifier = "Shift"
		}
		if modifier != "" {
			if modifiers[modifier] {
				return "", fmt.Errorf("duplicate modifier in %q", value)
			}
			modifiers[modifier] = true
			continue
		}
		if key != "" {
			return "", fmt.Errorf("shortcut must contain one key: %q", value)
		}
		key = canonicalKeybindingKey(part)
		if key == "" {
			return "", fmt.Errorf("unsupported key %q", part)
		}
	}
	if key == "" || len(modifiers) == 0 {
		return "", fmt.Errorf("shortcut needs one key and a modifier: %q", value)
	}
	if modifiers["Ctrl"] && modifiers["Alt"] {
		return "", errors.New("Ctrl+Alt shortcuts are ambiguous with AltGr")
	}
	ordered := make([]string, 0, 5)
	for _, modifier := range []string{"Ctrl", "Meta", "Alt", "Shift"} {
		if modifiers[modifier] {
			ordered = append(ordered, modifier)
		}
	}
	ordered = append(ordered, key)
	chord := strings.Join(ordered, "+")
	if err := validateKeybindingChord(chord, key, modifiers); err != nil {
		return "", err
	}
	return chord, nil
}

func canonicalKeybindingKey(value string) string {
	if len(value) == 1 {
		if value[0] >= 'a' && value[0] <= 'z' {
			return strings.ToUpper(value)
		}
		if value[0] >= 'A' && value[0] <= 'Z' {
			return value
		}
	}
	keys := map[string]string{
		",": "Comma", ".": "Period", "/": "Slash", "\\": "Backslash", "[": "BracketLeft", "]": "BracketRight",
		";": "Semicolon", "'": "Quote", "`": "Backquote", "-": "Minus", "=": "Equal",
		"insert": "Insert", "arrowleft": "ArrowLeft", "arrowright": "ArrowRight", "arrowup": "ArrowUp", "arrowdown": "ArrowDown",
		"left": "ArrowLeft", "right": "ArrowRight", "up": "ArrowUp", "down": "ArrowDown",
		"comma": "Comma", "period": "Period", "slash": "Slash", "backslash": "Backslash", "bracketleft": "BracketLeft", "bracketright": "BracketRight",
		"semicolon": "Semicolon", "quote": "Quote", "backquote": "Backquote", "minus": "Minus", "equal": "Equal", "escape": "Escape", "tab": "Tab", "enter": "Enter",
		"home": "Home", "end": "End", "pageup": "PageUp", "pagedown": "PageDown", "delete": "Delete", "backspace": "Backspace",
	}
	if canonical, ok := keys[strings.ToLower(value)]; ok {
		return canonical
	}
	if len(value) >= 2 && len(value) <= 3 && (value[0] == 'f' || value[0] == 'F') {
		var number int
		if _, err := fmt.Sscanf(value[1:], "%d", &number); err == nil && number >= 1 && number <= 24 && fmt.Sprintf("F%d", number) == strings.ToUpper(value) {
			return strings.ToUpper(value)
		}
	}
	return ""
}

func validateKeybindingChord(chord, key string, modifiers map[string]bool) error {
	ctrlOnly := len(modifiers) == 1 && modifiers["Ctrl"]
	if ctrlOnly && len(key) == 1 && key[0] >= 'A' && key[0] <= 'Z' {
		if key == "C" || key == "V" {
			return fmt.Errorf("Ctrl+%s is reserved for terminal copy/control behavior", key)
		}
		return fmt.Errorf("Ctrl+%s is reserved for terminal or browser behavior", key)
	}
	if modifiers["Alt"] && key == "F4" {
		return errors.New("Alt+F4 is reserved by the operating system")
	}
	if modifiers["Meta"] && key == "Q" {
		return errors.New("Meta+Q is reserved by the operating system")
	}
	if modifiers["Ctrl"] && modifiers["Alt"] && key == "Delete" {
		return errors.New("Ctrl+Alt+Delete is reserved by the operating system")
	}
	if modifiers["Ctrl"] && !modifiers["Shift"] && !modifiers["Meta"] && (key == "R" || key == "W" || key == "L") {
		return fmt.Errorf("Ctrl+%s is reserved by the browser or terminal", key)
	}
	return nil
}

func validateEffectiveKeybindingConflicts(overrides []keybindingRecord) error {
	isMac := runtime.GOOS == "darwin"
	effective := map[string]string{}
	putDefaults := func(action, key string) error {
		chord, err := normalizeKeybindingChord(key)
		if err != nil {
			return err
		}
		if previous, ok := effective[chord]; ok && previous != action {
			return fmt.Errorf("effective shortcut %s conflicts between %s and %s", chord, previous, action)
		}
		effective[chord] = action
		return nil
	}
	for _, definition := range actionListForGo() {
		keys := definition.other
		if isMac {
			keys = definition.mac
		}
		for _, key := range keys {
			if err := putDefaults(definition.id, key); err != nil {
				return err
			}
		}
	}
	for _, binding := range overrides {
		for chord, action := range effective {
			if action == binding.Action {
				delete(effective, chord)
			}
		}
	}
	for _, binding := range overrides {
		for _, raw := range binding.Keys {
			key, err := normalizeKeybindingChord(raw)
			if err != nil {
				return err
			}
			if previous, ok := effective[key]; ok && previous != binding.Action {
				return fmt.Errorf("effective shortcut %s conflicts between %s and %s", key, previous, binding.Action)
			}
			effective[key] = binding.Action
		}
	}
	return nil
}

type keybindingDefaults struct {
	id         string
	other, mac []string
}

func actionListForGo() []keybindingDefaults {
	return []keybindingDefaults{
		{id: "copySelection", other: []string{"Ctrl+Shift+C", "Ctrl+Insert"}, mac: []string{"Meta+C"}},
		{id: "pasteClipboard", other: []string{"Ctrl+Shift+V", "Shift+Insert"}, mac: []string{"Meta+V"}},
		{id: "toggleCopyMode", other: []string{"Ctrl+Shift+M"}, mac: []string{"Ctrl+Shift+M"}},
		{id: "splitRight", other: []string{"Ctrl+Shift+ArrowRight"}, mac: []string{"Ctrl+Shift+ArrowRight"}},
		{id: "splitDown", other: []string{"Ctrl+Shift+ArrowDown"}, mac: []string{"Ctrl+Shift+ArrowDown"}},
		{id: "focusNextPane", other: []string{"Ctrl+Shift+F6"}, mac: []string{"Ctrl+Shift+F6"}},
		{id: "closePane", other: []string{"Ctrl+Shift+F4"}, mac: []string{"Ctrl+Shift+F4"}},
		{id: "openSettings", other: []string{"Ctrl+Shift+Comma"}, mac: []string{"Ctrl+Shift+Comma"}},
	}
}
