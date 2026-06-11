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

// CreateWorktree fetches the matched branch and checks it out as a new
// worktree at the dirTemplate location ({{branch}} with /→-, {{slug}} =
// last segment) relative to the repo's main worktree. Returns the new path.
func CreateWorktree(m Match, dirTemplate string) (string, error) {
	wts, err := gitx.Worktrees(m.RepoDir)
	if err != nil || len(wts) == 0 {
		return "", fmt.Errorf("cannot determine main worktree of %s: %v", m.RepoDir, err)
	}
	mainRoot := wts[0].Path

	loc := strings.ReplaceAll(dirTemplate, "{{branch}}", strings.ReplaceAll(m.Branch, "/", "-"))
	loc = strings.ReplaceAll(loc, "{{slug}}", lastSegment(m.Branch))
	path := loc
	if !filepath.IsAbs(path) {
		path = filepath.Join(mainRoot, loc)
	}
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("worktree target already exists: %s", path)
	}

	if err := gitx.Fetch(m.RepoDir, m.Remote, m.Branch); err != nil {
		return "", err
	}
	if gitx.BranchExists(m.RepoDir, m.Branch) {
		err = gitx.AddWorktree(m.RepoDir, path, m.Branch)
	} else {
		err = gitx.AddWorktreeTracking(m.RepoDir, path, m.Branch, m.Remote)
	}
	if err != nil {
		return "", err
	}
	return path, nil
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
