//go:build darwin

// The Ghostty opener for macOS, driving Ghostty 1.3+'s AppleScript support.
// It deliberately avoids `open -na Ghostty --args --working-directory=...`,
// which drops the arguments when an instance is already running. The entry
// command is passed to osascript as an argv item — never interpolated into
// the script source — and `delay` covers the window-creation race before
// `input text` can reach the new terminal.
package opener

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/josh-padnick/twig/internal/config"
)

type ghostty struct {
	clear       bool
	delayMs     int
	reuseWindow bool
	// inside reports whether twig is itself running inside Ghostty; a field
	// so tests can drive the reuse logic without a real terminal.
	inside func() bool
}

func newGhostty(spec config.Opener) Opener {
	g := &ghostty{clear: true, delayMs: 300, reuseWindow: spec.ReuseWindow, inside: insideGhostty}
	if spec.Clear != nil {
		g.clear = *spec.Clear
	}
	if spec.DelayMs != nil {
		g.delayMs = *spec.DelayMs
	}
	return g
}

// insideGhostty reports whether the current process is running in a Ghostty
// terminal, via the environment Ghostty exports to its shells.
func insideGhostty() bool {
	return os.Getenv("TERM_PROGRAM") == "ghostty" || os.Getenv("GHOSTTY_RESOURCES_DIR") != ""
}

// modeFor resolves the effective target mode. With reuse_window set and twig
// running inside Ghostty, a default (new-window) open becomes a current-tab
// entry so the user stays in the session they're already in. An explicit
// current-tab request (-t) and the not-in-Ghostty case both pass through.
func (g *ghostty) modeFor(requested TargetMode) TargetMode {
	if requested == ModeWindow && g.reuseWindow && g.inside() {
		return ModeCurrentTab
	}
	return requested
}

func (g *ghostty) Name() string        { return "ghostty" }
func (g *ghostty) CanInject() bool     { return true }
func (g *ghostty) CanCurrentTab() bool { return true }

func (g *ghostty) Available() error {
	if _, err := exec.LookPath("osascript"); err != nil {
		return fmt.Errorf("osascript not found")
	}
	home, _ := os.UserHomeDir()
	for _, p := range []string{"/Applications/Ghostty.app", filepath.Join(home, "Applications", "Ghostty.app")} {
		if _, err := os.Stat(p); err == nil {
			return nil
		}
	}
	return fmt.Errorf("Ghostty.app not found in /Applications or ~/Applications")
}

// script builds the AppleScript for the target mode. The command text
// arrives via argv; only the numeric delay is formatted into the source.
func (g *ghostty) script(mode TargetMode) string {
	newWindow := ""
	if mode == ModeWindow {
		newWindow = fmt.Sprintf("    new window\n    delay %.3f\n", float64(g.delayMs)/1000)
	}
	return "on run argv\n" +
		"  set cmdText to item 1 of argv\n" +
		"  tell application \"Ghostty\"\n" +
		"    activate\n" +
		newWindow +
		"    input text cmdText & \"\\n\" to focused terminal of selected tab of front window\n" +
		"  end tell\n" +
		"end run\n"
}

func (g *ghostty) Open(t Target) error {
	line := EntryLine(t, g.clear)
	cmd := exec.Command("osascript", "-", line)
	cmd.Stdin = strings.NewReader(g.script(g.modeFor(t.Mode)))
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ghostty opener: %w", err)
	}
	return nil
}
