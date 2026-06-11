package resolve

// Fast scan tests over fake directory trees: a one-line .git file is enough
// for the scanner's worktree check, so no git binary is needed here.

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// fakeWorktree creates dir with a .git file so the scanner accepts it.
func fakeWorktree(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: /nonexistent\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanPatterns(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	root := filepath.Join(tmp, "root")
	cwd := filepath.Join(tmp, "cwd") // guaranteed non-repo working dir
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}

	alpha := filepath.Join(root, ".claude", "worktrees", "alpha-123")
	beta := filepath.Join(root, "proj", ".claude", "worktrees", "beta-456")
	gamma := filepath.Join(root, "gamma-789")
	delta := filepath.Join(root, "proj2", "delta-000")
	epsilon := filepath.Join(home, "conductor", "workspaces", "proj", "epsilon-111")
	for _, d := range []string{alpha, beta, gamma, delta, epsilon} {
		fakeWorktree(t, d)
	}
	// A directory without .git must be rejected.
	if err := os.MkdirAll(filepath.Join(root, "zeta-nogit"), 0o755); err != nil {
		t.Fatal(err)
	}

	r := &Resolver{Cwd: cwd, Home: home, Roots: []string{root}, Providers: Builtin}

	tests := []struct {
		name, frag, want string
	}{
		{"root/.claude/worktrees pattern", "alpha", alpha},
		{"root/*/.claude/worktrees pattern", "beta", beta},
		{"root/* pattern", "gamma", gamma},
		{"root/*/* pattern", "delta", delta},
		{"conductor fixed location", "epsilon", epsilon},
		{"case-insensitive", "ALPHA", alpha},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := r.Resolve(tt.frag)
			if err != nil {
				t.Fatal(err)
			}
			if res.Chosen == nil || res.Chosen.Path != tt.want {
				t.Errorf("got %+v, want %s", res.Chosen, tt.want)
			}
		})
	}

	t.Run("dir without .git is rejected", func(t *testing.T) {
		_, err := r.Resolve("zeta")
		var noMatch *NoMatchError
		if !errors.As(err, &noMatch) {
			t.Errorf("got %v, want NoMatchError", err)
		}
	})
}

func TestScanExactDirShortCircuitsSubstring(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "root")
	cwd := filepath.Join(tmp, "cwd")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	exact := filepath.Join(root, "widget")
	longer := filepath.Join(root, "widget-extra")
	fakeWorktree(t, exact)
	fakeWorktree(t, longer)

	r := &Resolver{Cwd: cwd, Home: filepath.Join(tmp, "nohome"), Roots: []string{root}, Providers: Builtin}
	res, err := r.Resolve("widget")
	if err != nil {
		t.Fatal(err)
	}
	if res.Chosen == nil || res.Chosen.Path != exact {
		t.Errorf("got %+v, want exact basename %s to win without a picker", res.Chosen, exact)
	}
}

// Conductor leaves renamed workspaces behind as symlinks to the new
// directory; the scanner must follow them, and dedupe must collapse the
// alias and its target into the canonical path.
func TestScanFollowsWorkspaceSymlinks(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	cwd := filepath.Join(tmp, "cwd")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(home, "conductor", "workspaces", "proj")
	target := filepath.Join(project, "memphis")
	fakeWorktree(t, target)
	alias := filepath.Join(project, "old-name")
	if err := os.Symlink(target, alias); err != nil {
		t.Fatal(err)
	}

	r := &Resolver{Cwd: cwd, Home: home, Providers: Builtin}

	res, err := r.Resolve("old-name")
	if err != nil {
		t.Fatal(err)
	}
	if res.Chosen == nil || res.Chosen.Path != alias {
		t.Errorf("alias fragment: got %+v, want %s", res.Chosen, alias)
	}

	// Both spellings exist; All must collapse them to the canonical target.
	all := r.All()
	if len(all) != 1 {
		t.Fatalf("All() = %d candidates, want 1 after symlink dedupe: %+v", len(all), all)
	}
	if got, err := filepath.EvalSymlinks(all[0].Path); err != nil || got != mustEval(t, target) {
		t.Errorf("All() kept %s, want canonical %s", all[0].Path, target)
	}
}

func mustEval(t *testing.T, p string) string {
	t.Helper()
	out, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestAllEnumeratesEverything(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "root")
	cwd := filepath.Join(tmp, "cwd")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	a := filepath.Join(root, "one")
	b := filepath.Join(root, "two")
	fakeWorktree(t, a)
	fakeWorktree(t, b)
	if err := os.MkdirAll(filepath.Join(root, "not-a-worktree"), 0o755); err != nil {
		t.Fatal(err)
	}

	r := &Resolver{Cwd: cwd, Home: filepath.Join(tmp, "nohome"), Roots: []string{root}, Providers: Builtin}
	all := r.All()
	if len(all) != 2 {
		t.Fatalf("All() = %d candidates, want 2: %+v", len(all), all)
	}
}
