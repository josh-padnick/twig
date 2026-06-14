// Package remote implements remote-branch pickup: cloud sessions (Claude
// Code web, Codex) leave their work as branches on GitHub that no local
// checkout knows about yet. When local resolution finds nothing, twig can
// ask the remotes of repos already on disk via `git ls-remote` — host-
// agnostic, no API tokens — then fetch the branch and create a worktree
// for it. Cloning a repo that isn't on disk at all is explicitly out of
// scope for v0.1.
package remote

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/josh-padnick/twig/internal/gitx"
	"github.com/josh-padnick/twig/internal/resolve"
)

// Match is one remote branch that matched the fragment.
type Match struct {
	RepoDir string // local repo whose remote has the branch
	Remote  string // remote name, e.g. origin
	Branch  string // short branch name
	Tier    resolve.Tier
}

// DisplayMatch renders a match for the picker and confirmation prompts.
func DisplayMatch(m Match) string {
	return fmt.Sprintf("%s  [%s of %s]", m.Branch, m.Remote, m.RepoDir)
}

// CandidateRepos enumerates the local repos whose remotes are worth
// querying: the current repo first, then the roots themselves and repos
// one level under each root. Deeper worktrees share those repos' remotes,
// so querying them too would only repeat network calls.
func CandidateRepos(cwd string, roots []string) []string {
	var repos []string
	if gitx.InRepo(cwd) {
		if r, err := gitx.RepoRoot(cwd); err == nil {
			repos = append(repos, r)
		}
	}
	for _, root := range roots {
		if hasGit(root) {
			repos = append(repos, root)
		}
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			d := filepath.Join(root, e.Name())
			if hasGit(d) {
				repos = append(repos, d)
			}
		}
	}
	return dedupe(repos)
}

// Search queries every repo's remotes in parallel and returns the
// strongest-tier matches, current repo (first element of repos) preferred
// within a tier.
func Search(frag string, repos []string) []Match {
	perRepo := make([][]Match, len(repos))
	var wg sync.WaitGroup
	for i, repo := range repos {
		wg.Add(1)
		go func(i int, repo string) {
			defer wg.Done()
			remotes, err := gitx.Remotes(repo)
			if err != nil {
				return
			}
			for _, rem := range remotes {
				branches, err := gitx.LsRemoteHeads(repo, rem)
				if err != nil {
					continue
				}
				for _, branch := range branches {
					if tier, ok := resolve.BranchTier(frag, branch); ok {
						perRepo[i] = append(perRepo[i], Match{RepoDir: repo, Remote: rem, Branch: branch, Tier: tier})
					}
				}
			}
		}(i, repo)
	}
	wg.Wait()

	var all []Match
	for _, ms := range perRepo { // repo order preserved: current repo first
		all = append(all, ms...)
	}
	if len(all) == 0 {
		return nil
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].Tier < all[j].Tier })
	best := all[0].Tier
	cut := len(all)
	for i, m := range all {
		if m.Tier > best {
			cut = i
			break
		}
	}
	return all[:cut]
}

// CreateWorktree makes the matched branch reachable as a local worktree and
// returns its path. If the branch is already checked out in a worktree of
// the repo it returns that existing path with reused=true (git allows only
// one worktree per branch, so this is both the correct answer and what the
// user wants — land in the work that already exists). Otherwise it fetches
// the branch and checks it out at the dirTemplate location ({{branch}} with
// /→-, {{slug}} = last segment) relative to the repo's main worktree.
func CreateWorktree(m Match, dirTemplate string) (path string, reused bool, err error) {
	// The branch may already be checked out — common when a cloud session's
	// branch was picked up earlier, possibly from a different directory.
	// `git worktree add` would refuse a second worktree for it; reuse the
	// existing one instead of failing.
	if existing, ok := ExistingCheckout(m); ok {
		return existing, true, nil
	}

	wts, err := gitx.Worktrees(m.RepoDir)
	if err != nil || len(wts) == 0 {
		return "", false, fmt.Errorf("cannot determine main worktree of %s: %v", m.RepoDir, err)
	}
	mainRoot := wts[0].Path

	loc := strings.ReplaceAll(dirTemplate, "{{branch}}", strings.ReplaceAll(m.Branch, "/", "-"))
	loc = strings.ReplaceAll(loc, "{{slug}}", lastSegment(m.Branch))
	path = loc
	if !filepath.IsAbs(path) {
		path = filepath.Join(mainRoot, loc)
	}
	if _, err := os.Stat(path); err == nil {
		return "", false, fmt.Errorf("worktree target already exists: %s", path)
	}

	if err := gitx.Fetch(m.RepoDir, m.Remote, m.Branch); err != nil {
		return "", false, err
	}
	if gitx.BranchExists(m.RepoDir, m.Branch) {
		err = gitx.AddWorktree(m.RepoDir, path, m.Branch)
	} else {
		err = gitx.AddWorktreeTracking(m.RepoDir, path, m.Branch, m.Remote)
	}
	if err != nil {
		return "", false, err
	}
	return path, false, nil
}

// ExistingCheckout returns the path of a worktree in the match's repo that
// already has the branch checked out, if any. git permits only one worktree
// per branch, so when this hits there is nothing to fetch or create — the
// caller can enter the returned path directly, no confirmation needed.
func ExistingCheckout(m Match) (string, bool) {
	wts, err := gitx.Worktrees(m.RepoDir)
	if err != nil {
		return "", false
	}
	for _, wt := range wts {
		if wt.Branch != "" && wt.Branch == m.Branch {
			return wt.Path, true
		}
	}
	return "", false
}

func hasGit(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		key := s
		if resolved, err := filepath.EvalSymlinks(s); err == nil {
			key = resolved
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, s)
	}
	return out
}

func lastSegment(s string) string {
	if i := strings.LastIndex(s, "/"); i >= 0 {
		return s[i+1:]
	}
	return s
}
