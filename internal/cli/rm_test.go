package cli

import (
	"bytes"
	"os"
	"testing"

	"github.com/josh-padnick/twig/internal/gitx"
	"github.com/josh-padnick/twig/internal/testutil"
)

func TestRmForceRemovesWorktreeAndPrunes(t *testing.T) {
	f := testutil.StandardFixture(t)
	isolate(t, f)
	t.Chdir(f.App)

	root := newRootCmd("test", "test")
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"rm", "--force", "app-hotfix"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(f.Hotfix); !os.IsNotExist(err) {
		t.Errorf("worktree dir still exists: %v", err)
	}
	wts, err := gitx.Worktrees(f.App)
	if err != nil {
		t.Fatal(err)
	}
	for _, wt := range wts {
		if wt.Path == f.Hotfix {
			t.Error("worktree record not pruned")
		}
	}
}

func TestRmRefusesMainWorktree(t *testing.T) {
	f := testutil.StandardFixture(t)
	isolate(t, f)
	t.Chdir(f.App)

	root := newRootCmd("test", "test")
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"rm", "--force", f.App})
	if err := root.Execute(); err == nil {
		t.Fatal("removing the main worktree must fail")
	}
	if _, err := os.Stat(f.App); err != nil {
		t.Errorf("main worktree harmed: %v", err)
	}
}
