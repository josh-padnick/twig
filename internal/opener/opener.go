// Package opener turns "enter a worktree" into actions on the user's tools:
// terminals, editors, browsers. Openers are named in config (or a trusted
// repo's twig.toml) and run as a set. This package is the contribution
// point for new tools — a built-in opener is one file plus a case in
// FromConfig; anything launchable from a shell already works today via the
// generic "command" kind.
package opener

import (
	"fmt"
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
	Dir      string     // worktree directory (absolute)
	EnterCmd string     // twig-enter command to run in the shell, "" for none
	Mode     TargetMode
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

// EntryLine composes the shell line a terminal opener types on entry:
// cd into the worktree, optionally clear (gw parity), then twig enter.
func EntryLine(t Target, clear bool) string {
	parts := []string{"cd " + Quote(t.Dir)}
	if clear {
		parts = append(parts, "clear")
	}
	if t.EnterCmd != "" {
		parts = append(parts, t.EnterCmd)
	}
	return strings.Join(parts, " && ")
}
