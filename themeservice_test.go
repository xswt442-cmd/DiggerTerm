package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testThemeService(dir string) *ThemeService {
	return &ThemeService{directory: func() (string, error) { return dir, nil }}
}

func writeFixture(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestListThemesLoadsPairsAndCSSOnlyInStableOrder(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "aurora.json", `{"version":1,"name":"暮光森林","base":"pine","colors":{"accent":"#abc"},"terminal":{"brightBlue":"#12345678"},"appearance":{"radius":8,"paneGap":6}}`)
	writeFixture(t, dir, "aurora.css", `.layout[data-dt-theme] { --dt-accent: #abcdef; }`)
	writeFixture(t, dir, "minimal.css", `.layout[data-dt-theme] { --dt-terminal-bg: #101318; }`)
	writeFixture(t, dir, "custom.css", `.layout[data-dt-theme] { --dt-border: #333; }`)
	writeFixture(t, dir, "keybindings.json", `{"version":1,"bindings":[]}`)
	if err := os.Mkdir(filepath.Join(dir, "nested"), 0o700); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, filepath.Join(dir, "nested"), "ignored.json", `not scanned`)

	catalog, err := testThemeService(dir).ListThemes()
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Themes) != 2 || catalog.Themes[0].ID != "user:aurora" || catalog.Themes[1].ID != "user:minimal" {
		t.Fatalf("unexpected stable catalog: %#v", catalog.Themes)
	}
	aurora := catalog.Themes[0]
	if aurora.Name != "暮光森林" || aurora.Base != "pine" || aurora.Colors["accent"] != "#abc" || aurora.Terminal["brightBlue"] != "#12345678" || aurora.Appearance["paneGap"] != 6 || !strings.Contains(aurora.CSS, "--dt-accent") {
		t.Fatalf("paired theme fields missing: %#v", aurora)
	}
	if catalog.Themes[1].Name != "minimal" || catalog.Themes[1].Base != "nightfall" || catalog.Themes[1].CSS == "" {
		t.Fatalf("CSS-only fallback incorrect: %#v", catalog.Themes[1])
	}
	if !strings.Contains(catalog.CustomCSS, "--dt-border") || len(catalog.Errors) != 0 {
		t.Fatalf("global CSS or errors incorrect: %#v", catalog)
	}
}

func TestListThemesReportsBadFilesAndContinues(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "good.json", `{"version":1,"name":"Good"}`)
	writeFixture(t, dir, "bad.json", `{"version":1,"name":"Bad","colors":{"accent":"red"}}`)
	writeFixture(t, dir, "unsafe.css", `.layout[data-dt-theme]{background:url(https://example.com/x)}`)
	writeFixture(t, dir, "escaped.css", `.layout[data-dt-theme]{background:u\\72l(https://example.com/x)}`)
	writeFixture(t, dir, "broken.css", `.layout[data-dt-theme] { --dt-text: #fff `)
	writeFixture(t, dir, "oversize.css", strings.Repeat("x", maxThemeFileSize+1))
	symlinkCreated := os.Symlink(filepath.Join(dir, "good.json"), filepath.Join(dir, "linked.json")) == nil

	catalog, err := testThemeService(dir).ListThemes()
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Themes) != 1 || catalog.Themes[0].ID != "user:good" {
		t.Fatalf("valid theme did not survive: %#v", catalog.Themes)
	}
	got := map[string]string{}
	for _, item := range catalog.Errors {
		got[item.File] = item.Message
	}
	for _, name := range []string{"bad.json", "unsafe.css", "escaped.css", "broken.css", "oversize.css"} {
		if got[name] == "" {
			t.Errorf("expected error for %s; errors=%#v", name, catalog.Errors)
		}
	}
	if symlinkCreated && got["linked.json"] == "" {
		t.Errorf("expected symlink rejection; errors=%#v", catalog.Errors)
	}
	if !strings.Contains(got["unsafe.css"], "url()") || !strings.Contains(got["escaped.css"], "escapes") {
		t.Fatalf("CSS network syntax accepted: %#v", got)
	}
}

func TestListThemesRejectsCaseInsensitiveDuplicates(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "shade.css", `.layout[data-dt-theme]{--dt-text:#fff}`)
	if _, err := os.Stat(filepath.Join(dir, "SHADE.css")); err == nil {
		t.Skip("filesystem is case-insensitive; cannot create case-distinct filenames")
	}
	writeFixture(t, dir, "SHADE.css", `.layout[data-dt-theme]{--dt-text:#000}`)
	catalog, err := testThemeService(dir).ListThemes()
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Themes) != 1 || len(catalog.Errors) != 1 || !strings.Contains(catalog.Errors[0].Message, "duplicate") {
		t.Fatalf("duplicate handling incorrect: %#v", catalog)
	}
}

func TestThemeDirectoryAndMissingDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "absent")
	svc := testThemeService(dir)
	resolved, err := svc.ThemeDirectory()
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(resolved) {
		t.Fatalf("directory is not absolute: %q", resolved)
	}
	catalog, err := svc.ListThemes()
	if err != nil {
		t.Fatal(err)
	}
	if catalog.Directory != resolved || len(catalog.Themes) != 0 || len(catalog.Errors) != 0 {
		t.Fatalf("missing directory should yield empty catalog: %#v", catalog)
	}
}

func TestCSSParserRejectsAtRulesAndAcceptsThemeVariables(t *testing.T) {
	if err := validateThemeCSS([]byte(`.layout[data-dt-theme]{--dt-terminal-cursor:#fff}`)); err != nil {
		t.Fatal(err)
	}
	for _, cssText := range []string{`@import "x.css";`, `@\69mport "x.css";`, `.layout{background:u\72l(https://x)}`, `.layout{background:url(https://x)}`} {
		if err := validateThemeCSS([]byte(cssText)); err == nil {
			t.Errorf("unsafe CSS accepted: %s", cssText)
		}
	}
}

func TestThemeSizeErrorIncludesLimit(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "large.css", strings.Repeat("a", maxThemeFileSize+1))
	catalog, err := testThemeService(dir).ListThemes()
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Errors) != 1 || !strings.Contains(catalog.Errors[0].Message, fmt.Sprint(maxThemeFileSize/1024)) {
		t.Fatalf("size limit missing: %#v", catalog.Errors)
	}
}

func TestThemeRefreshReflectsNamesAndDeletedFiles(t *testing.T) {
	dir := t.TempDir()
	service := testThemeService(dir)
	writeFixture(t, dir, "forest.json", `{"version":1,"name":"松林工作台","base":"pine"}`)
	writeFixture(t, dir, "ocean.json", `{"version":1,"name":"松林工作台","base":"bluehour"}`)
	first, err := service.ListThemes()
	if err != nil || len(first.Themes) != 2 || first.Themes[0].Name != first.Themes[1].Name || first.Themes[0].ID == first.Themes[1].ID {
		t.Fatalf("same-name themes must retain distinct IDs: %+v, %v", first, err)
	}
	writeFixture(t, dir, "forest.json", `{"version":1,"name":"新的显示名称"}`)
	if err := os.Remove(filepath.Join(dir, "ocean.json")); err != nil {
		t.Fatal(err)
	}
	second, err := service.ListThemes()
	if err != nil || len(second.Themes) != 1 || second.Themes[0].ID != "user:forest" || second.Themes[0].Name != "新的显示名称" {
		t.Fatalf("refresh returned stale theme metadata: %+v, %v", second, err)
	}
}
