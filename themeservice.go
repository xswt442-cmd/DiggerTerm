package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const maxThemeFileSize = 128 * 1024

// ThemeService reads GUI-only theme files from the current user's home folder.
// It never reads files outside that single directory or communicates with the daemon.
type ThemeService struct {
	directory func() (string, error)
}

type ThemeCatalog struct {
	Directory string       `json:"directory"`
	Themes    []ThemeFile  `json:"themes"`
	CustomCSS string       `json:"customCSS"`
	Errors    []ThemeError `json:"errors"`
}

type ThemeFile struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	Base       string             `json:"base"`
	Colors     map[string]string  `json:"colors"`
	Terminal   map[string]string  `json:"terminal"`
	Appearance map[string]float64 `json:"appearance"`
	CSS        string             `json:"css"`
}

type ThemeError struct {
	File    string `json:"file"`
	Message string `json:"message"`
}

type themeJSON struct {
	Version     int                `json:"version"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Base        string             `json:"base"`
	Colors      map[string]string  `json:"colors"`
	Terminal    map[string]string  `json:"terminal"`
	Appearance  map[string]float64 `json:"appearance"`
}

func NewThemeService() *ThemeService {
	return &ThemeService{directory: func() (string, error) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("find user home directory: %w", err)
		}
		return themeDirectoryForHome(home)
	}}
}

func themeDirectoryForHome(home string) (string, error) {
	if !filepath.IsAbs(home) {
		return "", errors.New("user home directory must be absolute")
	}
	return filepath.Join(home, "DiggerTerm"), nil
}

func (s *ThemeService) ThemeDirectory() (string, error) {
	dir, err := s.directory()
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(dir) {
		return "", errors.New("theme directory must be absolute")
	}
	return filepath.Clean(dir), nil
}

func (s *ThemeService) OpenThemeDirectory() error {
	dir, err := s.ThemeDirectory()
	if err != nil {
		return err
	}
	if err := ensureThemeDirectory(dir); err != nil {
		return fmt.Errorf("prepare theme directory: %w", err)
	}
	return application.Get().Env.OpenFileManager(dir, false)
}

func ensureThemeDirectory(dir string) error {
	if !filepath.IsAbs(dir) {
		return errors.New("theme directory must be absolute")
	}
	parent := filepath.Dir(dir)
	if _, err := os.Lstat(parent); errors.Is(err, os.ErrNotExist) {
		if err := os.Mkdir(parent, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("create theme parent: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("inspect theme parent: %w", err)
	}
	parentInfo, err := os.Lstat(parent)
	if err != nil {
		return fmt.Errorf("inspect theme parent: %w", err)
	}
	if !parentInfo.IsDir() || parentInfo.Mode()&os.ModeSymlink != 0 {
		return errors.New("theme parent must be a real directory")
	}
	if err := os.Mkdir(dir, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return fmt.Errorf("create theme directory: %w", err)
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return fmt.Errorf("inspect theme directory: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("theme directory must be a real directory")
	}
	return nil
}

func (s *ThemeService) ListThemes() (ThemeCatalog, error) {
	dir, err := s.ThemeDirectory()
	if err != nil {
		return ThemeCatalog{}, err
	}
	catalog := ThemeCatalog{Directory: dir, Themes: []ThemeFile{}, Errors: []ThemeError{}}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return catalog, nil
	}
	if err != nil {
		return ThemeCatalog{}, fmt.Errorf("read theme directory: %w", err)
	}
	if info, statErr := os.Lstat(dir); statErr == nil && !info.IsDir() {
		return ThemeCatalog{}, fmt.Errorf("theme directory must be a real directory")
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return ThemeCatalog{}, fmt.Errorf("open theme directory: %w", err)
	}
	defer root.Close()

	jsonFiles, cssFiles := map[string]string{}, map[string]string{}
	for _, entry := range entries {
		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".json" && ext != ".css" {
			continue
		}
		if strings.EqualFold(name, "keybindings.json") {
			continue
		}
		stem := strings.TrimSuffix(name, filepath.Ext(name))
		if !validThemeStem(stem) {
			catalog.Errors = append(catalog.Errors, ThemeError{name, "invalid theme filename"})
			continue
		}
		info, statErr := os.Lstat(filepath.Join(dir, name))
		if statErr != nil {
			catalog.Errors = append(catalog.Errors, ThemeError{name, statErr.Error()})
			continue
		}
		if !info.Mode().IsRegular() {
			catalog.Errors = append(catalog.Errors, ThemeError{name, "theme entry must be a regular file; symlinks are not allowed"})
			continue
		}
		key := strings.ToLower(stem)
		bucket := jsonFiles
		if ext == ".css" {
			bucket = cssFiles
		}
		if _, exists := bucket[key]; exists {
			catalog.Errors = append(catalog.Errors, ThemeError{name, "duplicate theme filename ignoring case"})
			continue
		}
		bucket[key] = name
	}

	stems := make([]string, 0, len(jsonFiles)+len(cssFiles))
	seen := map[string]bool{}
	for stem := range jsonFiles {
		seen[stem] = true
	}
	for stem := range cssFiles {
		seen[stem] = true
	}
	for stem := range seen {
		stems = append(stems, stem)
	}
	sort.Strings(stems)
	for _, stem := range stems {
		jsonName, hasJSON := jsonFiles[stem]
		cssName, hasCSS := cssFiles[stem]
		actualStem := stem
		if hasJSON {
			actualStem = strings.TrimSuffix(jsonName, filepath.Ext(jsonName))
		} else if hasCSS {
			actualStem = strings.TrimSuffix(cssName, filepath.Ext(cssName))
		}
		if stem == "custom" {
			if hasJSON {
				catalog.Errors = append(catalog.Errors, ThemeError{jsonName, "custom is reserved for the global custom.css file"})
			}
			if hasCSS {
				data, readErr := readThemeFile(root, cssName)
				if readErr != nil {
					catalog.Errors = append(catalog.Errors, ThemeError{cssName, readErr.Error()})
				} else if validateErr := validateThemeCSS(data); validateErr != nil {
					catalog.Errors = append(catalog.Errors, ThemeError{cssName, validateErr.Error()})
				} else {
					catalog.CustomCSS = string(data)
				}
			}
			continue
		}
		item := ThemeFile{ID: "user:" + actualStem, Name: actualStem, Base: "nightfall", Colors: map[string]string{}, Terminal: map[string]string{}, Appearance: map[string]float64{}}
		if hasJSON {
			data, readErr := readThemeFile(root, jsonName)
			if readErr != nil {
				catalog.Errors = append(catalog.Errors, ThemeError{jsonName, readErr.Error()})
				continue
			}
			var value themeJSON
			decoder := json.NewDecoder(bytes.NewReader(data))
			decoder.DisallowUnknownFields()
			if decodeErr := decoder.Decode(&value); decodeErr != nil {
				catalog.Errors = append(catalog.Errors, ThemeError{jsonName, "invalid JSON: " + decodeErr.Error()})
				continue
			}
			if decodeErr := decoder.Decode(new(any)); decodeErr != io.EOF {
				catalog.Errors = append(catalog.Errors, ThemeError{jsonName, "invalid JSON: trailing data"})
				continue
			}
			if validateErr := validateThemeJSON(&value); validateErr != nil {
				catalog.Errors = append(catalog.Errors, ThemeError{jsonName, validateErr.Error()})
				continue
			}
			item.Name, item.Base = value.Name, value.Base
			item.Colors, item.Terminal, item.Appearance = value.Colors, value.Terminal, value.Appearance
			if item.Colors == nil {
				item.Colors = map[string]string{}
			}
			if item.Terminal == nil {
				item.Terminal = map[string]string{}
			}
			if item.Appearance == nil {
				item.Appearance = map[string]float64{}
			}
		}
		if hasCSS {
			data, readErr := readThemeFile(root, cssName)
			if readErr != nil {
				catalog.Errors = append(catalog.Errors, ThemeError{cssName, readErr.Error()})
			} else if validateErr := validateThemeCSS(data); validateErr != nil {
				catalog.Errors = append(catalog.Errors, ThemeError{cssName, validateErr.Error()})
			} else {
				item.CSS = string(data)
			}
		}
		if hasJSON || item.CSS != "" {
			catalog.Themes = append(catalog.Themes, item)
		}
	}
	sort.Slice(catalog.Errors, func(i, j int) bool {
		if catalog.Errors[i].File == catalog.Errors[j].File {
			return catalog.Errors[i].Message < catalog.Errors[j].Message
		}
		return catalog.Errors[i].File < catalog.Errors[j].File
	})
	return catalog, nil
}

func validThemeStem(stem string) bool {
	if stem == "" || len(stem) > 96 || stem == "." || stem == ".." {
		return false
	}
	for _, r := range stem {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

func readThemeFile(root *os.Root, name string) ([]byte, error) {
	info, err := os.Lstat(filepath.Join(root.Name(), name))
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("theme entry must be a regular file; symlinks are not allowed")
	}
	if info.Size() > maxThemeFileSize {
		return nil, fmt.Errorf("file exceeds %d KiB limit", maxThemeFileSize/1024)
	}
	f, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxThemeFileSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxThemeFileSize {
		return nil, fmt.Errorf("file exceeds %d KiB limit", maxThemeFileSize/1024)
	}
	return data, nil
}

var colorFields = setOf("sidebar", "canvas", "surface", "surfaceHover", "surfaceActive", "border", "borderFaint", "text", "muted", "faint", "accent", "accentBg", "accentText", "positive", "negative")
var terminalFields = setOf("background", "foreground", "cursor", "selectionBackground", "selectionForeground", "selectionInactiveBackground", "cursorAccent", "black", "red", "green", "yellow", "blue", "magenta", "cyan", "white", "brightBlack", "brightRed", "brightGreen", "brightYellow", "brightBlue", "brightMagenta", "brightCyan", "brightWhite")

func setOf(values ...string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range values {
		result[value] = true
	}
	return result
}

func validateThemeJSON(value *themeJSON) error {
	if value.Version != 1 {
		return errors.New("version must be 1")
	}
	if !utf8.ValidString(value.Name) || strings.TrimSpace(value.Name) == "" || len([]rune(value.Name)) > 64 {
		return errors.New("name must contain 1–64 non-space characters")
	}
	if value.Base == "" {
		value.Base = "nightfall"
	}
	if value.Base != "nightfall" && value.Base != "graphite" && value.Base != "pine" && value.Base != "bluehour" {
		return errors.New("base must be nightfall, graphite, pine, or bluehour")
	}
	for key, color := range value.Colors {
		if !colorFields[key] {
			return fmt.Errorf("unknown colors field %q", key)
		}
		if !validHexColor(color) {
			return fmt.Errorf("colors.%s must be #RGB, #RRGGBB, or #RRGGBBAA", key)
		}
	}
	for key, color := range value.Terminal {
		if !terminalFields[key] {
			return fmt.Errorf("unknown terminal field %q", key)
		}
		if !validHexColor(color) {
			return fmt.Errorf("terminal.%s must be #RGB, #RRGGBB, or #RRGGBBAA", key)
		}
	}
	for key, number := range value.Appearance {
		if key == "radius" && (number < 0 || number > 24) {
			return errors.New("appearance.radius must be between 0 and 24")
		}
		if key == "paneGap" && (number < 0 || number > 32) {
			return errors.New("appearance.paneGap must be between 0 and 32")
		}
		if key != "radius" && key != "paneGap" {
			return fmt.Errorf("unknown appearance field %q", key)
		}
	}
	return nil
}

func validHexColor(value string) bool {
	// CSS colors use only #RGB, #RRGGBB, or #RRGGBBAA to avoid CSS value injection.
	if len(value) != 4 && len(value) != 7 && len(value) != 9 || value[0] != '#' {
		return false
	}
	for _, c := range value[1:] {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}
	return true
}

func validateThemeCSS(data []byte) error {
	if !utf8.Valid(data) {
		return errors.New("CSS must be valid UTF-8")
	}
	if bytes.ContainsRune(data, '\\') {
		return errors.New("CSS escapes are not allowed")
	}
	lexer := css.NewLexer(parse.NewInput(bytes.NewReader(data)))
	var delimiters []css.TokenType
	for {
		token, value := lexer.Next()
		if token == css.ErrorToken {
			break
		}
		if token == css.AtKeywordToken {
			return fmt.Errorf("CSS at-rules are not allowed (%s)", strings.ToLower(string(value)))
		}
		if token == css.URLToken || token == css.BadURLToken {
			return errors.New("CSS url() values are not allowed")
		}
		if token == css.FunctionToken {
			name := strings.ToLower(strings.TrimSuffix(string(value), "("))
			if name != "rgb" && name != "rgba" && name != "hsl" && name != "hsla" && name != "var" && name != "calc" {
				return fmt.Errorf("CSS function %q is not allowed", name)
			}
		}
		switch token {
		case css.LeftBraceToken, css.LeftBracketToken, css.LeftParenthesisToken:
			delimiters = append(delimiters, token)
		case css.RightBraceToken, css.RightBracketToken, css.RightParenthesisToken:
			if len(delimiters) == 0 || !matchingDelimiter(delimiters[len(delimiters)-1], token) {
				return errors.New("invalid CSS delimiter nesting")
			}
			delimiters = delimiters[:len(delimiters)-1]
		}
	}
	if len(delimiters) != 0 {
		return errors.New("invalid CSS: unclosed delimiter")
	}
	if err := lexer.Err(); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("invalid CSS: %w", err)
	}
	parser := css.NewParser(parse.NewInput(bytes.NewReader(data)), false)
	for {
		grammar, _, _ := parser.Next()
		if grammar == css.ErrorGrammar {
			break
		}
	}
	if parser.HasParseError() {
		return fmt.Errorf("invalid CSS: %w", parser.Err())
	}
	return nil
}

func matchingDelimiter(open, close css.TokenType) bool {
	return open == css.LeftBraceToken && close == css.RightBraceToken ||
		open == css.LeftBracketToken && close == css.RightBracketToken ||
		open == css.LeftParenthesisToken && close == css.RightParenthesisToken
}
