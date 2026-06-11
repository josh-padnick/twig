// twig.toml parsing: the [setup] and [run] script sections plus the
// optional per-repo [open] override (consumed by the opener layer, and only
// when the manifest is trusted).
package setup

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"

	"github.com/josh-padnick/twig/internal/config"
)

// Manifest mirrors twig.toml.
type Manifest struct {
	Setup ScriptSection `toml:"setup"`
	Run   ScriptSection `toml:"run"`
	Open  config.Open   `toml:"open"`
}

// ScriptSection holds one script. Watch is only meaningful under [setup]:
// setup re-runs when any watched file's hash changes.
type ScriptSection struct {
	Run   string   `toml:"run"`
	Watch []string `toml:"watch"`
}

// Loaded couples a parsed manifest with the exact bytes it came from — the
// trust store is keyed on those bytes, never on the parsed form.
type Loaded struct {
	Path     string
	Content  []byte
	Manifest Manifest
	Warnings []string
}

// Load locates and parses the manifest governing dir. Returns ErrNoManifest
// (wrapped) when none exists.
func Load(dir string) (*Loaded, error) {
	path, err := FindManifest(dir)
	if err != nil {
		return nil, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	md, err := toml.Decode(string(content), &m)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	var warnings []string
	for _, key := range md.Undecoded() {
		warnings = append(warnings, fmt.Sprintf("unknown key %q in %s", key, path))
	}
	return &Loaded{Path: path, Content: content, Manifest: m, Warnings: warnings}, nil
}
