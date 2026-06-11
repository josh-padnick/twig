// Package gitx is twig's thin wrapper around the system git binary. Every
// function shells out with an explicit working directory and returns trimmed
// output or an error carrying git's stderr, so callers never parse exit
// codes or stream noise.
package gitx

import (
	"bytes"
	"fmt"
	"os/exec"
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

// IsDirty reports whether the worktree at dir has uncommitted changes
// (staged, unstaged, or untracked).
func IsDirty(dir string) (bool, error) {
	out, err := run(dir, "status", "--porcelain", "--no-renames")
	if err != nil {
		return false, err
	}
	return out != "", nil
}
