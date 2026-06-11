// Package testutil builds hermetic git fixtures for resolver and engine
// tests: real repos and linked worktrees created with the system git binary,
// isolated from the developer's global git config, replicating the layouts
// twig must resolve (Claude Code in-repo worktrees, Conductor workspaces,
// plain siblings).
package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// RequireGit skips the test when no git binary is available.
func RequireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

// Git runs a git command in dir with hermetic config and identity, failing
// the test on error and returning trimmed combined output.
func Git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_AUTHOR_NAME=twig-test",
		"GIT_AUTHOR_EMAIL=twig@test.invalid",
		"GIT_COMMITTER_NAME=twig-test",
		"GIT_COMMITTER_EMAIL=twig@test.invalid",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
	return strings.TrimSpace(string(out))
}

// NewRepo creates a git repo at dir with one empty commit on main.
func NewRepo(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	Git(t, dir, "init", "-b", "main")
	Git(t, dir, "commit", "--allow-empty", "-m", "init")
}

// AddWorktree creates branch from HEAD and checks it out as a linked
// worktree at path (which may live anywhere, including outside the repo —
// the Conductor shape).
func AddWorktree(t *testing.T, repoDir, branch, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	Git(t, repoDir, "worktree", "add", "-b", branch, path)
}

// AddDetachedWorktree checks out HEAD as a detached linked worktree at path.
func AddDetachedWorktree(t *testing.T, repoDir, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	Git(t, repoDir, "worktree", "add", "--detach", path)
}

// Fixture is the canonical multi-layout tree resolver tests run against.
type Fixture struct {
	Tmp      string // top-level temp dir (guaranteed outside any git repo)
	Home     string // fake $HOME holding the Conductor layout
	CodeRoot string // the configured scan root (<tmp>/code/fabricahq)
	App      string // main repo (<CodeRoot>/app)

	Matsumoto string // <App>/.claude/worktrees/competent-matsumoto-493452 (branch claude/competent-matsumoto-493452)
	Vibrant   string // <App>/.claude/worktrees/vibrant-curran-5ff1af     (branch claude/vibrant-curran-5ff1af)
	Hotfix    string // <CodeRoot>/app-hotfix                              (branch hotfix/login)
	Docs      string // <CodeRoot>/docs — unrelated repo
	Berlin    string // <Home>/conductor/workspaces/fabrica/berlin        (branch josh/berlin)
	Bordeaux  string // <Home>/conductor/workspaces/fabrica/bordeaux      (branch josh/bordeaux)
}

// StandardFixture builds the canonical layout: a main repo with Claude Code
// in-repo worktrees, a sibling worktree, an unrelated repo, and Conductor
// workspaces under a fake home.
func StandardFixture(t *testing.T) *Fixture {
	t.Helper()
	RequireGit(t)
	// Canonicalize: on macOS t.TempDir() is under /var → /private/var, and
	// git reports resolved paths, so fixture paths must be canonical for
	// string comparisons in tests to hold.
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f := &Fixture{
		Tmp:      tmp,
		Home:     filepath.Join(tmp, "home"),
		CodeRoot: filepath.Join(tmp, "code", "fabricahq"),
	}
	f.App = filepath.Join(f.CodeRoot, "app")
	f.Matsumoto = filepath.Join(f.App, ".claude", "worktrees", "competent-matsumoto-493452")
	f.Vibrant = filepath.Join(f.App, ".claude", "worktrees", "vibrant-curran-5ff1af")
	f.Hotfix = filepath.Join(f.CodeRoot, "app-hotfix")
	f.Docs = filepath.Join(f.CodeRoot, "docs")
	f.Berlin = filepath.Join(f.Home, "conductor", "workspaces", "fabrica", "berlin")
	f.Bordeaux = filepath.Join(f.Home, "conductor", "workspaces", "fabrica", "bordeaux")

	NewRepo(t, f.App)
	AddWorktree(t, f.App, "claude/competent-matsumoto-493452", f.Matsumoto)
	AddWorktree(t, f.App, "claude/vibrant-curran-5ff1af", f.Vibrant)
	AddWorktree(t, f.App, "hotfix/login", f.Hotfix)
	NewRepo(t, f.Docs)
	AddWorktree(t, f.App, "josh/berlin", f.Berlin)
	AddWorktree(t, f.App, "josh/bordeaux", f.Bordeaux)
	return f
}
