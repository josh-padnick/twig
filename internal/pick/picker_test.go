package pick

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// pickRun feeds raw key bytes to the picker core and returns the selected
// index plus everything drawn, so tests can assert on both.
func pickRun(t *testing.T, keys string, items []string) (int, string, error) {
	t.Helper()
	var out bytes.Buffer
	idx, err := runPicker(strings.NewReader(keys), &out, "Pick one:", items, func(s string) string { return s })
	return idx, out.String(), err
}

func TestPickerArrowsAndEnter(t *testing.T) {
	items := []string{"alpha", "beta", "gamma"}
	cases := []struct {
		name string
		keys string
		want int
	}{
		{"enter selects cursor at top", "\r", 0},
		{"down then enter", "\x1b[B\r", 1},
		{"j moves down", "j\r", 1},
		{"j then k returns to top", "jk\r", 0},
		{"k wraps to bottom", "k\r", 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			idx, _, err := pickRun(t, c.keys, items)
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			if idx != c.want {
				t.Errorf("idx = %d, want %d", idx, c.want)
			}
		})
	}
}

func TestPickerNumberJumpSelects(t *testing.T) {
	items := []string{"alpha", "beta", "gamma"}
	// "3" jumps straight to the third item, no enter needed.
	idx, _, err := pickRun(t, "3", items)
	if err != nil || idx != 2 {
		t.Errorf("idx = %d err = %v, want 2", idx, err)
	}
	// An out-of-range digit is ignored; the following enter takes the cursor.
	idx, _, err = pickRun(t, "9\r", items)
	if err != nil || idx != 0 {
		t.Errorf("idx = %d err = %v, want 0", idx, err)
	}
}

func TestPickerCancel(t *testing.T) {
	items := []string{"alpha", "beta"}
	for _, keys := range []string{"q", "\x03" /* Ctrl-C */} {
		if _, _, err := pickRun(t, keys, items); !errors.Is(err, ErrCancelled) {
			t.Errorf("keys %q: err = %v, want ErrCancelled", keys, err)
		}
	}
}

func TestPickerRendersHeaderAndItems(t *testing.T) {
	items := []string{"alpha", "beta", "gamma"}
	_, out, err := pickRun(t, "\r", items)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Pick one:", "alpha", "beta", "gamma", "enter selects", "q cancels"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q; got:\n%s", want, out)
		}
	}
}
