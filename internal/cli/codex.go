// Codex session pickup: a local Codex session is a thread, not a worktree
// layout, so it can't be scanned for. Instead twig reads the session's rollout
// log to learn the directory it ran in, then enters that directory through the
// normal open/cd flow — the same shape as remote-branch pickup, keyed on a
// codex://threads/<id> URI (or a bare thread id) rather than a fragment.
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/josh-padnick/twig/internal/codex"
	"github.com/josh-padnick/twig/internal/resolve"
	"github.com/josh-padnick/twig/internal/ui"
)

// resolveCodexThread maps a local Codex session id to the directory it ran in.
func resolveCodexThread(id string) (resolve.Candidate, error) {
	var zero resolve.Candidate
	home, err := os.UserHomeDir()
	if err != nil {
		return zero, err
	}
	sessionsDir := filepath.Join(home, ".codex", "sessions")
	s, err := codex.Find(sessionsDir, id)
	if err != nil {
		return zero, err
	}
	if fi, statErr := os.Stat(s.Cwd); statErr != nil || !fi.IsDir() {
		return zero, fmt.Errorf("Codex session %s ran in %s, but that directory no longer exists", shortID(id), s.Cwd)
	}
	ui.Stepf("Codex session %s ran in %s", shortID(id), ui.Tilde(s.Cwd))
	return resolve.Candidate{Path: s.Cwd, Source: resolve.SourceCodex}, nil
}

// shortID trims a UUID to its leading segment for readable narration, the way
// Codex itself abbreviates thread ids.
func shortID(id string) string {
	if i := len(id); i > 8 {
		return id[:8] + "…"
	}
	return id
}
