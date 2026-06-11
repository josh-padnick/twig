// Package resolve turns a user-typed fragment — a branch name, a branch
// suffix like "claude/foo" → "foo", a directory slug, or a literal path —
// into git worktree directories. The resolver is pure: it never prompts,
// returning either one chosen candidate or the best-tier candidate set for
// the caller's interactive picker.
package resolve

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/josh-padnick/twig/internal/gitx"
)

// Source records which resolution step produced a candidate.
type Source int

const (
	SourceLiteral Source = iota // the fragment was an existing directory path
	SourceGit                   // matched via `git worktree list` of the current repo
	SourceScan                  // matched by scanning provider locations and roots
	SourceRemote                // worktree created by remote-branch pickup
)

// Tier ranks how a candidate matched; lower is stronger. Exact tiers
// short-circuit substring tiers so `api` resolves to branch `feat/api`
// without a picker even when `feat/api-v2` also exists.
type Tier int

const (
	TierExactBranch  Tier = iota // branch equals the fragment
	TierExactDir                 // dir basename equals the stripped fragment
	TierBranchSuffix             // branch's last segment equals the stripped fragment
	TierBranchSubstr             // fragment is a substring of the branch
	TierDirSubstr                // stripped fragment is a substring of the dir basename
	tierNone                     // no match
)

// Candidate is one resolved worktree directory.
type Candidate struct {
	Path   string // absolute path
	Branch string // short branch name, "" when detached or unknown
	Source Source
	Tier   Tier
	Stale  bool // git record exists but the directory is gone (All only)
}

// Result is the outcome of a resolution: exactly one of Chosen or
// Candidates is populated. Candidates always contains the single best tier,
// sorted by path.
type Result struct {
	Chosen     *Candidate
	Candidates []Candidate
}

// Resolver resolves fragments relative to a working directory, a set of
// scan roots, and the active providers.
type Resolver struct {
	Cwd       string
	Home      string // used for ~ expansion and provider fixed locations
	Roots     []string
	Providers []Provider
}

// NoMatchError reports that nothing matched, naming what was searched so
// the user can tell whether a root or provider is missing from config.
type NoMatchError struct {
	Fragment    string
	SearchedGit bool
	Roots       []string
	Providers   []string
}

func (e *NoMatchError) Error() string {
	var searched []string
	if e.SearchedGit {
		searched = append(searched, "the current repo's worktrees")
	}
	if len(e.Providers) > 0 {
		searched = append(searched, "providers "+strings.Join(e.Providers, ", "))
	}
	switch len(e.Roots) {
	case 0:
	case 1:
		searched = append(searched, "root "+e.Roots[0])
	default:
		searched = append(searched, fmt.Sprintf("%d roots", len(e.Roots)))
	}
	if len(searched) == 0 {
		searched = append(searched, "nothing — no repo, providers, or roots")
	}
	return fmt.Sprintf("no worktree matching %q (searched %s)", e.Fragment, strings.Join(searched, "; "))
}

// StaleError reports that the fragment only matched worktree records whose
// directories are gone.
type StaleError struct {
	Fragment string
	Paths    []string
	RepoDir  string
}

func (e *StaleError) Error() string {
	return fmt.Sprintf("%q matches a worktree record at %s but the directory is gone — run 'git worktree prune' in %s",
		e.Fragment, strings.Join(e.Paths, ", "), e.RepoDir)
}

// Resolve resolves frag. An empty fragment means "the current repo's
// worktrees" (interactive-pick mode). Order: literal path, then the current
// repo's worktree list, then the filesystem scan.
func (r *Resolver) Resolve(frag string) (Result, error) {
	if frag == "" {
		return r.resolveNoArg()
	}
	if c, ok := r.literal(frag); ok {
		return Result{Chosen: &c}, nil
	}
	res, matched, err := r.viaGit(frag)
	if err != nil {
		return Result{}, err
	}
	if matched {
		return res, nil
	}
	return r.viaScan(frag)
}

// resolveNoArg implements the no-argument mode: every live worktree of the
// current repo becomes a candidate for the picker.
func (r *Resolver) resolveNoArg() (Result, error) {
	if !gitx.InRepo(r.Cwd) {
		return Result{}, errors.New("not inside a git repository (usage: twig <fragment>)")
	}
	wts, err := gitx.Worktrees(r.Cwd)
	if err != nil {
		return Result{}, err
	}
	var cands []Candidate
	for _, wt := range wts {
		if isStale(wt) {
			continue
		}
		cands = append(cands, Candidate{Path: wt.Path, Branch: wt.Branch, Source: SourceGit, Tier: tierNone})
	}
	return finishTier(cands)
}

// literal treats the fragment as a path (with ~ expansion, relative to Cwd)
// and accepts any existing directory — no .git requirement, matching the
// original zsh behavior.
func (r *Resolver) literal(frag string) (Candidate, bool) {
	p := frag
	if p == "~" || strings.HasPrefix(p, "~/") {
		p = filepath.Join(r.Home, strings.TrimPrefix(p[1:], "/"))
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(r.Cwd, p)
	}
	if fi, err := os.Stat(p); err == nil && fi.IsDir() {
		return Candidate{Path: filepath.Clean(p), Source: SourceLiteral, Tier: tierNone}, true
	}
	return Candidate{}, false
}

// viaGit matches the fragment against the current repo's worktree list.
// matched reports whether any live worktree matched; a fragment that only
// matches stale records returns a StaleError so the user learns the
// directory is gone instead of falling through to an unrelated scan match.
func (r *Resolver) viaGit(frag string) (res Result, matched bool, err error) {
	if !gitx.InRepo(r.Cwd) {
		return Result{}, false, nil
	}
	wts, gerr := gitx.Worktrees(r.Cwd)
	if gerr != nil {
		return Result{}, false, gerr
	}
	byTier := map[Tier][]Candidate{}
	var staleMatches []string
	for _, wt := range wts {
		tier := gitTier(frag, wt)
		if tier == tierNone {
			continue
		}
		if isStale(wt) {
			staleMatches = append(staleMatches, wt.Path)
			continue
		}
		byTier[tier] = append(byTier[tier], Candidate{Path: wt.Path, Branch: wt.Branch, Source: SourceGit, Tier: tier})
	}
	if best, ok := bestTier(byTier); ok {
		res, err := finishTier(best)
		return res, true, err
	}
	if len(staleMatches) > 0 {
		repoDir := r.Cwd
		if len(wts) > 0 {
			repoDir = wts[0].Path // first porcelain entry is the main worktree
		}
		sort.Strings(staleMatches)
		return Result{}, false, &StaleError{Fragment: frag, Paths: staleMatches, RepoDir: repoDir}
	}
	return Result{}, false, nil
}

// BranchTier scores a branch name against a fragment using the branch
// tiers only — shared by the git resolution step and remote-branch pickup.
// All comparisons are case-insensitive.
func BranchTier(frag, branch string) (Tier, bool) {
	if branch == "" {
		return tierNone, false
	}
	f := strings.ToLower(frag)
	s := lastSegment(f)
	b := strings.ToLower(branch)
	switch {
	case b == f:
		return TierExactBranch, true
	case lastSegment(b) == s:
		return TierBranchSuffix, true
	case strings.Contains(b, s) || strings.Contains(b, f):
		return TierBranchSubstr, true
	}
	return tierNone, false
}

// gitTier scores one worktree against the fragment. Branch tiers outrank
// directory tiers so git's own branch names stay authoritative, except an
// exact directory-name match beats inexact branch matches.
func gitTier(frag string, wt gitx.Worktree) Tier {
	s := lastSegment(strings.ToLower(frag))
	base := strings.ToLower(filepath.Base(wt.Path))
	branchTier, branchOK := BranchTier(frag, wt.Branch)
	switch {
	case branchOK && branchTier == TierExactBranch:
		return TierExactBranch
	case base == s:
		return TierExactDir
	case branchOK:
		return branchTier
	case strings.Contains(base, s):
		return TierDirSubstr
	}
	return tierNone
}

// isStale reports whether a worktree record's directory is gone, combining
// git's prunable flag (2.36+) with a direct stat for older versions.
func isStale(wt gitx.Worktree) bool {
	if wt.Prunable {
		return true
	}
	fi, err := os.Stat(wt.Path)
	return err != nil || !fi.IsDir()
}

// bestTier returns the candidates of the strongest populated tier.
func bestTier(byTier map[Tier][]Candidate) ([]Candidate, bool) {
	for tier := TierExactBranch; tier < tierNone; tier++ {
		if cands := byTier[tier]; len(cands) > 0 {
			return cands, true
		}
	}
	return nil, false
}

// finishTier dedupes and sorts one tier's candidates and packages them as a
// Result, auto-choosing when exactly one remains.
func finishTier(cands []Candidate) (Result, error) {
	cands = dedupe(cands)
	sort.Slice(cands, func(i, j int) bool { return cands[i].Path < cands[j].Path })
	if len(cands) == 1 {
		return Result{Chosen: &cands[0]}, nil
	}
	return Result{Candidates: cands}, nil
}

// dedupe collapses candidates that point at the same directory through
// different scan patterns or symlinks, preferring the canonical
// (non-symlink) path when both spellings appear.
func dedupe(cands []Candidate) []Candidate {
	seen := map[string]int{} // realpath -> index in out
	var out []Candidate
	for _, c := range cands {
		key := c.Path
		if resolved, err := filepath.EvalSymlinks(c.Path); err == nil {
			key = resolved
		}
		if i, ok := seen[key]; ok {
			if out[i].Path != key && c.Path == key {
				out[i] = c
			}
			continue
		}
		seen[key] = len(out)
		out = append(out, c)
	}
	return out
}

// lastSegment returns the substring after the final "/", porting the zsh
// ${target##*/} prefix-strip so full branch names like claude/foo match.
func lastSegment(s string) string {
	if i := strings.LastIndex(s, "/"); i >= 0 {
		return s[i+1:]
	}
	return s
}
