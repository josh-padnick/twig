// Package gitx is twig's thin wrapper around the system git binary. Every
// function shells out with an explicit working directory and returns trimmed
// output or an error carrying git's stderr, so callers never parse exit
// codes or stream noise.
package gitx

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// run executes git with args in dir, returning trimmed stdout. On failure
// the error message carries git's stderr so callers can surface it directly.
func run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimRight(stdout.String(), "\n"), nil
}

// InRepo reports whether dir is inside a git working tree.
func InRepo(dir string) bool {
	out, err := run(dir, "rev-parse", "--is-inside-work-tree")
	return err == nil && out == "true"
}

// Worktrees lists every worktree git knows about for the repo containing
// dir, regardless of where those worktrees live on disk.
func Worktrees(dir string) ([]Worktree, error) {
	out, err := run(dir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return ParseWorktrees(out), nil
}

// RepoRoot returns the top-level directory of the worktree containing dir.
func RepoRoot(dir string) (string, error) {
	return run(dir, "rev-parse", "--show-toplevel")
}

// GitDir returns the absolute git directory for the worktree containing dir.
// For a linked worktree this is <main>/.git/worktrees/<name>, which is where
// twig keeps per-worktree state.
func GitDir(dir string) (string, error) {
	return run(dir, "rev-parse", "--absolute-git-dir")
}

// CurrentBranch returns the short branch name checked out in dir, or "" for
// a detached HEAD.
func CurrentBranch(dir string) (string, error) {
	out, err := run(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	if out == "HEAD" {
		return "", nil
	}
	return out, nil
}

// Remotes lists the repo's remote names.
func Remotes(repoDir string) ([]string, error) {
	out, err := run(repoDir, "remote")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}

// LsRemoteHeads lists short branch names on the named remote. This is a
// network call (or a filesystem read for path remotes).
func LsRemoteHeads(repoDir, remote string) ([]string, error) {
	out, err := run(repoDir, "ls-remote", "--heads", remote)
	if err != nil {
		return nil, err
	}
	var branches []string
	for _, line := range strings.Split(out, "\n") {
		// Lines look like "<sha>\trefs/heads/<branch>".
		if _, ref, ok := strings.Cut(line, "\t"); ok {
			if branch, ok := strings.CutPrefix(ref, "refs/heads/"); ok {
				branches = append(branches, branch)
			}
		}
	}
	return branches, nil
}

// Fetch fetches one branch from the named remote.
func Fetch(repoDir, remote, branch string) error {
	_, err := run(repoDir, "fetch", remote, branch)
	return err
}

// BranchExists reports whether a local branch exists.
func BranchExists(repoDir, branch string) bool {
	_, err := run(repoDir, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch)
	return err == nil
}

// AddWorktree checks out an existing local branch as a worktree at path.
func AddWorktree(repoDir, path, branch string) error {
	_, err := run(repoDir, "worktree", "add", path, branch)
	return err
}

// AddWorktreeTracking creates local branch from remote/branch and checks it
// out as a worktree at path, with upstream tracking set.
func AddWorktreeTracking(repoDir, path, branch, remote string) error {
	_, err := run(repoDir, "worktree", "add", "--track", "-b", branch, path, remote+"/"+branch)
	return err
}

// RemoveWorktree removes the worktree at path, invoked from repoDir (any
// worktree of the same repo). force removes even with uncommitted changes.
func RemoveWorktree(repoDir, path string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)
	_, err := run(repoDir, args...)
	return err
}

// Prune drops worktree records whose directories are gone.
func Prune(repoDir string) error {
	_, err := run(repoDir, "worktree", "prune")
	return err
}

// Version returns the git version string, for doctor.
func Version() (string, error) {
	return run("", "version")
}

// AheadOfUpstream returns how many commits the worktree's branch has that
// its upstream doesn't. ok is false when there is no upstream to compare
// against (no remote, no tracking branch).
func AheadOfUpstream(dir string) (count int, ok bool) {
	out, err := run(dir, "rev-list", "--count", "@{upstream}..HEAD")
	if err != nil {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		return 0, false
	}
	return n, true
}

// IsDirty reports whether the worktree at dir has uncommitted changes
// (staged, unstaged, or untracked).
func IsDirty(dir string) (bool, error) {
	out, err := run(dir, "status", "--porcelain", "--no-renames")
	if err != nil {
		return false, err
	}
	return out != "", nil
}
