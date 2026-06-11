// Package setup owns twig.toml: locating it, parsing it, and running its
// trust-gated setup/run scripts with hash-based skip logic. This file is the
// locator: a worktree's own twig.toml wins, falling back to the main repo
// root's so a manifest can be committed once and govern every worktree.
package setup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/josh-padnick/twig/internal/gitx"
)

// ErrNoManifest reports that no twig.toml governs a directory. Callers that
// treat "no manifest" as a no-op (entering a repo that doesn't use twig
// setup) match with errors.Is.
var ErrNoManifest = errors.New("no twig.toml found")

// FindManifest locates the twig.toml governing dir: <worktree root>/twig.toml
// first, then <main repo root>/twig.toml. Outside a repo only dir itself is
// checked, so `twig trust .` still works in odd setups.
func FindManifest(dir string) (string, error) {
	if !gitx.InRepo(dir) {
		if p := filepath.Join(dir, "twig.toml"); fileExists(p) {
			return p, nil
		}
		return "", fmt.Errorf("%w in %s", ErrNoManifest, dir)
	}
	root, err := gitx.RepoRoot(dir)
	if err != nil {
		return "", err
	}
	if p := filepath.Join(root, "twig.toml"); fileExists(p) {
		return p, nil
	}
	if wts, err := gitx.Worktrees(dir); err == nil && len(wts) > 0 {
		// The first porcelain entry is the main worktree.
		if p := filepath.Join(wts[0].Path, "twig.toml"); p != filepath.Join(root, "twig.toml") && fileExists(p) {
			return p, nil
		}
	}
	return "", fmt.Errorf("%w for worktree %s", ErrNoManifest, root)
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.Mode().IsRegular()
}
