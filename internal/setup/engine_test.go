package setup

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/josh-padnick/twig/internal/testutil"
	"github.com/josh-padnick/twig/internal/trust"
)

// engineFixture is one repo + one linked worktree with a main-root manifest,
// a pre-wired trust store, and helpers to count setup executions (the setup
// script appends a line to a log file in the worktree).
type engineFixture struct {
	repo, worktree, manifest, log string
	store                         *trust.Store
	engine                        *Engine
}

func newEngineFixture(t *testing.T, manifestContent string) *engineFixture {
	t.Helper()
	testutil.RequireGit(t)
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f := &engineFixture{
		repo:     filepath.Join(tmp, "repo"),
		worktree: filepath.Join(tmp, "wt"),
	}
	testutil.NewRepo(t, f.repo)
	testutil.AddWorktree(t, f.repo, "feat/thing", f.worktree)
	f.manifest = filepath.Join(f.repo, "twig.toml")
	f.log = filepath.Join(f.worktree, "setup-log")
	f.writeManifest(t, manifestContent)
	f.store = &trust.Store{Dir: filepath.Join(tmp, "trust")}
	f.engine = &Engine{Trust: f.store, Version: "test"}
	return f
}

func (f *engineFixture) writeManifest(t *testing.T, content string) {
	t.Helper()
	if err := os.WriteFile(f.manifest, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func (f *engineFixture) approve(t *testing.T) {
	t.Helper()
	content, err := os.ReadFile(f.manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.store.Approve(f.manifest, content); err != nil {
		t.Fatal(err)
	}
}

// setupRuns counts how many times the setup script has executed.
func (f *engineFixture) setupRuns(t *testing.T) int {
	t.Helper()
	data, err := os.ReadFile(f.log)
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Count(string(data), "ran")
}

const basicManifest = "[setup]\nrun = \"echo ran >> setup-log\"\n"

func TestEnterRunsSetupOnceAndSkipsWhenUnchanged(t *testing.T) {
	f := newEngineFixture(t, basicManifest)
	f.approve(t)

	if err := f.engine.Enter(EnterOptions{Dir: f.worktree}); err != nil {
		t.Fatal(err)
	}
	if got := f.setupRuns(t); got != 1 {
		t.Fatalf("setup runs = %d, want 1", got)
	}
	if err := f.engine.Enter(EnterOptions{Dir: f.worktree}); err != nil {
		t.Fatal(err)
	}
	if got := f.setupRuns(t); got != 1 {
		t.Errorf("setup runs after re-entry = %d, want 1 (skip)", got)
	}
	if err := f.engine.Enter(EnterOptions{Dir: f.worktree, ForceSetup: true}); err != nil {
		t.Fatal(err)
	}
	if got := f.setupRuns(t); got != 2 {
		t.Errorf("setup runs after --setup = %d, want 2", got)
	}
}

func TestEnterRerunsWhenWatchFileChanges(t *testing.T) {
	f := newEngineFixture(t, "[setup]\nrun = \"echo ran >> setup-log\"\nwatch = [\"dep.txt\"]\n")
	f.approve(t)

	if err := f.engine.Enter(EnterOptions{Dir: f.worktree}); err != nil {
		t.Fatal(err)
	}
	// Creating a previously-absent watch file is a change.
	if err := os.WriteFile(filepath.Join(f.worktree, "dep.txt"), []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := f.engine.Enter(EnterOptions{Dir: f.worktree}); err != nil {
		t.Fatal(err)
	}
	if got := f.setupRuns(t); got != 2 {
		t.Fatalf("setup runs after watch file appeared = %d, want 2", got)
	}
	// Unchanged content: skip.
	if err := f.engine.Enter(EnterOptions{Dir: f.worktree}); err != nil {
		t.Fatal(err)
	}
	if got := f.setupRuns(t); got != 2 {
		t.Errorf("setup runs with unchanged watch file = %d, want 2", got)
	}
	// Edited content: re-run.
	if err := os.WriteFile(filepath.Join(f.worktree, "dep.txt"), []byte("v2"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := f.engine.Enter(EnterOptions{Dir: f.worktree}); err != nil {
		t.Fatal(err)
	}
	if got := f.setupRuns(t); got != 3 {
		t.Errorf("setup runs after watch file edit = %d, want 3", got)
	}
}

func TestEnterManifestEditRetripsTrustThenReruns(t *testing.T) {
	f := newEngineFixture(t, basicManifest)
	f.approve(t)
	if err := f.engine.Enter(EnterOptions{Dir: f.worktree}); err != nil {
		t.Fatal(err)
	}

	f.writeManifest(t, basicManifest+"# comment that changes the hash\n")
	err := f.engine.Enter(EnterOptions{Dir: f.worktree})
	if err == nil || !strings.Contains(err.Error(), "twig trust") {
		t.Fatalf("edited manifest: err = %v, want trust instruction", err)
	}
	if got := f.setupRuns(t); got != 1 {
		t.Fatalf("untrusted edit must not run setup; runs = %d", got)
	}

	f.approve(t)
	if err := f.engine.Enter(EnterOptions{Dir: f.worktree}); err != nil {
		t.Fatal(err)
	}
	if got := f.setupRuns(t); got != 2 {
		t.Errorf("setup runs after re-approval = %d, want 2 (manifest hash changed)", got)
	}
}

func TestEnterUntrustedWithInteractiveApproval(t *testing.T) {
	f := newEngineFixture(t, basicManifest)
	f.engine.Confirm = func(string) (bool, error) { return true, nil }

	if err := f.engine.Enter(EnterOptions{Dir: f.worktree}); err != nil {
		t.Fatal(err)
	}
	if got := f.setupRuns(t); got != 1 {
		t.Errorf("setup runs after interactive approval = %d, want 1", got)
	}
	content, _ := os.ReadFile(f.manifest)
	if ok, _ := f.store.Check(f.manifest, content); !ok {
		t.Error("interactive approval should persist in the store")
	}
}

func TestEnterSetupFailureAbortsAndKeepsNoState(t *testing.T) {
	f := newEngineFixture(t, "[setup]\nrun = \"echo ran >> setup-log\\nexit 3\"\n\n[run]\nrun = \"echo run-ran >> setup-log\"\n")
	f.approve(t)

	err := f.engine.Enter(EnterOptions{Dir: f.worktree, RunAfter: true})
	if err == nil || !strings.Contains(err.Error(), "setup failed") {
		t.Fatalf("err = %v, want setup failure", err)
	}
	data, _ := os.ReadFile(f.log)
	if strings.Contains(string(data), "run-ran") {
		t.Error("[run] must not execute after setup failure")
	}
	// No success marker: next entry runs setup again.
	if readState(f.worktree) != nil {
		t.Error("failed setup must not record state")
	}
}

func TestEnterRunScript(t *testing.T) {
	f := newEngineFixture(t, basicManifest+"\n[run]\nrun = \"echo run-ran >> setup-log\"\n")
	f.approve(t)

	if err := f.engine.Enter(EnterOptions{Dir: f.worktree, RunAfter: true}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(f.log)
	if !strings.Contains(string(data), "run-ran") {
		t.Errorf("log = %q, want run-ran", data)
	}
}

func TestEnterRunExitCodePropagates(t *testing.T) {
	f := newEngineFixture(t, "[run]\nrun = \"exit 7\"\n")
	f.approve(t)

	err := f.engine.Enter(EnterOptions{Dir: f.worktree, RunAfter: true})
	var ec *ExitCodeError
	if !errors.As(err, &ec) || ec.Code != 7 {
		t.Fatalf("err = %v, want ExitCodeError{7}", err)
	}
}

func TestEnterNoManifestIsANoOp(t *testing.T) {
	testutil.RequireGit(t)
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(tmp, "repo")
	testutil.NewRepo(t, repo)
	engine := &Engine{Trust: &trust.Store{Dir: filepath.Join(tmp, "trust")}, Version: "test"}

	if err := engine.Enter(EnterOptions{Dir: repo}); err != nil {
		t.Errorf("no manifest should be a silent no-op, got %v", err)
	}
	if err := engine.Enter(EnterOptions{Dir: repo, RunAfter: true}); !errors.Is(err, ErrNoManifest) {
		t.Errorf("--run with no manifest: err = %v, want ErrNoManifest", err)
	}
}

func TestEnterSetupEnvironment(t *testing.T) {
	f := newEngineFixture(t, "[setup]\nrun = \"echo $TWIG_BRANCH:$TWIG_WORKTREE:$TWIG_REPO_ROOT > env-log\"\n")
	f.approve(t)
	if err := f.engine.Enter(EnterOptions{Dir: f.worktree}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(f.worktree, "env-log"))
	if err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(string(data))
	want := "feat/thing:" + f.worktree + ":" + f.repo
	if got != want {
		t.Errorf("env = %q, want %q", got, want)
	}
}
