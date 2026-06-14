//go:build darwin

package opener

import (
	"strings"
	"testing"

	"github.com/josh-padnick/twig/internal/config"
)

func TestGhosttyScriptModes(t *testing.T) {
	g := newGhostty(config.Opener{}).(*ghostty)

	window := g.script(ModeWindow)
	for _, want := range []string{"new window", "delay 0.300", "input text cmdText", "on run argv"} {
		if !strings.Contains(window, want) {
			t.Errorf("window script missing %q:\n%s", want, window)
		}
	}

	tab := g.script(ModeCurrentTab)
	if strings.Contains(tab, "new window") || strings.Contains(tab, "delay") {
		t.Errorf("current-tab script must not create a window:\n%s", tab)
	}
	if !strings.Contains(tab, "input text cmdText") {
		t.Errorf("current-tab script must still inject:\n%s", tab)
	}
}

func TestGhosttyConfigOverrides(t *testing.T) {
	clear := false
	delay := 500
	g := newGhostty(config.Opener{Clear: &clear, DelayMs: &delay}).(*ghostty)
	if g.clear || g.delayMs != 500 {
		t.Errorf("ghostty = %+v", g)
	}
	if !strings.Contains(g.script(ModeWindow), "delay 0.500") {
		t.Error("delay override not applied")
	}
}

func TestGhosttyReuseWindow(t *testing.T) {
	on := newGhostty(config.Opener{ReuseWindow: true}).(*ghostty)

	// Inside Ghostty, a default (new-window) open is redirected to the
	// current tab so the user stays in the session they're already in.
	on.inside = func() bool { return true }
	if got := on.modeFor(ModeWindow); got != ModeCurrentTab {
		t.Errorf("reuse inside Ghostty: mode = %v, want current-tab", got)
	}
	// An explicit -t is always honored.
	if got := on.modeFor(ModeCurrentTab); got != ModeCurrentTab {
		t.Errorf("explicit -t: mode = %v, want current-tab", got)
	}
	// Outside Ghostty there's no window to reuse — open a new one.
	on.inside = func() bool { return false }
	if got := on.modeFor(ModeWindow); got != ModeWindow {
		t.Errorf("reuse outside Ghostty: mode = %v, want window", got)
	}

	// Without the option, twig opens a new window even inside Ghostty.
	off := newGhostty(config.Opener{}).(*ghostty)
	off.inside = func() bool { return true }
	if got := off.modeFor(ModeWindow); got != ModeWindow {
		t.Errorf("reuse off: mode = %v, want window", got)
	}
}

func TestGhosttyClearOnlyForNewWindow(t *testing.T) {
	clear := true
	g := newGhostty(config.Opener{Clear: &clear, ReuseWindow: true}).(*ghostty)
	tgt := Target{Dir: "/code/app", EnterCmd: "twig enter"}

	// A new window (twig not running inside Ghostty) clears for a fresh start.
	g.inside = func() bool { return false }
	if line := g.entryLine(tgt); !strings.Contains(line, "clear") {
		t.Errorf("new-window line should clear: %q", line)
	}
	// Reusing the current window must not clear — that scrollback is the
	// user's working context.
	g.inside = func() bool { return true }
	if line := g.entryLine(tgt); strings.Contains(line, "clear") {
		t.Errorf("reused-window line must not clear: %q", line)
	}
	// An explicit -t enters the current tab and likewise skips the clear.
	if line := g.entryLine(Target{Dir: "/code/app", EnterCmd: "twig enter", Mode: ModeCurrentTab}); strings.Contains(line, "clear") {
		t.Errorf("-t line must not clear: %q", line)
	}
}

func TestInsideGhostty(t *testing.T) {
	t.Setenv("GHOSTTY_RESOURCES_DIR", "")
	t.Setenv("TERM_PROGRAM", "ghostty")
	if !insideGhostty() {
		t.Error("TERM_PROGRAM=ghostty should be detected")
	}
	t.Setenv("TERM_PROGRAM", "iTerm.app")
	if insideGhostty() {
		t.Error("a non-Ghostty TERM_PROGRAM should not be detected")
	}
	t.Setenv("GHOSTTY_RESOURCES_DIR", "/Applications/Ghostty.app/Contents/Resources")
	if !insideGhostty() {
		t.Error("GHOSTTY_RESOURCES_DIR should be detected")
	}
}
