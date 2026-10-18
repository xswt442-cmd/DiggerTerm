package desktop

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func testQuickSelectService(home string) *QuickSelectService {
	keys := &KeybindingService{homeDirectory: func() (string, error) { return home, nil }}
	return &QuickSelectService{configRoot: keys.openConfigRoot, userHome: keys.homeDirectory}
}

func TestQuickSelectConfigDefaultsAndAtomicRoundTrip(t *testing.T) {
	home := t.TempDir()
	service := testQuickSelectService(home)
	config, err := service.QuickSelectConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.Version != 1 || len(config.EnabledTypes) != 4 || len(config.Rules) != 0 {
		t.Fatalf("unexpected defaults: %#v", config)
	}

	config.EnabledTypes = []string{"url", "gitHash"}
	config.Rules = []QuickSelectRule{{ID: "rule-one", Name: "Issue ID", Pattern: `#[A-Z]+-[0-9]+`, Enabled: true}}
	if err := service.SaveQuickSelectConfig(config); err != nil {
		t.Fatal(err)
	}
	path, err := service.QuickSelectPath()
	if err != nil || path != filepath.Join(home, "DiggerTerm", quickSelectFileName) {
		t.Fatalf("QuickSelectPath() = %q, %v", path, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("config permissions = %o, want 600", info.Mode().Perm())
	}
	loaded, err := service.QuickSelectConfig()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Version != config.Version || strings.Join(loaded.EnabledTypes, ",") != "url,gitHash" || len(loaded.Rules) != 1 || loaded.Rules[0] != config.Rules[0] {
		t.Fatalf("round trip changed config: %#v", loaded)
	}
}

func TestQuickSelectConfigRejectsMalformedAndUnsafeFiles(t *testing.T) {
	home := t.TempDir()
	service := testQuickSelectService(home)
	if err := os.Mkdir(filepath.Join(home, "DiggerTerm"), 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, "DiggerTerm", quickSelectFileName)
	if err := os.WriteFile(path, []byte(`{"version":1,"enabledTypes":["unknown"],"rules":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := service.QuickSelectConfig(); err == nil {
		t.Fatal("invalid type was accepted")
	}
	if err := os.WriteFile(path, []byte(`{"version":1,"enabledTypes":[],"rules":[],"unexpected":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := service.QuickSelectConfig(); err == nil {
		t.Fatal("unknown JSON field was accepted")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(home, "outside.json"), path); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	if _, err := service.QuickSelectConfig(); err == nil || !strings.Contains(err.Error(), "symlinks are not allowed") {
		t.Fatalf("symlink config error = %v", err)
	}
}

func TestQuickSelectConfigBoundsRulesAndRejectsDuplicates(t *testing.T) {
	config := defaultQuickSelectConfig()
	config.Rules = make([]QuickSelectRule, maxQuickSelectRules+1)
	if err := validateQuickSelectConfig(config); err == nil {
		t.Fatal("accepted more than the configured rule limit")
	}
	config.Rules = []QuickSelectRule{
		{ID: "same", Name: "First", Pattern: "one"},
		{ID: "same", Name: "Second", Pattern: "two"},
	}
	if err := validateQuickSelectConfig(config); err == nil {
		t.Fatal("accepted duplicate rule identities")
	}
}
