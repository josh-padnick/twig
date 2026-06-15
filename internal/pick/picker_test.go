package pick

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

// pickRun feeds raw key bytes to the picker core and returns the selected
// index plus everything drawn, so tests can assert on both.
func pickRun(t *testing.T, keys string, items []string) (int, string, error) {
	t.Helper()
	return pickRunRows(t, keys, items, 0)
}

func pickRunRows(t *testing.T, keys string, items []string, maxRows int) (int, string, error) {
	t.Helper()
	var out bytes.Buffer
	idx, err := runPicker(strings.NewReader(keys), &out, "Pick one:", items, func(s string) string { return s }, maxRows)
	return idx, out.String(), err
}

type escThenBlockReader struct {
	unblock chan struct{}
	sentEsc bool
}

func (r *escThenBlockReader) Read(p []byte) (int, error) {
	if !r.sentEsc {
		r.sentEsc = true
		p[0] = 0x1b
		return 1, nil
	}
	<-r.unblock
	return 0, errors.New("unblocked")
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
	for _, keys := range []string{"q", "\x03" /* Ctrl-C */, "\x1b" /* Esc */} {
		if _, _, err := pickRun(t, keys, items); !errors.Is(err, ErrCancelled) {
			t.Errorf("keys %q: err = %v, want ErrCancelled", keys, err)
		}
	}
}

func TestPickerBareEscCancelsWithoutMoreInput(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()

	done := make(chan error, 1)
	go func() {
		var out bytes.Buffer
		_, err := runPicker(r, &out, "Pick one:", []string{"alpha", "beta"}, func(s string) string { return s }, 0)
		done <- err
	}()

	if _, err := w.Write([]byte{0x1b}); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-done:
		if !errors.Is(err, ErrCancelled) {
			t.Fatalf("err = %v, want ErrCancelled", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("picker blocked after bare Esc")
	}
}

func TestPickerBareEscCancelsWhenDeadlineUnsupported(t *testing.T) {
	r := &escThenBlockReader{unblock: make(chan struct{})}
	defer close(r.unblock)

	done := make(chan error, 1)
	go func() {
		var out bytes.Buffer
		_, err := runPicker(r, &out, "Pick one:", []string{"alpha", "beta"}, func(s string) string { return s }, 0)
		done <- err
	}()

	select {
	case err := <-done:
		if !errors.Is(err, ErrCancelled) {
			t.Fatalf("err = %v, want ErrCancelled", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("picker blocked after bare Esc without read deadlines")
	}
}

func TestPickerRedrawsBoundedViewport(t *testing.T) {
	items := []string{"alpha", "beta", "gamma", "delta"}
	idx, out, err := pickRunRows(t, "jj\r", items, 2)
	if err != nil {
		t.Fatal(err)
	}
	if idx != 2 {
		t.Fatalf("idx = %d, want 2", idx)
	}
	if !strings.Contains(out, "gamma") {
		t.Fatalf("output missing selected item; got:\n%s", out)
	}
	if strings.Contains(out, "delta") {
		t.Fatalf("bounded viewport rendered offscreen item; got:\n%s", out)
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
