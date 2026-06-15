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

// Infof prints raw informational text to stderr, unprefixed — for wizard
// prose and listings that aren't twig narrating an action. Operational
// steps should use Stepf so they carry the consistent "twig:" marker.
func Infof(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

// Stepf narrates one step of twig's own work to stderr. The "twig:" prefix
// is dimmed on a terminal so the message — and any subprocess output
// (git, setup scripts) interleaved with it — stays readable at a glance,
// and so a reader can pick twig's lines out of a busy log at a glance.
func Stepf(format string, args ...any) {
	fmt.Fprintln(os.Stderr, dim("twig:")+" "+fmt.Sprintf(format, args...))
}

// Boldf prints an emphasized line to stderr, degrading to plain text when
// stderr isn't a terminal (logs, pipes).
func Boldf(format string, args ...any) {
	fmt.Fprintln(os.Stderr, sgr("1", fmt.Sprintf(format, args...)))
}

// Warnf prints a warning to stderr, with the severity highlighted so it
// stands out from ordinary step narration.
func Warnf(format string, args ...any) {
	fmt.Fprintln(os.Stderr, dim("twig:")+" "+sgr("33", "warning:")+" "+fmt.Sprintf(format, args...))
}

// Errorf prints an error to stderr; the "twig:" prefix is reddened so
// failures jump out of the log.
func Errorf(format string, args ...any) {
	fmt.Fprintln(os.Stderr, sgr("31", "twig:")+" "+fmt.Sprintf(format, args...))
}

// sgr wraps s in an ANSI SGR code when stderr is a terminal, and returns it
// untouched otherwise so piped or redirected output stays free of escapes.
func sgr(code, s string) string {
	if !IsTTY(os.Stderr) {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

// dim renders s faintly on a terminal (used for the "twig:" prefix).
func dim(s string) string { return sgr("2", s) }

// Tilde shortens an absolute path under the user's home directory to its ~
// form for display only — never feed the result back to anything that needs
// a real path (e.g. the path `twig cd` writes to stdout).
func Tilde(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	if rest, ok := strings.CutPrefix(path, home+string(os.PathSeparator)); ok {
		return "~" + string(os.PathSeparator) + rest
	}
	return path
}

// IsTTY reports whether f is attached to a terminal.
func IsTTY(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}

// ReadLine prompts on the controlling terminal and returns the trimmed
// line the user types. Errors when no terminal is available.
func ReadLine(prompt string) (string, error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return "", fmt.Errorf("no terminal available for input: %w", err)
	}
	defer tty.Close()
	fmt.Fprint(tty, prompt)
	line, err := bufio.NewReader(tty).ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading input: %w", err)
	}
	return strings.TrimSpace(line), nil
}

// HasTTY reports whether a controlling terminal exists for prompts.
func HasTTY() bool {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return false
	}
	tty.Close()
	return true
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
