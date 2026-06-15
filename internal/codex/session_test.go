package codex

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestParseThreadURI(t *testing.T) {
	cases := []struct {
		in   string
		ok   bool
		want string
	}{
		{"codex://threads/019eccfa-62f5-7733-8fc0-059abf2ea60b", true, "019eccfa-62f5-7733-8fc0-059abf2ea60b"},
		{"codex://thread/abc123", true, "abc123"},                         // singular form
		{"codex://threads/019eccfa-62f5/messages", true, "019eccfa-62f5"}, // trailing path ignored
		{"https://github.com/x/y/pull/1", false, ""},
		{"matsumoto", false, ""},
		{"", false, ""},
	}
	for _, c := range cases {
		id, ok := ParseThreadURI(c.in)
		if ok != c.ok || id != c.want {
			t.Errorf("ParseThreadURI(%q) = (%q,%v), want (%q,%v)", c.in, id, ok, c.want, c.ok)
		}
	}
}

func TestThreadID(t *testing.T) {
	cases := []struct {
		in string
		ok bool
	}{
		{"019eccfa-62f5-7733-8fc0-059abf2ea60b", true},
		{"matsumoto", false},
		{"claude/foo", false},
		{"019eccfa", false}, // too short to be a UUID
	}
	for _, c := range cases {
		if _, ok := ThreadID(c.in); ok != c.ok {
			t.Errorf("ThreadID(%q) ok = %v, want %v", c.in, ok, c.ok)
		}
	}
}

// writeSession lays down a rollout log under the dated tree Codex uses, with a
// realistic session_meta first line.
func writeSession(t *testing.T, sessionsDir, id, cwd string) {
	t.Helper()
	dir := filepath.Join(sessionsDir, "2026", "06", "15")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"timestamp":"2026-06-15T20:30:34.895Z","type":"session_meta","payload":{"id":"` + id + `","cwd":"` + cwd + `","originator":"Codex Desktop"}}
{"type":"event_msg","payload":{"type":"task_started"}}
`
	f := filepath.Join(dir, "rollout-2026-06-15T13-30-21-"+id+".jsonl")
	if err := os.WriteFile(f, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFindReturnsCwd(t *testing.T) {
	tmp := t.TempDir()
	sessions := filepath.Join(tmp, ".codex", "sessions")
	id := "019eccfa-62f5-7733-8fc0-059abf2ea60b"
	cwd := filepath.Join(tmp, "Code", "fabricahq", "app")
	writeSession(t, sessions, id, cwd)

	s, err := Find(sessions, id)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if s.Cwd != cwd || s.ID != id {
		t.Errorf("session = %+v, want cwd=%s id=%s", s, cwd, id)
	}
}

func TestFindNotFound(t *testing.T) {
	tmp := t.TempDir()
	sessions := filepath.Join(tmp, ".codex", "sessions")
	writeSession(t, sessions, "019eccfa-62f5-7733-8fc0-059abf2ea60b", filepath.Join(tmp, "repo"))

	_, err := Find(sessions, "00000000-0000-0000-0000-000000000000")
	var nf *NotFoundError
	if !errors.As(err, &nf) {
		t.Errorf("err = %v, want *NotFoundError", err)
	}
}

func TestFindMissingDir(t *testing.T) {
	// A sessions dir that doesn't exist at all resolves to NotFound, not a
	// walk error.
	_, err := Find(filepath.Join(t.TempDir(), "nope"), "019eccfa-62f5-7733-8fc0-059abf2ea60b")
	var nf *NotFoundError
	if !errors.As(err, &nf) {
		t.Errorf("err = %v, want *NotFoundError", err)
	}
}
