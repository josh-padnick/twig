package cli

// Asserts the stdout contract of `twig cd`: exactly the resolved path and a
// newline, nothing else, so `cd "$(twig cd frag)"` always works.

import (
	"bytes"
	"testing"

	"github.com/josh-padnick/twig/internal/testutil"
)

// isolate keeps CLI tests hermetic: config and home point into the fixture
// so the real ~/.config/twig and ~/conductor never leak in.
func isolate(t *testing.T, f *testutil.Fixture) {
	t.Helper()
	t.Setenv("HOME", f.Home)
	t.Setenv("XDG_CONFIG_HOME", f.Home+"/.config")
	t.Setenv("XDG_DATA_HOME", f.Home+"/.local/share")
}

func TestCdPrintsExactlyThePath(t *testing.T) {
	f := testutil.StandardFixture(t)
	isolate(t, f)
	t.Chdir(f.App)

	root := newRootCmd("test", "test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"cd", "matsumoto"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), f.Matsumoto+"\n"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestCdNoMatchWritesNothingToStdout(t *testing.T) {
	f := testutil.StandardFixture(t)
	isolate(t, f)
	t.Chdir(f.Tmp)

	root := newRootCmd("test", "test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"cd", "zzz-no-such-thing"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error for unmatched fragment")
	}
	if out.Len() != 0 {
		t.Errorf("stdout should be empty on failure, got %q", out.String())
	}
}
