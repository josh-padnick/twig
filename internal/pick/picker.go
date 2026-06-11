// Package pick is the interactive chooser for ambiguous resolutions. It
// always uses the built-in fuzzy finder (no fzf dependency); when no
// controlling terminal exists it fails with the candidate list on stderr so
// scripted callers get a useful error instead of a hung TUI.
package pick

import (
	"errors"
	"fmt"
	"os"
	"strings"

	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"

	"github.com/josh-padnick/twig/internal/resolve"
)

// ErrCancelled is returned when the user aborts the picker (Esc/Ctrl-C).
var ErrCancelled = errors.New("cancelled")

// NoTTYError reports an ambiguous resolution in a non-interactive context,
// carrying the candidates so the error message can enumerate them.
type NoTTYError struct {
	Candidates []resolve.Candidate
}

func (e *NoTTYError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d worktrees match and no terminal is available to pick one:", len(e.Candidates))
	for _, c := range e.Candidates {
		fmt.Fprintf(&b, "\n  %s", Display(c))
	}
	return b.String()
}

// One returns the only candidate, or runs the fuzzy picker over several.
func One(cands []resolve.Candidate) (resolve.Candidate, error) {
	switch len(cands) {
	case 0:
		return resolve.Candidate{}, errors.New("no candidates to pick from")
	case 1:
		return cands[0], nil
	}
	if !hasTTY() {
		return resolve.Candidate{}, &NoTTYError{Candidates: cands}
	}
	idx, err := fuzzyfinder.Find(cands, func(i int) string { return Display(cands[i]) })
	if err != nil {
		if errors.Is(err, fuzzyfinder.ErrAbort) {
			return resolve.Candidate{}, ErrCancelled
		}
		return resolve.Candidate{}, fmt.Errorf("picker: %w", err)
	}
	return cands[idx], nil
}

// Display renders a candidate as "path  [branch]", shortening the home
// prefix to ~ for readability in the picker and error lists.
func Display(c resolve.Candidate) string {
	p := c.Path
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if rel, ok := strings.CutPrefix(p, home+string(os.PathSeparator)); ok {
			p = "~" + string(os.PathSeparator) + rel
		}
	}
	if c.Branch != "" {
		return p + "  [" + c.Branch + "]"
	}
	return p
}

// hasTTY checks for a controlling terminal rather than stdout, because the
// picker must keep working inside command substitution like
// `cd "$(twig cd frag)"` where stdout is captured.
func hasTTY() bool {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return false
	}
	tty.Close()
	return true
}
