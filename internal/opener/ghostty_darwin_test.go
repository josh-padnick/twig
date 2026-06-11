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
