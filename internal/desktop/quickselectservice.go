package desktop

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	quickSelectFileName        = "quick-select.json"
	maxQuickSelectBytes        = 64 * 1024
	maxQuickSelectRules        = 32
	maxQuickSelectNameRunes    = 32
	maxQuickSelectPatternRunes = 512
)

var quickSelectTypes = map[string]bool{"url": true, "ip": true, "path": true, "gitHash": true}

type QuickSelectRule struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Pattern string `json:"pattern"`
	Enabled bool   `json:"enabled"`
}

type QuickSelectConfig struct {
	Version      int               `json:"version"`
	EnabledTypes []string          `json:"enabledTypes"`
	Rules        []QuickSelectRule `json:"rules"`
}

type QuickSelectService struct {
	configRoot func(create bool) (*os.Root, bool, error)
	userHome   func() (string, error)
}

func NewQuickSelectService() *QuickSelectService {
	keys := NewKeybindingService()
	return &QuickSelectService{configRoot: keys.openConfigRoot, userHome: keys.homeDirectory}
}

func (s *QuickSelectService) QuickSelectPath() (string, error) {
	home, err := s.userHome()
	if err != nil {
		return "", fmt.Errorf("find user home directory: %w", err)
	}
	return filepath.Join(home, "DiggerTerm", quickSelectFileName), nil
}

func defaultQuickSelectConfig() QuickSelectConfig {
	return QuickSelectConfig{Version: 1, EnabledTypes: []string{"url", "ip", "path", "gitHash"}, Rules: []QuickSelectRule{}}
}

func (s *QuickSelectService) QuickSelectConfig() (QuickSelectConfig, error) {
	root, missing, err := s.configRoot(false)
	if err != nil {
		return QuickSelectConfig{}, err
	}
	if missing {
		return defaultQuickSelectConfig(), nil
	}
	defer root.Close()
	info, err := root.Lstat(quickSelectFileName)
	if errors.Is(err, os.ErrNotExist) {
		return defaultQuickSelectConfig(), nil
	}
	if err != nil {
		return QuickSelectConfig{}, fmt.Errorf("inspect quick-select.json: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return QuickSelectConfig{}, errors.New("quick-select.json must be a regular file; symlinks are not allowed")
	}
	if info.Size() > maxQuickSelectBytes {
		return QuickSelectConfig{}, fmt.Errorf("quick-select.json exceeds %d KiB", maxQuickSelectBytes/1024)
	}
	file, err := root.Open(quickSelectFileName)
	if err != nil {
		return QuickSelectConfig{}, fmt.Errorf("open quick-select.json: %w", err)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxQuickSelectBytes+1))
	if err != nil {
		return QuickSelectConfig{}, fmt.Errorf("read quick-select.json: %w", err)
	}
	if len(data) > maxQuickSelectBytes {
		return QuickSelectConfig{}, fmt.Errorf("quick-select.json exceeds %d KiB", maxQuickSelectBytes/1024)
	}
	var config QuickSelectConfig
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return QuickSelectConfig{}, fmt.Errorf("decode quick-select.json: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return QuickSelectConfig{}, errors.New("quick-select.json contains trailing data")
	}
	if err := validateQuickSelectConfig(config); err != nil {
		return QuickSelectConfig{}, err
	}
	return config, nil
}

func (s *QuickSelectService) SaveQuickSelectConfig(config QuickSelectConfig) error {
	if err := validateQuickSelectConfig(config); err != nil {
		return err
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("encode quick-select.json: %w", err)
	}
	if len(data) > maxQuickSelectBytes {
		return fmt.Errorf("quick-select.json exceeds %d KiB", maxQuickSelectBytes/1024)
	}
	root, missing, err := s.configRoot(true)
	if err != nil {
		return err
	}
	if missing || root == nil {
		return errors.New("create DiggerTerm configuration directory")
	}
	defer root.Close()
	var temporary string
	var file *os.File
	for range 5 {
		temporary = ".quick-select-" + rand.Text() + ".tmp"
		file, err = root.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil || !errors.Is(err, os.ErrExist) {
			break
		}
	}
	if err != nil {
		return fmt.Errorf("create temporary quick-select config: %w", err)
	}
	defer root.Remove(temporary)
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return fmt.Errorf("write quick-select.json: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync quick-select.json: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close quick-select.json: %w", err)
	}
	if err := root.Rename(temporary, quickSelectFileName); err != nil {
		return fmt.Errorf("replace quick-select.json: %w", err)
	}
	return nil
}

func validateQuickSelectConfig(config QuickSelectConfig) error {
	if config.Version != 1 {
		return errors.New("quick-select version must be 1")
	}
	if config.EnabledTypes == nil || config.Rules == nil {
		return errors.New("quick-select enabledTypes and rules must be arrays")
	}
	seenTypes := make(map[string]bool)
	for _, kind := range config.EnabledTypes {
		if !quickSelectTypes[kind] || seenTypes[kind] {
			return fmt.Errorf("unknown or duplicate Quick Select type %q", kind)
		}
		seenTypes[kind] = true
	}
	if len(config.Rules) > maxQuickSelectRules {
		return fmt.Errorf("quick-select.json can contain at most %d custom rules", maxQuickSelectRules)
	}
	seenIDs := make(map[string]bool)
	for _, rule := range config.Rules {
		if rule.ID == "" || len(rule.ID) > 64 || strings.ContainsAny(rule.ID, "\r\n\x00") || seenIDs[rule.ID] {
			return errors.New("Quick Select rule IDs must be unique and 1-64 safe characters")
		}
		seenIDs[rule.ID] = true
		if name := strings.TrimSpace(rule.Name); name == "" || !utf8.ValidString(name) || utf8.RuneCountInString(name) > maxQuickSelectNameRunes {
			return fmt.Errorf("Quick Select rule names must be 1-%d characters", maxQuickSelectNameRunes)
		}
		if !utf8.ValidString(rule.Pattern) || strings.TrimSpace(rule.Pattern) == "" || utf8.RuneCountInString(rule.Pattern) > maxQuickSelectPatternRunes {
			return fmt.Errorf("Quick Select rule patterns must be 1-%d characters", maxQuickSelectPatternRunes)
		}
	}
	return nil
}
