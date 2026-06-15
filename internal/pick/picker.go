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
	"time"

	"golang.org/x/term"

	"github.com/josh-padnick/twig/internal/resolve"
)

const escapeSequenceTimeout = 50 * time.Millisecond

type readDeadliner interface {
	SetReadDeadline(time.Time) error
}

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

	_, height, err := term.GetSize(int(tty.Fd()))
	if err != nil {
		height = 0
	}
	maxRows := 0
	if height > 2 {
		maxRows = height - 2
	} else if height > 0 {
		maxRows = 1
	}
	idx, err := runPicker(tty, tty, header, items, display, maxRows)
	if err != nil {
		return zero, err
	}
	return items[idx], nil
}

// runPicker is the terminal-agnostic core, separated so tests can feed key
// bytes and capture output without a real tty. It mirrors runChecklist's
// top-down redraw: two header lines, then a bounded item window, redrawing in
// place as the cursor moves. Expects raw mode (keys arrive unbuffered, lines
// end \r\n). Returns the selected item's index.
func runPicker[T any](in io.Reader, out io.Writer, header string, items []T, display func(T) string, maxRows int) (int, error) {
	cursor := 0
	top := 0
	width := len(fmt.Sprintf("%d", len(items))) // digit count for number alignment
	visibleRows := len(items)
	if maxRows > 0 && maxRows < visibleRows {
		visibleRows = maxRows
	}

	if header == "" {
		header = "Select one:"
	}
	fmt.Fprintf(out, "%s%s%s\r\n", ansiBold, header, ansiReset)
	fmt.Fprintf(out, "%s↑/↓ move · 1-9 jump · enter selects · q cancels%s\r\n", ansiDim, ansiReset)

	draw := func(first bool) {
		if !first {
			fmt.Fprintf(out, "\x1b[%dA", visibleRows)
		}
		for row := 0; row < visibleRows; row++ {
			i := top + row
			it := items[i]
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

	keepCursorVisible := func() {
		switch {
		case cursor < top:
			top = cursor
		case cursor >= top+visibleRows:
			top = cursor - visibleRows + 1
		}
	}

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
		case b == 0x1b: // Esc cancels; arrow keys arrive as ESC [ A/B.
			b1, err := readEscapeByte(reader, in)
			if err != nil {
				return 0, ErrCancelled
			}
			b2, err := readEscapeByte(reader, in)
			if err != nil {
				return 0, ErrCancelled
			}
			if b1 != '[' {
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
		keepCursorVisible()
		draw(false)
	}
}

func readEscapeByte(reader *bufio.Reader, in io.Reader) (byte, error) {
	if reader.Buffered() > 0 {
		return reader.ReadByte()
	}
	deadliner, ok := in.(readDeadliner)
	if !ok {
		return 0, io.EOF
	}
	if err := deadliner.SetReadDeadline(time.Now().Add(escapeSequenceTimeout)); err != nil {
		return 0, err
	}
	b, err := reader.ReadByte()
	_ = deadliner.SetReadDeadline(time.Time{})
	return b, err
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
