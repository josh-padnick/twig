//go:build !darwin

// Non-macOS stub for the Ghostty opener: Ghostty's scripting interface is
// AppleScript-only, so on Linux the right tool is a command opener such as
// `ghostty -e bash -c {{cmd}}`. The stub exists so configs referencing
// "ghostty" fail with that guidance instead of a missing-symbol surprise.
package opener

import (
	"fmt"

	"github.com/josh-padnick/twig/internal/config"
)

type ghostty struct{}

func newGhostty(config.Opener) Opener { return &ghostty{} }

func (g *ghostty) Name() string                     { return "ghostty" }
func (g *ghostty) CanInject() bool                  { return false }
func (g *ghostty) CanCurrentTab() bool              { return false }
func (g *ghostty) EntersCurrentTab(TargetMode) bool { return false }
func (g *ghostty) LaunchCmd(Target) (string, bool)  { return "", false }

func (g *ghostty) Available() error {
	return fmt.Errorf("the built-in ghostty opener is macOS-only; define a command opener instead, e.g. command = \"ghostty -e bash -c {{cmd}}\"")
}

func (g *ghostty) Open(Target) error {
	return g.Available()
}
