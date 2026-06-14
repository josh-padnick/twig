// Package config loads twig's user configuration from
// $XDG_CONFIG_HOME/twig/config.toml (~/.config/twig/config.toml). A missing
// file is fully supported — every setting has a default chosen so that the
// zero-config experience works on a machine with Conductor or Claude Code
// installed. Unknown keys and dangling references produce warnings, never
// hard failures, so a typo can't lock the user out of their own tool.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/josh-padnick/twig/internal/resolve"
	"github.com/josh-padnick/twig/internal/xdg"
)

// Config mirrors config.toml. Pointer fields distinguish "absent" from
// "explicitly set to the zero value" where the default isn't the zero value.
type Config struct {
	Roots     []string `toml:"roots"`
	Providers []string `toml:"providers"`
	Open      Open     `toml:"open"`
	Remote    Remote   `toml:"remote"`
	Setup     Setup    `toml:"setup"`
}

// Open configures what happens on enter: which openers run by default and
// the user-defined opener catalog.
type Open struct {
	Default []string          `toml:"default"`
	Openers map[string]Opener `toml:"openers"`
}

// Opener is one named way of opening a worktree. Kind "ghostty" is the
// built-in AppleScript launcher; kind "command" runs a template with
// {{dir}} (and optionally {{cmd}} for injection-capable terminals).
type Opener struct {
	Kind        string `toml:"kind"`
	Command     string `toml:"command"`
	Clear       *bool  `toml:"clear"`        // ghostty: inject `clear` after cd (default true)
	DelayMs     *int   `toml:"delay_ms"`     // ghostty: delay before input text (default 300)
	ReuseWindow bool   `toml:"reuse_window"` // ghostty: when already inside Ghostty, enter the current window instead of opening a new one
}

// Remote configures remote-branch pickup for cloud sessions.
type Remote struct {
	Auto               bool   `toml:"auto"`                 // search remotes on every local miss (no -r needed)
	ConfirmBeforeFetch *bool  `toml:"confirm_before_fetch"` // prompt before fetching a matched branch (default true); false = fetch without asking
	Dir                string `toml:"dir"`                  // worktree location template relative to the main repo root
}

// Setup configures the on-entry setup behavior.
type Setup struct {
	Auto *bool `toml:"auto"` // run setup on entry (default true)
}

// Default returns the zero-config configuration.
func Default() Config {
	return Config{
		Providers: resolve.BuiltinNames(),
		Open:      Open{Default: []string{"ghostty"}},
		Remote:    Remote{Dir: filepath.Join(".claude", "worktrees", "{{slug}}")},
	}
}

// Path returns the config file location (which may not exist).
func Path() (string, error) {
	dir, err := xdg.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

// Load reads the user config, returning defaults when no file exists.
// Warnings cover unknown keys and dangling references; only unparseable
// TOML is a hard error.
func Load() (Config, []string, error) {
	path, err := Path()
	if err != nil {
		return Default(), nil, err
	}
	return LoadFrom(path)
}

// LoadFrom is Load against an explicit path, for tests and doctor.
func LoadFrom(path string) (Config, []string, error) {
	cfg := Default()
	md, err := toml.DecodeFile(path, &cfg)
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, nil, nil
	}
	if err != nil {
		return Default(), nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	var warnings []string
	for _, key := range md.Undecoded() {
		warnings = append(warnings, fmt.Sprintf("unknown config key %q in %s", key, path))
	}
	warnings = append(warnings, cfg.validate()...)
	return cfg, warnings, nil
}

// validate flags references that will silently do nothing at runtime:
// unknown provider names, opener kinds twig doesn't ship, command openers
// without a command, and default entries that name no known opener.
func (c Config) validate() []string {
	var warnings []string
	known := map[string]bool{}
	for _, name := range resolve.BuiltinNames() {
		known[name] = true
	}
	for _, p := range c.Providers {
		if !known[p] {
			warnings = append(warnings, fmt.Sprintf("unknown provider %q (known: %s)", p, strings.Join(resolve.BuiltinNames(), ", ")))
		}
	}
	for name, op := range c.Open.Openers {
		switch op.Kind {
		case "ghostty":
		case "command":
			if strings.TrimSpace(op.Command) == "" {
				warnings = append(warnings, fmt.Sprintf("opener %q has kind \"command\" but no command", name))
			}
		default:
			warnings = append(warnings, fmt.Sprintf("opener %q has unknown kind %q (known: ghostty, command)", name, op.Kind))
		}
	}
	for _, name := range c.Open.Default {
		if name == "ghostty" {
			continue // built-in, usable without definition
		}
		if _, ok := c.Open.Openers[name]; !ok {
			warnings = append(warnings, fmt.Sprintf("open.default names undefined opener %q", name))
		}
	}
	return warnings
}

// ExpandedRoots returns the configured roots with ~ expanded and paths
// cleaned, ready for the resolver.
func (c Config) ExpandedRoots(home string) []string {
	var roots []string
	for _, r := range c.Roots {
		if r == "~" || strings.HasPrefix(r, "~/") {
			r = filepath.Join(home, strings.TrimPrefix(r[1:], "/"))
		}
		roots = append(roots, filepath.Clean(r))
	}
	return roots
}

// SetupAuto reports whether setup should run on entry (default true).
func (c Config) SetupAuto() bool {
	return c.Setup.Auto == nil || *c.Setup.Auto
}

// RemoteConfirmBeforeFetch reports whether remote pickup should ask before
// fetching a matched branch (default true). Set it false to fetch and create
// the worktree without a prompt.
func (c Config) RemoteConfirmBeforeFetch() bool {
	return c.Remote.ConfirmBeforeFetch == nil || *c.Remote.ConfirmBeforeFetch
}
