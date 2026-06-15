// Package pick is the interactive chooser for ambiguous resolutions. It
// renders a top-down list right where the cursor is — a header and key help
// on top, options below, the current row highlighted — rather than taking
// over an alternate screen the way a fuzzy finder would, so the choice stays
// in the scrollback. When no controlling terminal exists it fails with the
// candidate list on stderr so scripted callers get a useful error instead of
// a hung prompt.
package pick

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/josh-padnick/twig/internal/resolve"
)

// ErrCancelled is returned when the user aborts the picker (Esc/q/Ctrl-C).
var ErrCancelled = errors.New("cancelled")

// NoTTYError reports an ambiguous choice in a non-interactive context,
// carrying rendered lines so the error message can enumerate the options.
type NoTTYError struct {
	Lines []string
}

func (e *NoTTYError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d matches and no terminal is available to pick one:", len(e.Lines))
	for _, line := range e.Lines {
		fmt.Fprintf(&b, "\n  %s", line)
	}
	return b.String()
}

// OneOf returns the only item, or runs the interactive picker over several.
// header is the prompt shown above the list (e.g. what's being chosen).
func OneOf[T any](items []T, display func(T) string, header string) (T, error) {
	var zero T
	switch len(items) {
	case 0:
		return zero, errors.New("no candidates to pick from")
	case 1:
		return items[0], nil
	}

	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		lines := make([]string, len(items))
		for i, it := range items {
			lines[i] = display(it)
		}
		return zero, &NoTTYError{Lines: lines}
	}
	defer tty.Close()

	oldState, err := term.MakeRaw(int(tty.Fd()))
	if err != nil {
		return zero, fmt.Errorf("cannot enter raw mode: %w", err)
	}
	defer term.Restore(int(tty.Fd()), oldState)

	idx, err := runPicker(tty, tty, header, items, display)
	if err != nil {
		return zero, err
	}
	return items[idx], nil
}

// runPicker is the terminal-agnostic core, separated so tests can feed key
// bytes and capture output without a real tty. It mirrors runChecklist's
// top-down redraw: two header lines, then one line per item, redrawing the
// item block in place as the cursor moves. Expects raw mode (keys arrive
// unbuffered, lines end \r\n). Returns the selected item's index.
func runPicker[T any](in io.Reader, out io.Writer, header string, items []T, display func(T) string) (int, error) {
	cursor := 0
	width := len(fmt.Sprintf("%d", len(items))) // digit count for number alignment

	if header == "" {
		header = "Select one:"
	}
	fmt.Fprintf(out, "%s%s%s\r\n", ansiBold, header, ansiReset)
	fmt.Fprintf(out, "%s↑/↓ move · 1-9 jump · enter selects · q cancels%s\r\n", ansiDim, ansiReset)

	draw := func(first bool) {
		if !first {
			fmt.Fprintf(out, "\x1b[%dA", len(items))
		}
		for i, it := range items {
			// The pointer must occupy exactly the same columns as the indent,
			// or rows shift sideways as the cursor moves.
			prefix := "  "
			num := fmt.Sprintf("%s%*d.%s ", ansiDim, width, i+1, ansiReset)
			name := display(it)
			if i == cursor {
				prefix = ansiCyan + "› " + ansiReset
				num = fmt.Sprintf("%*d. ", width, i+1)
				name = ansiBold + name + ansiReset
			}
			fmt.Fprintf(out, "%s%s%s%s\r\n", ansiClearLine, prefix, num, name)
		}
	}
	draw(true)

	reader := bufio.NewReader(in)
	for {
		b, err := reader.ReadByte()
		if err != nil { // EOF / Ctrl-D
			return 0, ErrCancelled
		}
		switch {
		case b == 3 || b == 'q': // Ctrl-C, q
			return 0, ErrCancelled
		case b == '\r' || b == '\n':
			return cursor, nil
		case b == 'j':
			cursor = (cursor + 1) % len(items)
		case b == 'k':
			cursor = (cursor - 1 + len(items)) % len(items)
		case b >= '1' && b <= '9': // direct jump-and-select
			if n := int(b - '1'); n < len(items) {
				return n, nil
			}
			continue
		case b == 0x1b: // arrow keys: ESC [ A/B
			b1, err1 := reader.ReadByte()
			b2, err2 := reader.ReadByte()
			if err1 != nil || err2 != nil || b1 != '[' {
				continue
			}
			switch b2 {
			case 'A':
				cursor = (cursor - 1 + len(items)) % len(items)
			case 'B':
				cursor = (cursor + 1) % len(items)
			default:
				continue
			}
		default:
			continue
		}
		draw(false)
	}
}

// One picks among worktree candidates. header is the prompt shown above the
// list.
func One(cands []resolve.Candidate, header string) (resolve.Candidate, error) {
	return OneOf(cands, Display, header)
}

// Display renders a candidate as "path  [branch]", shortening the home
// prefix to ~ for readability in the picker and error lists.
func Display(c resolve.Candidate) string {
	p := c.Path
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if rel, ok := strings.CutPrefix(p, home+string(os.PathSeparator)); ok {
			p = "~" + string(os.PathSeparator) + rel
		}
	}
	if c.Branch != "" {
		return p + "  [" + c.Branch + "]"
	}
	return p
}
