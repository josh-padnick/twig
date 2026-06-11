// Package xdg resolves the user directories twig stores state in, honoring
// XDG_CONFIG_HOME / XDG_DATA_HOME with the conventional ~/.config and
// ~/.local/share fallbacks on every OS — including macOS, matching direnv,
// so config lives at ~/.config/twig rather than ~/Library.
package xdg

import (
	"os"
	"path/filepath"
)

// ConfigDir returns the directory for twig's user configuration
// (config.toml). The directory is not created.
func ConfigDir() (string, error) {
	return userDir("XDG_CONFIG_HOME", ".config")
}

// DataDir returns the directory for twig's persistent data (trust store).
// The directory is not created.
func DataDir() (string, error) {
	return userDir("XDG_DATA_HOME", filepath.Join(".local", "share"))
}

// userDir resolves $envVar/twig when the variable is set to an absolute
// path, otherwise ~/<fallback>/twig.
func userDir(envVar, fallback string) (string, error) {
	if dir := os.Getenv(envVar); dir != "" && filepath.IsAbs(dir) {
		return filepath.Join(dir, "twig"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, fallback, "twig"), nil
}
