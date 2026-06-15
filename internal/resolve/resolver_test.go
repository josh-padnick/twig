package resolve

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/josh-padnick/twig/internal/testutil"
)

// newTestResolver builds a resolver rooted at the fixture's layout.
func newTestResolver(f *testutil.Fixture, cwd string) *Resolver {
	return &Resolver{Cwd: cwd, Home: f.Home, Roots: []string{f.CodeRoot}, Providers: Builtin}
}

func TestResolveTraceNarratesScan(t *testing.T) {
	f := testutil.StandardFixture(t)
	r := newTestResolver(f, f.Tmp) // outside any repo: forces the filesystem scan
	var lines []string
	r.Trace = func(msg string) { lines = append(lines, msg) }

	if _, err := r.Resolve("matsumoto"); err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	joined := strings.Join(lines, "\n")
	for _, want := range []string{"resolving", "scanning", "checking ", "matched"} {
		if !strings.Contains(joined, want) {
			t.Errorf("trace missing %q; got:\n%s", want, joined)
		}
	}
	// One of the checked locations must be a real scan parent on disk.
	checkedClaudeWorktrees := false
	for _, l := range lines {
		if strings.HasPrefix(l, "checking ") && strings.Contains(l, filepath.Join(".claude", "worktrees")) {
			checkedClaudeWorktrees = true
		}
	}
	if !checkedClaudeWorktrees {
		t.Errorf("expected a 'checking …/.claude/worktrees' line; got:\n%s", joined)
	}
}

func TestResolveTraceSilentByDefault(t *testing.T) {
	f := testutil.StandardFixture(t)
	// A nil Trace must never be invoked — resolution stays pure by default.
	if _, err := newTestResolver(f, f.App).Resolve("matsumoto"); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
}

func TestResolveAgainstStandardFixture(t *testing.T) {
	f := testutil.StandardFixture(t)

	tests := []struct {
		name     string
		cwd      string
		frag     string
		wantPath string // expected Chosen.Path; "" means expect candidates or error
		wantTier Tier
		wantSrc  Source
	}{
		{"full branch with prefix", f.App, "claude/competent-matsumoto-493452", f.Matsumoto, TierExactBranch, SourceGit},
		{"slug fragment", f.App, "matsumoto", f.Matsumoto, TierBranchSubstr, SourceGit},
		{"hex suffix", f.App, "493452", f.Matsumoto, TierBranchSubstr, SourceGit},
		{"prefix-strip on partial branch", f.App, "claude/vibrant", f.Vibrant, TierBranchSubstr, SourceGit},
		{"exact dir name of sibling", f.App, "app-hotfix", f.Hotfix, TierExactDir, SourceGit},
		{"git knows conductor worktrees", f.App, "berlin", f.Berlin, TierExactDir, SourceGit},
		{"case-insensitive branch match", f.App, "MATSUMOTO", f.Matsumoto, TierBranchSubstr, SourceGit},
		{"outside repo: conductor provider scan", f.Tmp, "berlin", f.Berlin, TierExactDir, SourceScan},
		{"outside repo: claude-code provider scan", f.Tmp, "matsumoto", f.Matsumoto, TierDirSubstr, SourceScan},
		{"outside repo: sibling via root scan", f.Tmp, "app-hotfix", f.Hotfix, TierExactDir, SourceScan},
		{"outside repo: case-insensitive scan", f.Tmp, "Berlin", f.Berlin, TierExactDir, SourceScan},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := newTestResolver(f, tt.cwd).Resolve(tt.frag)
			if err != nil {
				t.Fatalf("Resolve(%q) error: %v", tt.frag, err)
			}
			if res.Chosen == nil {
				t.Fatalf("Resolve(%q): no single choice, got %d candidates", tt.frag, len(res.Candidates))
			}
			if res.Chosen.Path != tt.wantPath {
				t.Errorf("path = %s, want %s", res.Chosen.Path, tt.wantPath)
			}
			if res.Chosen.Tier != tt.wantTier {
				t.Errorf("tier = %d, want %d", res.Chosen.Tier, tt.wantTier)
			}
			if res.Chosen.Source != tt.wantSrc {
				t.Errorf("source = %d, want %d", res.Chosen.Source, tt.wantSrc)
			}
		})
	}
}

func TestResolveLiteralPaths(t *testing.T) {
	f := testutil.StandardFixture(t)
	r := newTestResolver(f, f.Tmp)

	for name, frag := range map[string]string{
		"relative": filepath.Join("code", "fabricahq", "app"),
		"absolute": f.App,
	} {
		t.Run(name, func(t *testing.T) {
			res, err := r.Resolve(frag)
			if err != nil {
				t.Fatal(err)
			}
			if res.Chosen == nil || res.Chosen.Path != f.App || res.Chosen.Source != SourceLiteral {
				t.Errorf("got %+v, want literal %s", res.Chosen, f.App)
			}
		})
	}

	t.Run("tilde", func(t *testing.T) {
		res, err := r.Resolve("~/conductor/workspaces/fabrica/berlin")
		if err != nil {
			t.Fatal(err)
		}
		if res.Chosen == nil || res.Chosen.Path != f.Berlin || res.Chosen.Source != SourceLiteral {
			t.Errorf("got %+v, want literal %s", res.Chosen, f.Berlin)
		}
	})
}

func TestResolveAmbiguousReturnsBestTierOnly(t *testing.T) {
	f := testutil.StandardFixture(t)
	res, err := newTestResolver(f, f.App).Resolve("claude")
	if err != nil {
		t.Fatal(err)
	}
	if res.Chosen != nil {
		t.Fatalf("expected candidates, got single choice %s", res.Chosen.Path)
	}
	if len(res.Candidates) != 2 {
		t.Fatalf("got %d candidates, want 2: %+v", len(res.Candidates), res.Candidates)
	}
	// finishTier sorts by path: competent-matsumoto before vibrant-curran.
	if res.Candidates[0].Path != f.Matsumoto || res.Candidates[1].Path != f.Vibrant {
		t.Errorf("candidates = %s, %s", res.Candidates[0].Path, res.Candidates[1].Path)
	}
}

func TestResolveNoArgListsCurrentRepoWorktrees(t *testing.T) {
	f := testutil.StandardFixture(t)
	res, err := newTestResolver(f, f.App).Resolve("")
	if err != nil {
		t.Fatal(err)
	}
	// app main + matsumoto + vibrant + hotfix + berlin + bordeaux
	if res.Chosen != nil || len(res.Candidates) != 6 {
		t.Fatalf("got chosen=%v candidates=%d, want 6 candidates", res.Chosen, len(res.Candidates))
	}
	if _, err := newTestResolver(f, f.Tmp).Resolve(""); err == nil {
		t.Error("no-arg outside a repo should error")
	}
}

func TestResolveExactBeatsSubstring(t *testing.T) {
	testutil.RequireGit(t)
	f := testutil.StandardFixture(t)
	repo := filepath.Join(f.Tmp, "exact-repo")
	testutil.NewRepo(t, repo)
	testutil.AddWorktree(t, repo, "feat/api", filepath.Join(f.Tmp, "exact-wt-one"))
	testutil.AddWorktree(t, repo, "feat/api-v2", filepath.Join(f.Tmp, "exact-wt-two"))

	res, err := newTestResolver(f, repo).Resolve("api")
	if err != nil {
		t.Fatal(err)
	}
	if res.Chosen == nil {
		t.Fatalf("expected single choice (exact suffix should beat substring), got %d candidates", len(res.Candidates))
	}
	if res.Chosen.Branch != "feat/api" {
		t.Errorf("branch = %s, want feat/api", res.Chosen.Branch)
	}
}

func TestResolveBranchTierBeatsDirTier(t *testing.T) {
	testutil.RequireGit(t)
	f := testutil.StandardFixture(t)
	repo := filepath.Join(f.Tmp, "tier-repo")
	testutil.NewRepo(t, repo)
	testutil.AddWorktree(t, repo, "feat/special", filepath.Join(f.Tmp, "plain-one"))
	testutil.AddWorktree(t, repo, "other/thing", filepath.Join(f.Tmp, "special-dir"))

	res, err := newTestResolver(f, repo).Resolve("special")
	if err != nil {
		t.Fatal(err)
	}
	if res.Chosen == nil || res.Chosen.Branch != "feat/special" {
		t.Errorf("got %+v, want branch feat/special to win over dir basename match", res.Chosen)
	}
}

func TestResolveDetachedWorktreeViaDirName(t *testing.T) {
	testutil.RequireGit(t)
	f := testutil.StandardFixture(t)
	repo := filepath.Join(f.Tmp, "detached-repo")
	testutil.NewRepo(t, repo)
	wt := filepath.Join(f.Tmp, "detached-zone")
	testutil.AddDetachedWorktree(t, repo, wt)

	res, err := newTestResolver(f, repo).Resolve("detached-zone")
	if err != nil {
		t.Fatal(err)
	}
	if res.Chosen == nil || res.Chosen.Path != wt || res.Chosen.Tier != TierExactDir {
		t.Errorf("got %+v, want detached worktree %s via exact dir tier", res.Chosen, wt)
	}
}

func TestResolveStaleRecord(t *testing.T) {
	testutil.RequireGit(t)
	f := testutil.StandardFixture(t)
	repo := filepath.Join(f.Tmp, "stale-repo")
	testutil.NewRepo(t, repo)
	wt := filepath.Join(f.Tmp, "stale-thing")
	testutil.AddWorktree(t, repo, "tmp/stale-thing", wt)
	if err := os.RemoveAll(wt); err != nil {
		t.Fatal(err)
	}

	_, err := newTestResolver(f, repo).Resolve("stale-thing")
	var staleErr *StaleError
	if !errors.As(err, &staleErr) {
		t.Fatalf("got %v, want StaleError", err)
	}
	if len(staleErr.Paths) != 1 || staleErr.Paths[0] != wt {
		t.Errorf("stale paths = %v, want [%s]", staleErr.Paths, wt)
	}
}

func TestResolveNoMatch(t *testing.T) {
	f := testutil.StandardFixture(t)
	_, err := newTestResolver(f, f.Tmp).Resolve("zzz-nothing-here")
	var noMatch *NoMatchError
	if !errors.As(err, &noMatch) {
		t.Fatalf("got %v, want NoMatchError", err)
	}
	if noMatch.Fragment != "zzz-nothing-here" {
		t.Errorf("fragment = %s", noMatch.Fragment)
	}
}

func TestResolveDuplicateRootsDedupe(t *testing.T) {
	f := testutil.StandardFixture(t)
	r := &Resolver{Cwd: f.Tmp, Home: f.Home, Roots: []string{f.CodeRoot, f.CodeRoot}, Providers: Builtin}
	res, err := r.Resolve("app-hotfix")
	if err != nil {
		t.Fatal(err)
	}
	if res.Chosen == nil {
		t.Fatalf("duplicate roots produced %d candidates, want deduped single choice", len(res.Candidates))
	}
}

func TestResolveProviderTrimming(t *testing.T) {
	f := testutil.StandardFixture(t)
	// Only the conductor provider, no roots: the Claude Code layout becomes invisible.
	r := &Resolver{Cwd: f.Tmp, Home: f.Home, Providers: ByNames([]string{"conductor"})}
	if _, err := r.Resolve("matsumoto"); err == nil {
		t.Error("expected no match with claude-code provider disabled and no roots")
	}
	res, err := r.Resolve("berlin")
	if err != nil || res.Chosen == nil || res.Chosen.Path != f.Berlin {
		t.Errorf("conductor provider alone should still find berlin: res=%+v err=%v", res, err)
	}
}
