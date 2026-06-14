package remote

// Hermetic remote-pickup tests: a local bare repo acts as `origin`
// (ls-remote and fetch work on filesystem paths), so the cloud-session
// scenario — a branch pushed by some other machine, never checked out
// locally — needs no network.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/josh-padnick/twig/internal/gitx"
	"github.com/josh-padnick/twig/internal/resolve"
	"github.com/josh-padnick/twig/internal/testutil"
)

// cloudFixture builds: a bare origin holding main plus a branch pushed by
// a "cloud session", and a local clone that has never fetched that branch.
type cloudFixture struct {
	tmp, origin, clone string
}

func newCloudFixture(t *testing.T, cloudBranch string) *cloudFixture {
	t.Helper()
	testutil.RequireGit(t)
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f := &cloudFixture{tmp: tmp, origin: filepath.Join(tmp, "origin.git"), clone: filepath.Join(tmp, "clone")}

	// Seed the bare origin via a throwaway repo.
	seed := filepath.Join(tmp, "seed")
	testutil.NewRepo(t, seed)
	if err := os.MkdirAll(f.origin, 0o755); err != nil {
		t.Fatal(err)
	}
	testutil.Git(t, f.origin, "init", "--bare", "-b", "main")
	testutil.Git(t, seed, "remote", "add", "origin", f.origin)
	testutil.Git(t, seed, "push", "origin", "main")

	// The local clone exists before the cloud branch does.
	testutil.Git(t, tmp, "clone", f.origin, f.clone)

	// A "cloud session" pushes its branch; the clone never fetches it.
	testutil.Git(t, seed, "switch", "-c", cloudBranch)
	testutil.Git(t, seed, "commit", "--allow-empty", "-m", "cloud work")
	testutil.Git(t, seed, "push", "origin", cloudBranch)
	return f
}

func TestSearchFindsUnfetchedRemoteBranch(t *testing.T) {
	f := newCloudFixture(t, "claude/fix-login-9a8b7c")

	matches := Search("fix-login", []string{f.clone})
	if len(matches) != 1 {
		t.Fatalf("matches = %+v, want exactly one", matches)
	}
	m := matches[0]
	if m.Branch != "claude/fix-login-9a8b7c" || m.Remote != "origin" || m.RepoDir != f.clone {
		t.Errorf("match = %+v", m)
	}
}

func TestSearchTierFiltering(t *testing.T) {
	f := newCloudFixture(t, "feat/api")
	// Push a second, weaker match.
	seed := filepath.Join(f.tmp, "seed")
	testutil.Git(t, seed, "switch", "-c", "feat/api-v2")
	testutil.Git(t, seed, "push", "origin", "feat/api-v2")

	matches := Search("api", []string{f.clone})
	if len(matches) != 1 || matches[0].Branch != "feat/api" {
		t.Errorf("matches = %+v, want only the exact-suffix feat/api", matches)
	}
}

func TestSearchNoMatch(t *testing.T) {
	f := newCloudFixture(t, "claude/something")
	if matches := Search("zzz-none", []string{f.clone}); len(matches) != 0 {
		t.Errorf("matches = %+v, want none", matches)
	}
}

func TestCreateWorktreeFetchesAndChecksOut(t *testing.T) {
	f := newCloudFixture(t, "claude/fix-login-9a8b7c")
	m := Search("fix-login", []string{f.clone})[0]

	path, reused, err := CreateWorktree(m, filepath.Join(".claude", "worktrees", "{{slug}}"))
	if err != nil || reused {
		t.Fatalf("path=%s reused=%v err=%v", path, reused, err)
	}
	want := filepath.Join(f.clone, ".claude", "worktrees", "fix-login-9a8b7c")
	if path != want {
		t.Errorf("path = %s, want %s", path, want)
	}
	branch, err := gitx.CurrentBranch(path)
	if err != nil || branch != "claude/fix-login-9a8b7c" {
		t.Errorf("branch = %q err=%v", branch, err)
	}

	// The new worktree is now resolvable locally — pickup chains into the
	// normal flow.
	r := &resolve.Resolver{Cwd: f.clone, Home: f.tmp}
	res, err := r.Resolve("fix-login")
	if err != nil || res.Chosen == nil || res.Chosen.Path != path {
		t.Errorf("post-pickup resolve: res=%+v err=%v", res, err)
	}

	// Re-running pickup must land in the existing worktree, not fail: the
	// branch is already checked out there.
	again, reused, err := CreateWorktree(m, filepath.Join(".claude", "worktrees", "{{slug}}"))
	if err != nil || !reused || again != path {
		t.Errorf("re-pickup: path=%s reused=%v err=%v, want reuse of %s", again, reused, err, path)
	}
}

func TestCreateWorktreeReusesExistingCheckout(t *testing.T) {
	f := newCloudFixture(t, "claude/already-out-1a2b3c")
	m := Search("already-out", []string{f.clone})[0]

	first, reused, err := CreateWorktree(m, filepath.Join("wt", "{{slug}}"))
	if err != nil || reused {
		t.Fatalf("first create: path=%s reused=%v err=%v", first, reused, err)
	}
	// A second pickup, even with a different target template, lands in the
	// existing checkout — reuse is keyed on the branch, not the target path.
	again, reused, err := CreateWorktree(m, filepath.Join("other", "{{slug}}"))
	if err != nil || !reused || again != first {
		t.Errorf("reuse: path=%s reused=%v err=%v, want reuse of %s", again, reused, err, first)
	}
}

func TestCreateWorktreeWithExistingLocalBranch(t *testing.T) {
	f := newCloudFixture(t, "feat/local-already")
	// The branch already exists locally (e.g. its worktree was removed).
	testutil.Git(t, f.clone, "fetch", "origin", "feat/local-already")
	testutil.Git(t, f.clone, "branch", "feat/local-already", "origin/feat/local-already")

	m := Match{RepoDir: f.clone, Remote: "origin", Branch: "feat/local-already", Tier: resolve.TierExactBranch}
	path, reused, err := CreateWorktree(m, "{{branch}}-wt")
	if err != nil || reused {
		t.Fatalf("path=%s reused=%v err=%v", path, reused, err)
	}
	if filepath.Base(path) != "feat-local-already-wt" {
		t.Errorf("path = %s, want {{branch}} sanitized", path)
	}
	if branch, _ := gitx.CurrentBranch(path); branch != "feat/local-already" {
		t.Errorf("branch = %q", branch)
	}
}

func TestCandidateRepos(t *testing.T) {
	f := newCloudFixture(t, "claude/x")
	root := filepath.Join(f.tmp, "root")
	repoUnderRoot := filepath.Join(root, "proj")
	testutil.NewRepo(t, repoUnderRoot)
	if err := os.MkdirAll(filepath.Join(root, "not-a-repo"), 0o755); err != nil {
		t.Fatal(err)
	}

	repos := CandidateRepos(f.clone, []string{root})
	if len(repos) != 2 || repos[0] != f.clone || repos[1] != repoUnderRoot {
		t.Errorf("repos = %v, want [clone, proj]", repos)
	}

	// Outside any repo: only root-level repos.
	repos = CandidateRepos(f.tmp+"/nowhere", []string{root})
	if len(repos) != 1 || repos[0] != repoUnderRoot {
		t.Errorf("repos = %v", repos)
	}
}
