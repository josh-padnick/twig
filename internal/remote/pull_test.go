package remote

// PR-pickup tests stay hermetic the same way the rest of the package does: a
// local bare repo whose path ends in <owner>/<repo> stands in for the GitHub
// remote, and a hand-written refs/pull/<n>/head ref stands in for the ref
// GitHub maintains for every PR. No network, no API token.

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/josh-padnick/twig/internal/testutil"
)

func TestParsePullURL(t *testing.T) {
	cases := []struct {
		in    string
		ok    bool
		host  string
		owner string
		repo  string
		num   int
	}{
		{"https://github.com/fabricahq/app/pull/140", true, "github.com", "fabricahq", "app", 140},
		{"https://github.com/fabricahq/app/pull/140/changes", true, "github.com", "fabricahq", "app", 140},
		{"https://github.com/fabricahq/app/pull/140/files", true, "github.com", "fabricahq", "app", 140},
		{"https://github.com/fabricahq/app/pull/140?diff=split", true, "github.com", "fabricahq", "app", 140},
		{"http://github.example.com/org/repo/pull/7", true, "github.example.com", "org", "repo", 7},
		{"https://github.com/fabricahq/app/issues/140", false, "", "", "", 0},
		{"claude/per-request-identity", false, "", "", "", 0},
		{"", false, "", "", "", 0},
	}
	for _, c := range cases {
		pr, ok := ParsePullURL(c.in)
		if ok != c.ok {
			t.Errorf("ParsePullURL(%q) ok=%v, want %v", c.in, ok, c.ok)
			continue
		}
		if !ok {
			continue
		}
		if pr.Host != c.host || pr.Owner != c.owner || pr.Repo != c.repo || pr.Number != c.num {
			t.Errorf("ParsePullURL(%q) = %+v", c.in, pr)
		}
	}
}

func TestParseRepoURL(t *testing.T) {
	cases := []struct {
		in    string
		host  string
		owner string
		repo  string
		ok    bool
	}{
		{"https://github.com/fabricahq/app.git", "github.com", "fabricahq", "app", true},
		{"https://github.com/fabricahq/app", "github.com", "fabricahq", "app", true},
		{"git@github.com:fabricahq/app.git", "github.com", "fabricahq", "app", true},
		{"ssh://git@github.com/fabricahq/app.git", "github.com", "fabricahq", "app", true},
		{"https://user@github.com:443/fabricahq/app.git", "github.com", "fabricahq", "app", true},
		{"/Users/me/tmp/remotes/fabricahq/app.git", "", "fabricahq", "app", true},
		{"not-a-url", "", "", "", false},
	}
	for _, c := range cases {
		host, owner, repo, ok := parseRepoURL(c.in)
		if ok != c.ok || host != c.host || owner != c.owner || repo != c.repo {
			t.Errorf("parseRepoURL(%q) = (%q,%q,%q,%v), want (%q,%q,%q,%v)",
				c.in, host, owner, repo, ok, c.host, c.owner, c.repo, c.ok)
		}
	}
}

// pullFixture builds a bare origin at <tmp>/remotes/fabricahq/app.git (so the
// remote URL parses to fabricahq/app), pushes a PR branch, and writes the
// refs/pull/140/head ref GitHub would maintain. The clone never fetched the
// branch — the cloud-PR scenario.
type pullFixture struct {
	tmp, origin, clone, branch string
}

func newPullFixture(t *testing.T, branch string, prNum int) *pullFixture {
	t.Helper()
	testutil.RequireGit(t)
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f := &pullFixture{
		tmp:    tmp,
		origin: filepath.Join(tmp, "remotes", "fabricahq", "app.git"),
		clone:  filepath.Join(tmp, "clone"),
		branch: branch,
	}

	seed := filepath.Join(tmp, "seed")
	testutil.NewRepo(t, seed)
	testutil.Git(t, tmp, "init", "--bare", "-b", "main", f.origin)
	testutil.Git(t, seed, "remote", "add", "origin", f.origin)
	testutil.Git(t, seed, "push", "origin", "main")

	testutil.Git(t, tmp, "clone", f.origin, f.clone)

	testutil.Git(t, seed, "switch", "-c", branch)
	testutil.Git(t, seed, "commit", "--allow-empty", "-m", "pr work")
	testutil.Git(t, seed, "push", "origin", branch)

	// GitHub exposes the PR head at refs/pull/<n>/head; mirror that.
	sha := testutil.Git(t, f.origin, "rev-parse", "refs/heads/"+branch)
	testutil.Git(t, f.origin, "update-ref", fmt.Sprintf("refs/pull/%d/head", prNum), sha)
	return f
}

func TestResolvePullRequestFindsHeadBranch(t *testing.T) {
	f := newPullFixture(t, "claude/per-request-identity", 140)
	pr := PullRequest{Host: "github.com", Owner: "fabricahq", Repo: "app", Number: 140, URL: "https://github.com/fabricahq/app/pull/140"}

	matches, err := ResolvePullRequest(pr, []string{f.clone})
	if err != nil {
		t.Fatalf("ResolvePullRequest: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("matches = %+v, want exactly one", matches)
	}
	m := matches[0]
	if m.Branch != "claude/per-request-identity" || m.Remote != "origin" || m.RepoDir != f.clone {
		t.Errorf("match = %+v", m)
	}

	// The branch resolves and fetches just like a -r pickup would.
	path, reused, err := CreateWorktree(m, filepath.Join(".claude", "worktrees", "{{slug}}"))
	if err != nil || reused {
		t.Fatalf("CreateWorktree: path=%s reused=%v err=%v", path, reused, err)
	}
}

func TestResolvePullRequestFailsWhenOnlyDefaultBranchMatchesPRHead(t *testing.T) {
	f := newPullFixture(t, "claude/per-request-identity", 140)
	pr := PullRequest{Host: "github.com", Owner: "fabricahq", Repo: "app", Number: 140, URL: "https://github.com/fabricahq/app/pull/140"}

	sha := testutil.Git(t, f.origin, "rev-parse", "refs/heads/"+f.branch)
	testutil.Git(t, f.origin, "update-ref", "refs/heads/main", sha)
	testutil.Git(t, f.origin, "update-ref", "-d", "refs/heads/"+f.branch)

	_, err := ResolvePullRequest(pr, []string{f.clone})
	var pe *PullError
	if !errors.As(err, &pe) {
		t.Fatalf("err = %v, want *PullError", err)
	}
}

func TestResolvePullRequestFailsWhenPRHeadBranchIsAmbiguous(t *testing.T) {
	f := newPullFixture(t, "claude/per-request-identity", 140)
	pr := PullRequest{Host: "github.com", Owner: "fabricahq", Repo: "app", Number: 140, URL: "https://github.com/fabricahq/app/pull/140"}

	sha := testutil.Git(t, f.origin, "rev-parse", "refs/heads/"+f.branch)
	testutil.Git(t, f.origin, "update-ref", "refs/heads/other-copy", sha)

	_, err := ResolvePullRequest(pr, []string{f.clone})
	var pe *PullError
	if !errors.As(err, &pe) {
		t.Fatalf("err = %v, want *PullError", err)
	}
}

func TestResolvePullRequestNoLocalRepo(t *testing.T) {
	f := newPullFixture(t, "claude/per-request-identity", 140)
	// A PR for a repo no local checkout points at.
	pr := PullRequest{Host: "github.com", Owner: "someoneelse", Repo: "other", Number: 140, URL: "u"}

	_, err := ResolvePullRequest(pr, []string{f.clone})
	var pe *PullError
	if !errors.As(err, &pe) {
		t.Fatalf("err = %v, want *PullError", err)
	}
}

func TestResolvePullRequestUnknownPR(t *testing.T) {
	f := newPullFixture(t, "claude/per-request-identity", 140)
	// The repo matches, but there's no refs/pull/999/head to map.
	pr := PullRequest{Host: "github.com", Owner: "fabricahq", Repo: "app", Number: 999, URL: "u"}

	_, err := ResolvePullRequest(pr, []string{f.clone})
	var pe *PullError
	if !errors.As(err, &pe) {
		t.Fatalf("err = %v, want *PullError", err)
	}
}
