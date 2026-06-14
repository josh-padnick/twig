// Package opener turns "enter a worktree" into actions on the user's tools:
// terminals, editors, browsers. Openers are named in config (or a trusted
// repo's twig.toml) and run as a set. This package is the contribution
// point for new tools — a built-in opener is one file plus a case in
// FromConfig; anything launchable from a shell already works today via the
// generic "command" kind.
package opener

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/josh-padnick/twig/internal/config"
)

// TargetMode selects where a terminal opener puts the session.
type TargetMode int

const (
	ModeWindow     TargetMode = iota // open a new window (default)
	ModeCurrentTab                   // enter in the currently focused tab (-t)
)

// Target is what an opener acts on.
type Target struct {
	Dir      string // worktree directory (absolute)
	EnterCmd string // twig-enter command to run in the shell, "" for none
	Mode     TargetMode
	// Fold holds other openers' launch commands to run after the cd and
	// before the entry command. Folded launches are best-effort so they cannot
	// prevent setup or --run from executing in the terminal.
	Fold []string
}

// Opener is one named way of opening a worktree.
type Opener interface {
	Name() string
	// CanInject reports whether the opener can type a command into the
	// terminal it opens — required to host setup output and [run] scripts.
	CanInject() bool
	// CanCurrentTab reports whether the opener supports -t (entering the
	// worktree in the currently focused tab instead of a new window).
	CanCurrentTab() bool
	// EntersCurrentTab reports whether, for the requested mode, this opener
	// will actually run the entry in the current tab (vs a new window). A
	// current-tab terminal can host the other openers' launch commands.
	EntersCurrentTab(requested TargetMode) bool
	// LaunchCmd returns the shell command that performs this opener's launch
	// (e.g. `cursor '/dir'`) and whether it can be folded into a host
	// terminal's entry line. Terminals that host the entry return ok=false.
	LaunchCmd(t Target) (cmd string, ok bool)
	// Available diagnoses whether the opener can work on this machine.
	Available() error
	Open(t Target) error
}

// FromConfig resolves a named opener against the user catalog. "ghostty" is
// usable without a definition; defining it only tunes clear/delay.
func FromConfig(name string, oc config.Open) (Opener, error) {
	spec, defined := oc.Openers[name]
	if !defined {
		if name == "ghostty" {
			return newGhostty(config.Opener{}), nil
		}
		return nil, fmt.Errorf("opener %q is not defined under [open.openers] in config", name)
	}
	switch spec.Kind {
	case "ghostty":
		return newGhostty(spec), nil
	case "command":
		return newCommand(name, spec)
	default:
		return nil, fmt.Errorf("opener %q has unknown kind %q (known: ghostty, command)", name, spec.Kind)
	}
}

// EntryLine composes the shell line a terminal opener types on entry: cd into
// the worktree, optionally clear (gw parity), then any folded opener launches,
// then twig enter. The cd/clear steps gate the rest; folded launch failures do
// not stop the entry command.
func EntryLine(t Target, clear bool) string {
	parts := []string{"cd " + Quote(t.Dir)}
	if clear {
		parts = append(parts, "clear")
	}
	return joinEntryLine(parts, t)
}

func joinEntryLine(gated []string, t Target) string {
	tail := append([]string{}, t.Fold...)
	if t.EnterCmd != "" {
		tail = append(tail, t.EnterCmd)
	}
	if len(t.Fold) == 0 {
		parts := append([]string{}, gated...)
		parts = append(parts, tail...)
		return strings.Join(parts, " && ")
	}
	tailLine := strings.Join(tail, "; ")
	if len(gated) == 0 {
		return tailLine
	}
	if tailLine == "" {
		return strings.Join(gated, " && ")
	}
	return strings.Join(gated, " && ") + " && { " + tailLine + "; }"
}

// SameDir reports whether target is the directory twig is running in,
// comparing resolved paths so a symlinked worktree root still matches. Used
// to skip a redundant self-cd when reusing the current terminal.
func SameDir(target string) bool {
	cwd, err := os.Getwd()
	if err != nil {
		return false
	}
	if r, err := filepath.EvalSymlinks(cwd); err == nil {
		cwd = r
	}
	if r, err := filepath.EvalSymlinks(target); err == nil {
		target = r
	}
	return cwd == target
}
