package gitx

import (
	"path/filepath"
	"testing"

	"github.com/josh-padnick/twig/internal/testutil"
)

func TestAheadOfUpstream(t *testing.T) {
	testutil.RequireGit(t)
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	// A repo with no upstream at all: not comparable.
	loner := filepath.Join(tmp, "loner")
	testutil.NewRepo(t, loner)
	if _, ok := AheadOfUpstream(loner); ok {
		t.Error("repo without upstream should report ok=false")
	}

	// A clone tracking a bare origin: 0 ahead, then 2 after local commits.
	origin := filepath.Join(tmp, "origin.git")
	testutil.Git(t, tmp, "init", "--bare", "-b", "main", origin)
	seed := filepath.Join(tmp, "seed")
	testutil.NewRepo(t, seed)
	testutil.Git(t, seed, "remote", "add", "origin", origin)
	testutil.Git(t, seed, "push", "-u", "origin", "main")

	if n, ok := AheadOfUpstream(seed); !ok || n != 0 {
		t.Errorf("freshly pushed: ahead=%d ok=%v, want 0 true", n, ok)
	}
	testutil.Git(t, seed, "commit", "--allow-empty", "-m", "one")
	testutil.Git(t, seed, "commit", "--allow-empty", "-m", "two")
	if n, ok := AheadOfUpstream(seed); !ok || n != 2 {
		t.Errorf("after two local commits: ahead=%d ok=%v, want 2 true", n, ok)
	}
}
