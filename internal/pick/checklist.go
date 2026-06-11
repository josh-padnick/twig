// A top-down multi-select checklist for the controlling terminal: the
// question and key help render at the top, options below. The init wizard
// uses this instead of the fuzzy finder because go-fuzzyfinder only does
// fzf's bottom-up layout, which buries the question under the options —
// and a ten-item wizard list needs clear keys more than fuzzy filtering.
package pick

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

const (
	ansiClearLine = "\x1b[2K\r"
	ansiBold      = "\x1b[1m"
	ansiDim       = "\x1b[2m"
	ansiGreen     = "\x1b[32m"
	ansiCyan      = "\x1b[36m"
	ansiReset     = "\x1b[0m"
)

// Checklist asks question with a top-down toggle list on /dev/tty and
// returns the checked items. preselected, when non-nil, marks items
// checked up front.
func Checklist[T any](question string, items []T, display func(T) string, preselected func(T) bool) ([]T, error) {
	if len(items) == 0 {
		return nil, nil
	}
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		lines := make([]string, len(items))
		for i, it := range items {
			lines[i] = display(it)
		}
		return nil, &NoTTYError{Lines: lines}
	}
	defer tty.Close()

	oldState, err := term.MakeRaw(int(tty.Fd()))
	if err != nil {
		return nil, fmt.Errorf("cannot enter raw mode: %w", err)
	}
	defer term.Restore(int(tty.Fd()), oldState)

	return runChecklist(tty, tty, question, items, display, preselected)
}

// runChecklist is the terminal-agnostic core, separated so tests can feed
// key bytes and capture output without a real tty. Expects the terminal
// to be in raw mode (lines end \r\n, keys arrive unbuffered).
func runChecklist[T any](in io.Reader, out io.Writer, question string, items []T, display func(T) string, preselected func(T) bool) ([]T, error) {
	checked := make([]bool, len(items))
	cursor := 0
	if preselected != nil {
		for i, it := range items {
			checked[i] = preselected(it)
		}
		for i, c := range checked {
			if c {
				cursor = i
				break
			}
		}
	}

	// Two header lines plus one line per item; redraws move the cursor
	// back up over the item block only.
	fmt.Fprintf(out, "%s%s%s\r\n", ansiBold, question, ansiReset)
	fmt.Fprintf(out, "%s↑/↓ move · space toggles · enter confirms · q cancels%s\r\n", ansiDim, ansiReset)

	draw := func(first bool) {
		if !first {
			fmt.Fprintf(out, "\x1b[%dA", len(items))
		}
		for i, it := range items {
			box := "[ ]"
			if checked[i] {
				box = ansiGreen + "[x]" + ansiReset
			}
			pointer := "  "
			line := fmt.Sprintf("%s %s %s", pointer, box, display(it))
			if i == cursor {
				line = fmt.Sprintf("%s› %s %s%s%s", ansiCyan, box, ansiBold, display(it), ansiReset)
			}
			fmt.Fprintf(out, "%s%s\r\n", ansiClearLine, line)
		}
	}
	draw(true)

	reader := bufio.NewReader(in)
	for {
		b, err := reader.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("reading key: %w", err)
		}
		switch b {
		case 3, 'q': // Ctrl-C, q
			return nil, ErrCancelled
		case '\r', '\n':
			var out []T
			for i, it := range items {
				if checked[i] {
					out = append(out, it)
				}
			}
			return out, nil
		case ' ', '\t':
			checked[cursor] = !checked[cursor]
		case 'j':
			cursor = (cursor + 1) % len(items)
		case 'k':
			cursor = (cursor - 1 + len(items)) % len(items)
		case 'a':
			all := true
			for _, c := range checked {
				if !c {
					all = false
					break
				}
			}
			for i := range checked {
				checked[i] = !all
			}
		case 0x1b: // arrow keys: ESC [ A/B
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
			}
		}
		draw(false)
	}
}
