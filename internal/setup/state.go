// The setup state marker: a JSON file in the worktree's real git directory
// (<main>/.git/worktrees/<name>/ for linked worktrees) recording the last
// successful setup run. Living in the git dir means it survives everything
// except worktree removal — exactly the lifetime setup state should have —
// without polluting the working tree.
package setup

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/josh-padnick/twig/internal/gitx"
)

const stateFileName = "twig-setup.json"

// State records one successful setup run.
type State struct {
	Version        int               `json:"version"`
	ManifestPath   string            `json:"manifest_path"`
	ManifestSHA256 string            `json:"manifest_sha256"`
	Watch          map[string]string `json:"watch"`
	LastSuccess    time.Time         `json:"last_success"`
	TwigVersion    string            `json:"twig_version"`
}

// readState returns the worktree's recorded state, or nil when there is
// none (never run, not a git worktree, or unreadable — all mean "run").
func readState(worktree string) *State {
	gitDir, err := gitx.GitDir(worktree)
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(gitDir, stateFileName))
	if err != nil {
		return nil
	}
	var st State
	if json.Unmarshal(data, &st) != nil {
		return nil
	}
	return &st
}

// writeState atomically records a successful run in the worktree's git dir.
func writeState(worktree string, st State) error {
	gitDir, err := gitx.GitDir(worktree)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(gitDir, ".twig-setup-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), filepath.Join(gitDir, stateFileName))
}

// hashBytes is the content hash used for both the manifest and watch files.
func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// watchHashes hashes each watched file relative to the worktree. Missing
// files hash to the sentinel "absent" so creating or deleting a watched
// file counts as a change.
func watchHashes(worktree string, watch []string) map[string]string {
	hashes := map[string]string{}
	for _, w := range watch {
		p := w
		if !filepath.IsAbs(p) {
			p = filepath.Join(worktree, w)
		}
		data, err := os.ReadFile(p)
		if errors.Is(err, fs.ErrNotExist) || err != nil {
			hashes[w] = "absent"
			continue
		}
		hashes[w] = hashBytes(data)
	}
	return hashes
}

// needsSetup reports whether setup must run: no recorded success, a changed
// manifest, or any changed watch file.
func needsSetup(worktree string, ld *Loaded) bool {
	st := readState(worktree)
	if st == nil {
		return true
	}
	if st.ManifestSHA256 != hashBytes(ld.Content) {
		return true
	}
	current := watchHashes(worktree, ld.Manifest.Setup.Watch)
	if len(current) != len(st.Watch) {
		return true
	}
	for file, hash := range current {
		if st.Watch[file] != hash {
			return true
		}
	}
	return false
}
