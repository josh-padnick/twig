package initwiz

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeRepo creates dir with a .git file; withClaude adds .claude/worktrees.
func fakeRepo(t *testing.T, dir string, withClaude bool) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if withClaude {
		if err := os.MkdirAll(filepath.Join(dir, ".claude", "worktrees"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDiscoverRanksClaudeSignalFirst(t *testing.T) {
	home := t.TempDir()
	// Org dir with a Claude-using repo: the strongest candidate.
	fakeRepo(t, filepath.Join(home, "Code", "fabricahq", "app"), true)
	fakeRepo(t, filepath.Join(home, "Code", "fabricahq", "docs"), false)
	// Org dir with more repos but no Claude signal.
	for _, name := range []string{"a", "b", "c"} {
		fakeRepo(t, filepath.Join(home, "Code", "otherorg", name), false)
	}
	// A base that directly contains repos (flat layout).
	fakeRepo(t, filepath.Join(home, "Projects", "solo"), false)
	// Noise: a dir with no repos.
	if err := os.MkdirAll(filepath.Join(home, "Code", "empty-dir"), 0o755); err != nil {
		t.Fatal(err)
	}

	cands := Discover(home)
	if len(cands) != 3 {
		t.Fatalf("candidates = %+v, want 3", cands)
	}
	if cands[0].Path != filepath.Join(home, "Code", "fabricahq") || cands[0].ClaudeCount != 1 || cands[0].RepoCount != 2 {
		t.Errorf("top candidate = %+v, want fabricahq with claude signal", cands[0])
	}
	if cands[1].Path != filepath.Join(home, "Code", "otherorg") || cands[1].RepoCount != 3 {
		t.Errorf("second = %+v, want otherorg by repo count", cands[1])
	}
	if cands[2].Path != filepath.Join(home, "Projects") {
		t.Errorf("third = %+v, want the flat Projects base", cands[2])
	}
}

func TestDiscoverEmptyHome(t *testing.T) {
	if cands := Discover(t.TempDir()); len(cands) != 0 {
		t.Errorf("candidates = %+v, want none", cands)
	}
}

func TestGenerate(t *testing.T) {
	home := "/Users/u"
	out := Generate(home, Answers{
		Roots:   []string{"/Users/u/Code/fabricahq", "/srv/elsewhere"},
		Editors: []Editor{{Name: "cursor", Command: "cursor {{dir}}"}},
	})
	for _, want := range []string{
		`"~/Code/fabricahq"`,
		`"/srv/elsewhere"`,
		`default = ["ghostty", "cursor"]`,
		"[open.openers.cursor]",
		`command = "cursor {{dir}}"`,
		"# [remote]",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("generated config missing %q:\n%s", want, out)
		}
	}
}

func TestGenerateNoAnswers(t *testing.T) {
	out := Generate("/Users/u", Answers{})
	if !strings.Contains(out, "roots = []") || !strings.Contains(out, `default = ["ghostty"]`) {
		t.Errorf("empty-answers config wrong:\n%s", out)
	}
}

func TestDetectShellRC(t *testing.T) {
	home := t.TempDir()
	t.Setenv("ZDOTDIR", "")
	t.Setenv("SHELL", "/bin/zsh")
	rc, ok := DetectShellRC(home)
	if !ok || rc.Path != filepath.Join(home, ".zshrc") || !strings.Contains(rc.Line, "shell-init zsh") {
		t.Errorf("zsh rc = %+v ok=%v", rc, ok)
	}
	t.Setenv("SHELL", "/usr/local/bin/fish")
	rc, ok = DetectShellRC(home)
	if !ok || rc.Path != filepath.Join(home, ".config", "fish", "config.fish") {
		t.Errorf("fish rc = %+v ok=%v", rc, ok)
	}
	t.Setenv("SHELL", "/bin/tcsh")
	if _, ok := DetectShellRC(home); ok {
		t.Error("unknown shell should not match")
	}
}

func TestInstallIsIdempotent(t *testing.T) {
	home := t.TempDir()
	rc := ShellRC{Shell: "zsh", Path: filepath.Join(home, ".zshrc"), Line: `eval "$(twig shell-init zsh)"`}

	added, err := Install(rc)
	if err != nil || !added {
		t.Fatalf("first install: added=%v err=%v", added, err)
	}
	content, _ := os.ReadFile(rc.Path)
	if !strings.Contains(string(content), "# added by twig init") || !strings.Contains(string(content), rc.Line) {
		t.Errorf("rc content:\n%s", content)
	}

	added, err = Install(rc)
	if err != nil || added {
		t.Fatalf("second install must be a no-op: added=%v err=%v", added, err)
	}
	if c2, _ := os.ReadFile(rc.Path); string(c2) != string(content) {
		t.Error("second install modified the file")
	}
}
