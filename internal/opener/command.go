// The generic "command" opener: a user-supplied template executed through
// bash, with {{dir}} replaced by the shell-quoted worktree path and {{cmd}}
// by the shell-quoted entry command (cd + twig enter). {{cmd}} makes a
// terminal template injection-capable; editor/browser templates use only
// {{dir}} or a fixed URL. This single kind covers VSCode, Cursor, iTerm,
// kitty, WezTerm, and Linux Ghostty without dedicated Go code.
package opener

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/josh-padnick/twig/internal/config"
)

type command struct {
	name     string
	template string
}

func newCommand(name string, spec config.Opener) (Opener, error) {
	if strings.TrimSpace(spec.Command) == "" {
		return nil, fmt.Errorf("opener %q has kind \"command\" but no command", name)
	}
	return &command{name: name, template: spec.Command}, nil
}

func (c *command) Name() string        { return c.name }
func (c *command) CanInject() bool     { return strings.Contains(c.template, "{{cmd}}") }
func (c *command) CanCurrentTab() bool { return false }

// EntersCurrentTab: a command opener always runs in its own process/window.
func (c *command) EntersCurrentTab(TargetMode) bool { return false }

// LaunchCmd offers this opener's launch for folding into a current-tab
// terminal's entry line. Only non-injecting launchers (editors, browsers)
// fold cleanly; a {{cmd}} template spawns its own terminal, so it runs on
// its own instead.
func (c *command) LaunchCmd(t Target) (string, bool) {
	if c.CanInject() {
		return "", false
	}
	return c.buildLine(t), true
}

// Available checks that the template's program exists on PATH.
func (c *command) Available() error {
	fields := strings.Fields(c.template)
	if len(fields) == 0 {
		return fmt.Errorf("opener %q has an empty command", c.name)
	}
	if _, err := exec.LookPath(fields[0]); err != nil {
		return fmt.Errorf("opener %q: %q not found on PATH", c.name, fields[0])
	}
	return nil
}

// buildLine renders the template for a target. Kept separate from Open so
// the substitution logic is testable without executing anything.
func (c *command) buildLine(t Target) string {
	line := strings.ReplaceAll(c.template, "{{dir}}", Quote(t.Dir))
	if strings.Contains(line, "{{cmd}}") {
		line = strings.ReplaceAll(line, "{{cmd}}", Quote(EntryLine(t, false)))
	}
	return line
}

func (c *command) Open(t Target) error {
	if t.Mode == ModeCurrentTab {
		return fmt.Errorf("opener %q cannot enter the current tab — use the tw shell function, or the ghostty opener", c.name)
	}
	cmd := exec.Command("bash", "-c", c.buildLine(t))
	// Both streams go to stderr: opener chatter must never pollute stdout.
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("opener %q: %w", c.name, err)
	}
	return nil
}
