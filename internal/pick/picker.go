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

// NoTTYError reports an ambiguous choice in a non-interactive context,
// carrying rendered lines so the error message can enumerate the options.
type NoTTYError struct {
	Lines []string
}

func (e *NoTTYError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d matches and no terminal is available to pick one:", len(e.Lines))
	for _, line := range e.Lines {
		fmt.Fprintf(&b, "\n  %s", line)
	}
	return b.String()
}

// OneOf returns the only item, or runs the fuzzy picker over several.
func OneOf[T any](items []T, display func(T) string) (T, error) {
	var zero T
	switch len(items) {
	case 0:
		return zero, errors.New("no candidates to pick from")
	case 1:
		return items[0], nil
	}
	if !hasTTY() {
		lines := make([]string, len(items))
		for i, it := range items {
			lines[i] = display(it)
		}
		return zero, &NoTTYError{Lines: lines}
	}
	idx, err := fuzzyfinder.Find(items, func(i int) string { return display(items[i]) })
	if err != nil {
		if errors.Is(err, fuzzyfinder.ErrAbort) {
			return zero, ErrCancelled
		}
		return zero, fmt.Errorf("picker: %w", err)
	}
	return items[idx], nil
}

// One picks among worktree candidates.
func One(cands []resolve.Candidate) (resolve.Candidate, error) {
	return OneOf(cands, Display)
}

// ManyOf runs a multi-select fuzzy picker (TAB toggles, Enter confirms;
// with no toggles the highlighted item is the selection). The header
// renders inside the picker UI — the picker takes over the screen, so any
// instructions printed beforehand would be invisible while it runs.
// preselected, when non-nil, marks items as selected up front.
func ManyOf[T any](items []T, display func(T) string, header string, preselected func(T) bool) ([]T, error) {
	if len(items) == 0 {
		return nil, nil
	}
	if !hasTTY() {
		lines := make([]string, len(items))
		for i, it := range items {
			lines[i] = display(it)
		}
		return nil, &NoTTYError{Lines: lines}
	}
	opts := []fuzzyfinder.Option{fuzzyfinder.WithHeader(header)}
	if preselected != nil {
		opts = append(opts, fuzzyfinder.WithPreselected(func(i int) bool { return preselected(items[i]) }))
	}
	idxs, err := fuzzyfinder.FindMulti(items, func(i int) string { return display(items[i]) }, opts...)
	if err != nil {
		if errors.Is(err, fuzzyfinder.ErrAbort) {
			return nil, ErrCancelled
		}
		return nil, fmt.Errorf("picker: %w", err)
	}
	out := make([]T, 0, len(idxs))
	for _, i := range idxs {
		out = append(out, items[i])
	}
	return out, nil
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
