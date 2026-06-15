// Codex session pickup: a local Codex session is a thread, not a worktree
// layout, so it can't be scanned for. Instead twig reads the session's rollout
// log to learn the directory it ran in, then enters that directory through the
// normal open/cd flow — the same shape as remote-branch pickup. It's reached
// either by a codex://threads/<id> URI (or bare thread id), or by -s, which
// fuzzy-matches the session's human title.
package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/josh-padnick/twig/internal/codex"
	"github.com/josh-padnick/twig/internal/pick"
	"github.com/josh-padnick/twig/internal/resolve"
	"github.com/josh-padnick/twig/internal/ui"
)

// resolveEntry chooses the resolution mode for the entry commands: a Codex
// session-title search with -s, otherwise the normal fragment/remote path.
func resolveEntry(frag string, session, remoteFlag, verbose bool) (resolve.Candidate, error) {
	if session {
		return resolveCodexSession(frag)
	}
	return resolveFragmentOrRemote(frag, remoteFlag, verbose)
}

// resolveCodexSession fuzzy-matches query against local Codex session titles
// (~/.codex/session_index.jsonl) and resolves the chosen one to the directory
// it ran in. Several matches open the picker; one match enters directly.
func resolveCodexSession(query string) (resolve.Candidate, error) {
	var zero resolve.Candidate
	if strings.TrimSpace(query) == "" {
		return zero, errors.New("twig -s needs a session title to search for")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return zero, err
	}
	codexDir := filepath.Join(home, ".codex")
	entries, err := codex.ReadIndex(filepath.Join(codexDir, "session_index.jsonl"))
	if err != nil {
		return zero, err
	}
	matches := codex.SearchByTitle(entries, query)
	if len(matches) == 0 {
		return zero, fmt.Errorf("no Codex session title matches %q", query)
	}

	// Resolve each match to its cwd so the picker can show where it lands;
	// skip any whose rollout log can't be located.
	sessionsDir := filepath.Join(codexDir, "sessions")
	type hit struct{ title, cwd string }
	var hits []hit
	for _, e := range matches {
		s, ferr := codex.Find(sessionsDir, e.ID)
		if ferr != nil {
			continue
		}
		hits = append(hits, hit{e.Title, s.Cwd})
	}
	if len(hits) == 0 {
		return zero, fmt.Errorf("matched %d Codex session(s) for %q, but couldn't locate their logs", len(matches), query)
	}

	chosen := hits[0]
	if len(hits) > 1 {
		chosen, err = pick.OneOf(hits, func(h hit) string { return h.title + "  [" + ui.Tilde(h.cwd) + "]" },
			fmt.Sprintf("%d Codex sessions match %q — select one:", len(hits), query))
		if err != nil {
			return zero, err
		}
	}
	if fi, statErr := os.Stat(chosen.cwd); statErr != nil || !fi.IsDir() {
		return zero, fmt.Errorf("Codex session %q ran in %s, but that directory no longer exists", chosen.title, chosen.cwd)
	}
	ui.Stepf("Codex session %q ran in %s", chosen.title, ui.Tilde(chosen.cwd))
	return resolve.Candidate{Path: chosen.cwd, Source: resolve.SourceCodex}, nil
}

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
