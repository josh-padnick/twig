// Package codex resolves a local Codex session to the directory it ran in.
//
// A local Codex session (Codex Desktop / CLI) is a thread, addressable as
// codex://threads/<id>. Unlike Claude Code and Conductor, local Codex has no
// worktree-directory layout to scan: a session just runs in some existing
// repo or worktree — a main checkout, a Conductor workspace, a .claude
// worktree — and records that cwd in its rollout log under
// ~/.codex/sessions/<yyyy>/<mm>/<dd>/rollout-<ts>-<id>.jsonl. twig reads that
// log's session_meta line to jump straight to where the session was working,
// the same way it turns a PR URL into a branch.
package codex

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// threadURIRe matches codex://threads/<id> (and the singular codex://thread/),
// ignoring any trailing path or query so a copied link still parses.
var threadURIRe = regexp.MustCompile(`^codex://threads?/([0-9a-fA-F-]+)`)

// uuidRe matches a bare session id (a UUID, any version).
var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// ParseThreadURI extracts the thread id from a codex://threads/<id> URI. The
// URI form is explicit, so callers resolve it straight away.
func ParseThreadURI(s string) (id string, ok bool) {
	m := threadURIRe.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return "", false
	}
	return m[1], true
}

// ThreadID returns s when it is a bare session id (UUID). It's intentionally
// strict so this only ever shadows a real worktree name as a last-resort
// fallback, never something a user would plausibly type as a fragment.
func ThreadID(s string) (id string, ok bool) {
	s = strings.TrimSpace(s)
	if uuidRe.MatchString(s) {
		return s, true
	}
	return "", false
}

// NotFoundError reports that no local session log matched the id.
type NotFoundError struct{ ID string }

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("no local Codex session %s found under ~/.codex/sessions", e.ID)
}

// Session is the slice of a rollout log twig cares about.
type Session struct {
	ID  string
	Cwd string // the directory the session ran in
}

// Find locates the session with id under sessionsDir (normally
// ~/.codex/sessions) and returns its recorded cwd.
func Find(sessionsDir, id string) (Session, error) {
	path, err := findRollout(sessionsDir, id)
	if err != nil {
		return Session{}, err
	}
	return readSessionMeta(path)
}

// findRollout walks sessionsDir for the rollout log whose filename carries the
// id. Filenames embed the id (rollout-<ts>-<id>.jsonl), so matching never
// reads file contents until the one log is found.
func findRollout(sessionsDir, id string) (string, error) {
	var found string
	_ = filepath.WalkDir(sessionsDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries rather than abort the search
		}
		if d.IsDir() {
			return nil
		}
		if name := d.Name(); strings.Contains(name, id) && strings.HasSuffix(name, ".jsonl") {
			found = p
			return fs.SkipAll
		}
		return nil
	})
	if found == "" {
		return "", &NotFoundError{ID: id}
	}
	return found, nil
}

// readSessionMeta returns the cwd recorded in the log's session_meta entry.
// That entry is the first line and can be large (it embeds the agent's base
// instructions), so the scanner runs with a generous buffer.
func readSessionMeta(path string) (Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return Session{}, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		var line struct {
			Type    string `json:"type"`
			Payload struct {
				ID  string `json:"id"`
				Cwd string `json:"cwd"`
			} `json:"payload"`
		}
		if json.Unmarshal(sc.Bytes(), &line) != nil {
			continue
		}
		if line.Type == "session_meta" && line.Payload.Cwd != "" {
			return Session{ID: line.Payload.ID, Cwd: line.Payload.Cwd}, nil
		}
	}
	if err := sc.Err(); err != nil {
		return Session{}, err
	}
	return Session{}, fmt.Errorf("Codex session log %s records no working directory", filepath.Base(path))
}
