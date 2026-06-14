package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// write creates a config file in a temp dir and returns its path.
func write(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadMissingFileGivesDefaults(t *testing.T) {
	cfg, warnings, err := LoadFrom(filepath.Join(t.TempDir(), "nope.toml"))
	if err != nil || len(warnings) != 0 {
		t.Fatalf("err=%v warnings=%v", err, warnings)
	}
	if len(cfg.Roots) != 0 {
		t.Errorf("default roots = %v, want empty", cfg.Roots)
	}
	if len(cfg.Providers) == 0 {
		t.Error("default providers should include all builtins")
	}
	if got := cfg.Open.Default; len(got) != 1 || got[0] != "ghostty" {
		t.Errorf("default openers = %v, want [ghostty]", got)
	}
	if !cfg.SetupAuto() {
		t.Error("setup.auto should default to true")
	}
	if !cfg.RemoteConfirmBeforeFetch() {
		t.Error("remote.confirm_before_fetch should default to true")
	}
}

func TestLoadPartialFileKeepsOtherDefaults(t *testing.T) {
	cfg, warnings, err := LoadFrom(write(t, `roots = ["~/Code/fabricahq"]`))
	if err != nil || len(warnings) != 0 {
		t.Fatalf("err=%v warnings=%v", err, warnings)
	}
	if len(cfg.Roots) != 1 {
		t.Errorf("roots = %v", cfg.Roots)
	}
	if len(cfg.Providers) == 0 {
		t.Error("providers default lost when only roots set")
	}
	got := cfg.ExpandedRoots("/home/u")
	if got[0] != "/home/u/Code/fabricahq" {
		t.Errorf("expanded root = %s", got[0])
	}
}

func TestLoadExplicitEmptyProvidersDisablesAll(t *testing.T) {
	cfg, _, err := LoadFrom(write(t, `providers = []`))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Providers) != 0 {
		t.Errorf("providers = %v, want empty", cfg.Providers)
	}
}

func TestLoadWarnings(t *testing.T) {
	cfg, warnings, err := LoadFrom(write(t, `
made_up_key = true
providers = ["conductor", "not-a-tool"]

[open]
default = ["ghostty", "phantom"]

[open.openers.broken]
kind = "teleport"

[open.openers.empty]
kind = "command"
`))
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(warnings, "\n")
	for _, want := range []string{"made_up_key", "not-a-tool", "phantom", "teleport", `"empty"`} {
		if !strings.Contains(joined, want) {
			t.Errorf("warnings missing %q:\n%s", want, joined)
		}
	}
	// Warnings must not block loading.
	if len(cfg.Providers) != 2 {
		t.Errorf("providers = %v", cfg.Providers)
	}
}

func TestLoadOpenersAndRemote(t *testing.T) {
	cfg, warnings, err := LoadFrom(write(t, `
[open]
default = ["ghostty", "cursor"]

[open.openers.cursor]
kind = "command"
command = "cursor {{dir}}"

[open.openers.ghostty]
kind = "ghostty"
clear = false
delay_ms = 500
reuse_window = true

[remote]
auto_include = true
confirm_before_fetch = false
dir = "wt/{{slug}}"
`))
	if err != nil || len(warnings) != 0 {
		t.Fatalf("err=%v warnings=%v", err, warnings)
	}
	cursor := cfg.Open.Openers["cursor"]
	if cursor.Kind != "command" || cursor.Command != "cursor {{dir}}" {
		t.Errorf("cursor opener = %+v", cursor)
	}
	gh := cfg.Open.Openers["ghostty"]
	if gh.Clear == nil || *gh.Clear || gh.DelayMs == nil || *gh.DelayMs != 500 || !gh.ReuseWindow {
		t.Errorf("ghostty opener = %+v", gh)
	}
	if !cfg.Remote.AutoInclude || cfg.RemoteConfirmBeforeFetch() || cfg.Remote.Dir != "wt/{{slug}}" {
		t.Errorf("remote = %+v", cfg.Remote)
	}
}

func TestLoadBadTOMLIsAnError(t *testing.T) {
	_, _, err := LoadFrom(write(t, `roots = [unclosed`))
	if err == nil {
		t.Fatal("expected parse error")
	}
}
