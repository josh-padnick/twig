package pick

import (
	"bytes"
	"errors"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// run feeds raw key bytes to the checklist core and returns the selection.
func run(t *testing.T, keys string, items []string, preselected func(string) bool) ([]string, error) {
	t.Helper()
	var out bytes.Buffer
	return runChecklist(strings.NewReader(keys), &out, "Pick some:", items, func(s string) string { return s }, preselected)
}

func TestChecklistTogglesAndConfirms(t *testing.T) {
	items := []string{"alpha", "beta", "gamma"}

	// Toggle the first item, arrow down, toggle the second, confirm.
	got, err := run(t, " \x1b[B \r", items, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"alpha", "beta"}) {
		t.Errorf("got %v", got)
	}
}

func TestChecklistPreselection(t *testing.T) {
	items := []string{"alpha", "beta", "gamma"}
	pre := func(s string) bool { return s == "beta" }

	// Enter alone confirms the preselection.
	got, err := run(t, "\r", items, pre)
	if err != nil || !reflect.DeepEqual(got, []string{"beta"}) {
		t.Errorf("got %v err=%v", got, err)
	}

	// Space first unchecks the preselected item (cursor starts on it).
	got, err = run(t, " \r", items, pre)
	if err != nil || len(got) != 0 {
		t.Errorf("got %v err=%v, want empty", got, err)
	}
}

func TestChecklistVimKeysAndSelectAll(t *testing.T) {
	items := []string{"alpha", "beta"}

	// j moves down, space toggles beta, k moves back up, tab toggles alpha.
	got, err := run(t, "j k\t\r", items, nil)
	if err != nil || !reflect.DeepEqual(got, []string{"alpha", "beta"}) {
		t.Errorf("got %v err=%v", got, err)
	}

	// a selects all; a again clears.
	got, err = run(t, "a\r", items, nil)
	if err != nil || len(got) != 2 {
		t.Errorf("select-all got %v err=%v", got, err)
	}
	got, err = run(t, "aa\r", items, nil)
	if err != nil || len(got) != 0 {
		t.Errorf("select-all twice got %v err=%v", got, err)
	}
}

func TestChecklistCancel(t *testing.T) {
	if _, err := run(t, "q", []string{"alpha"}, nil); !errors.Is(err, ErrCancelled) {
		t.Errorf("q: err = %v, want ErrCancelled", err)
	}
	if _, err := run(t, "\x03", []string{"alpha"}, nil); !errors.Is(err, ErrCancelled) {
		t.Errorf("ctrl-c: err = %v, want ErrCancelled", err)
	}
}

// Every row must put its checkbox in the same column whether or not the
// cursor is on it — a narrower pointer prefix makes rows jiggle sideways.
func TestChecklistRowsStayAligned(t *testing.T) {
	var out bytes.Buffer
	_, err := runChecklist(strings.NewReader("\r"), &out, "Pick:", []string{"alpha", "beta"}, func(s string) string { return s }, nil)
	if err != nil {
		t.Fatal(err)
	}
	stripped := regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`).ReplaceAllString(out.String(), "")
	for _, line := range strings.Split(stripped, "\r\n") {
		idx := strings.Index(line, "[")
		if idx == -1 {
			continue // not an item row
		}
		if idx != 2 {
			t.Errorf("checkbox at column %d, want 2, in line %q", idx, line)
		}
	}
}

func TestChecklistRendersQuestionAndHelpOnTop(t *testing.T) {
	var out bytes.Buffer
	_, err := runChecklist(strings.NewReader("\r"), &out, "Pick some:", []string{"alpha"}, func(s string) string { return s }, nil)
	if err != nil {
		t.Fatal(err)
	}
	rendered := out.String()
	q := strings.Index(rendered, "Pick some:")
	help := strings.Index(rendered, "space toggles")
	item := strings.Index(rendered, "alpha")
	if q == -1 || help == -1 || item == -1 || !(q < help && help < item) {
		t.Errorf("expected question, then help, then items; indexes q=%d help=%d item=%d", q, help, item)
	}
}
