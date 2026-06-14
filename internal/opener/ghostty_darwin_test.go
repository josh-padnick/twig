//go:build darwin

package opener

import (
	"os"
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
	if line, _ := g.entryLine(tgt); !strings.Contains(line, "clear") {
		t.Errorf("new-window line should clear: %q", line)
	}
	// Reusing the current window must not clear — that scrollback is the
	// user's working context.
	g.inside = func() bool { return true }
	if line, _ := g.entryLine(tgt); strings.Contains(line, "clear") {
		t.Errorf("reused-window line must not clear: %q", line)
	}
	// An explicit -t enters the current tab and likewise skips the clear.
	if line, _ := g.entryLine(Target{Dir: "/code/app", EnterCmd: "twig enter", Mode: ModeCurrentTab}); strings.Contains(line, "clear") {
		t.Errorf("-t line must not clear: %q", line)
	}
}

func TestGhosttyEntersCurrentTabAndFolds(t *testing.T) {
	g := newGhostty(config.Opener{ReuseWindow: true}).(*ghostty)
	g.inside = func() bool { return true }

	if !g.EntersCurrentTab(ModeWindow) {
		t.Error("reuse_window inside Ghostty should enter the current tab")
	}
	g.inside = func() bool { return false }
	if g.EntersCurrentTab(ModeWindow) {
		t.Error("outside Ghostty a default open is a new window")
	}

	// Folded launches land between the cd and the entry command, in order.
	g.inside = func() bool { return true }
	line, inject := g.entryLine(Target{Dir: "/code/other", EnterCmd: "twig enter", Fold: []string{"cursor '/code/other'"}})
	if !inject || line != "cd '/code/other' && cursor '/code/other' && twig enter" {
		t.Errorf("folded current-tab line = %q inject=%v", line, inject)
	}
}

func TestGhosttyReuseSkipsSelfCd(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	g := newGhostty(config.Opener{ReuseWindow: true}).(*ghostty)
	g.inside = func() bool { return true }

	// Already in the target worktree, nothing to run: inject nothing, so no
	// redundant cd is typed (and echoed) into the prompt.
	if line, inject := g.entryLine(Target{Dir: cwd}); inject {
		t.Errorf("same-dir reuse should inject nothing, got %q", line)
	}
	// Already there but with setup/run to do: inject just the enter command.
	if line, inject := g.entryLine(Target{Dir: cwd, EnterCmd: "twig enter"}); !inject || line != "twig enter" {
		t.Errorf("same-dir reuse with enter: line=%q inject=%v, want just the enter command", line, inject)
	}
	// A different worktree still cds.
	if line, inject := g.entryLine(Target{Dir: cwd + "/elsewhere"}); !inject || !strings.Contains(line, "cd ") {
		t.Errorf("different-dir reuse should cd: line=%q inject=%v", line, inject)
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
