package cli

// Asserts the stdout contract of `twig cd`: exactly the resolved path and a
// newline, nothing else, so `cd "$(twig cd frag)"` always works.

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/josh-padnick/twig/internal/resolve"
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

func TestBareCodexThreadSurfacesStaleCwd(t *testing.T) {
	f := testutil.StandardFixture(t)
	isolate(t, f)
	t.Chdir(f.Tmp)

	id := "019eccfa-62f5-7733-8fc0-059abf2ea60b"
	writeCodexSession(t, f.Home, id, filepath.Join(f.Tmp, "gone"))

	_, err := resolveFragmentOrRemote(id, false, false)
	if err == nil {
		t.Fatal("expected stale Codex cwd error")
	}
	var noMatch *resolve.NoMatchError
	if errors.As(err, &noMatch) {
		t.Fatalf("err = %v, want Codex stale cwd error", err)
	}
	if !strings.Contains(err.Error(), "directory no longer exists") {
		t.Fatalf("err = %v, want stale cwd message", err)
	}
}

func TestCdSessionTitleSearchPrintsCwd(t *testing.T) {
	f := testutil.StandardFixture(t)
	isolate(t, f)
	t.Chdir(f.Tmp)

	id := "019eccfa-62f5-7733-8fc0-059abf2ea60b"
	writeCodexSession(t, f.Home, id, f.App) // f.App exists, so the cwd is live
	writeCodexIndex(t, f.Home, id, "Frame PR 142 review")

	root := newRootCmd("test", "test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"cd", "-s", "frame pr 142"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), f.App+"\n"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func writeCodexSession(t *testing.T, home, id, cwd string) {
	t.Helper()
	dir := filepath.Join(home, ".codex", "sessions", "2026", "06", "15")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"type":"session_meta","payload":{"id":"` + id + `","cwd":"` + cwd + `"}}
`
	path := filepath.Join(dir, "rollout-2026-06-15T13-30-21-"+id+".jsonl")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeCodexIndex(t *testing.T, home, id, title string) {
	t.Helper()
	dir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	line := `{"id":"` + id + `","thread_name":"` + title + `","updated_at":"2026-06-15T20:30:42Z"}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "session_index.jsonl"), []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}
}
