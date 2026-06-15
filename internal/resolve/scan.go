// Filesystem scan: the resolution step for worktrees the current repo does
// not know about. Walks provider locations and configured roots with plain
// directory listings (never filepath.Glob, so fragments can't inject glob
// metacharacters and matching can be case-insensitive), accepting only
// directories that contain a .git entry.
package resolve

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/josh-padnick/twig/internal/gitx"
)

// viaScan matches the stripped fragment against directory basenames under
// every scan parent. An exact basename match short-circuits substring
// matches, mirroring the git-step tier behavior.
func (r *Resolver) viaScan(frag string) (Result, error) {
	s := strings.ToLower(lastSegment(frag))
	byTier := map[Tier][]Candidate{}
	parents := r.scanParents()
	r.trace("scanning %d worktree location(s) under your roots and providers", len(parents))
	for _, parent := range parents {
		r.trace("checking %s", r.homeRel(parent))
		for _, dir := range subdirs(parent) {
			base := strings.ToLower(filepath.Base(dir))
			var tier Tier
			switch {
			case base == s:
				tier = TierExactDir
			case strings.Contains(base, s):
				tier = TierDirSubstr
			default:
				continue
			}
			if !hasGitEntry(dir) {
				continue
			}
			byTier[tier] = append(byTier[tier], Candidate{Path: dir, Source: SourceScan, Tier: tier})
		}
	}
	best, ok := bestTier(byTier)
	if !ok {
		r.trace("no match under any scanned location")
		return Result{}, &NoMatchError{
			Fragment:    frag,
			SearchedGit: gitx.InRepo(r.Cwd),
			Roots:       r.Roots,
			Providers:   providerNames(r.Providers),
		}
	}
	r.trace("matched %d worktree(s) by scan", len(best))
	res, err := finishTier(best)
	if err != nil {
		return Result{}, err
	}
	fillBranches(res)
	return res, nil
}

// All enumerates every worktree twig can see: the current repo's records
// (including stale ones, flagged) plus everything the scan locations
// contain. Used by `twig list`.
func (r *Resolver) All() []Candidate {
	var cands []Candidate
	if gitx.InRepo(r.Cwd) {
		if wts, err := gitx.Worktrees(r.Cwd); err == nil {
			for _, wt := range wts {
				cands = append(cands, Candidate{
					Path: wt.Path, Branch: wt.Branch, Source: SourceGit, Tier: tierNone, Stale: isStale(wt),
				})
			}
		}
	}
	for _, parent := range r.scanParents() {
		for _, dir := range subdirs(parent) {
			if hasGitEntry(dir) {
				cands = append(cands, Candidate{Path: dir, Source: SourceScan, Tier: tierNone})
			}
		}
	}
	return dedupe(cands)
}

// scanParents builds the directories whose immediate children are worktree
// candidates: each root (the <root>/* pattern), each root's subdirectories
// (the <root>/*/* pattern), and every active provider's locations.
func (r *Resolver) scanParents() []string {
	var parents []string
	add := func(p string) {
		if dirExists(p) {
			parents = append(parents, p)
		}
	}
	for _, root := range r.Roots {
		add(root)
		for _, sub := range subdirs(root) {
			add(sub)
		}
	}
	for _, prov := range r.Providers {
		for _, p := range prov.Parents(r.Home, r.Roots) {
			add(p)
		}
	}
	return dedupeStrings(parents)
}

// subdirs lists dir's immediate subdirectories, skipping dot-entries to
// match the zsh glob behavior (and to stay out of .git internals). Symlinks
// that point at directories are followed: Conductor leaves a renamed
// workspace behind as a symlink to its new name.
func subdirs(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		p := filepath.Join(dir, e.Name())
		if e.IsDir() {
			out = append(out, p)
			continue
		}
		if e.Type()&os.ModeSymlink != 0 {
			if fi, err := os.Stat(p); err == nil && fi.IsDir() {
				out = append(out, p)
			}
		}
	}
	return out
}

// hasGitEntry reports whether dir contains a .git file or directory — a
// .git *file* is exactly what a linked worktree has.
func hasGitEntry(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

func dirExists(dir string) bool {
	fi, err := os.Stat(dir)
	return err == nil && fi.IsDir()
}

func dedupeStrings(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// fillBranches best-effort populates Branch on scan candidates so the
// picker and list output can show what's checked out. Only called for the
// final (small) candidate set to keep scans cheap.
func fillBranches(res Result) {
	fill := func(c *Candidate) {
		if c.Branch == "" {
			if b, err := gitx.CurrentBranch(c.Path); err == nil {
				c.Branch = b
			}
		}
	}
	if res.Chosen != nil {
		fill(res.Chosen)
	}
	for i := range res.Candidates {
		fill(&res.Candidates[i])
	}
}
