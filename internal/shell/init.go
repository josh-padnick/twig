// Package shell emits the tw() integration function (direnv/zoxide style):
// a tiny shell function that cd's into a resolved worktree in the current
// shell — something a plain binary cannot do — and then runs `twig enter`.
// Users add `eval "$(twig shell-init zsh)"` to their shell config.
package shell

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed templates/*
var templates embed.FS

// Render returns the integration snippet for shellName with the function
// named cmd and twig invoked via twigPath (quoted per shell, since the
// binary may live at a path with spaces).
func Render(shellName, cmd, twigPath string) (string, error) {
	data, err := templates.ReadFile("templates/tw." + shellName)
	if err != nil {
		return "", fmt.Errorf("unsupported shell %q (supported: zsh, bash, fish)", shellName)
	}
	quoted := posixQuote(twigPath)
	if shellName == "fish" {
		quoted = fishQuote(twigPath)
	}
	out := strings.ReplaceAll(string(data), "%CMD%", cmd)
	return strings.ReplaceAll(out, "%TWIG%", quoted), nil
}

// posixQuote single-quotes for zsh/bash using the '\” idiom.
func posixQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// fishQuote single-quotes for fish, whose only escapes inside single
// quotes are \' and \\.
func fishQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "'", `\'`)
	return "'" + s + "'"
}
