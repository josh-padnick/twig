// Package ui centralizes user-facing output. Everything twig says lands on
// stderr (or /dev/tty for prompts) so that command substitution such as
// `cd "$(twig cd foo)"` only ever captures the resolved path that the cd
// subcommand deliberately writes to stdout.
package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// Infof prints an informational message to stderr.
func Infof(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

// Warnf prints a warning to stderr.
func Warnf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "twig: warning: "+format+"\n", args...)
}

// Errorf prints an error to stderr.
func Errorf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "twig: "+format+"\n", args...)
}

// IsTTY reports whether f is attached to a terminal.
func IsTTY(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}

// ConfirmYN asks a yes/no question on the controlling terminal so the answer
// can't be satisfied by piped stdin and the prompt can't pollute captured
// stdout. Returns an error when no terminal is available (non-interactive).
func ConfirmYN(prompt string, def bool) (bool, error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return false, fmt.Errorf("no terminal available for confirmation: %w", err)
	}
	defer tty.Close()

	hint := "y/N"
	if def {
		hint = "Y/n"
	}
	fmt.Fprintf(tty, "%s [%s] ", prompt, hint)

	line, err := bufio.NewReader(tty).ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("reading confirmation: %w", err)
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	case "n", "no":
		return false, nil
	case "":
		return def, nil
	default:
		return false, nil
	}
}
