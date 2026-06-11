package opener

import (
	"strings"
	"testing"

	"github.com/josh-padnick/twig/internal/config"
)

func TestQuote(t *testing.T) {
	tests := map[string]string{
		"":                  "''",
		"plain":             "'plain'",
		"with space":        "'with space'",
		"it's quoted":       `'it'\''s quoted'`,
		"$HOME `cmd` \"x\"": "'$HOME `cmd` \"x\"'",
	}
	for in, want := range tests {
		if got := Quote(in); got != want {
			t.Errorf("Quote(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestEntryLine(t *testing.T) {
	target := Target{Dir: "/code/my worktree", EnterCmd: "'/usr/local/bin/twig' enter --run"}
	got := EntryLine(target, true)
	want := "cd '/code/my worktree' && clear && '/usr/local/bin/twig' enter --run"
	if got != want {
		t.Errorf("EntryLine = %q, want %q", got, want)
	}
	if got := EntryLine(Target{Dir: "/code/app"}, false); got != "cd '/code/app'" {
		t.Errorf("EntryLine bare = %q", got)
	}
}

func TestCommandOpenerTemplating(t *testing.T) {
	op, err := newCommand("wezterm", config.Opener{Kind: "command", Command: "wezterm cli spawn --cwd {{dir}} -- bash -c {{cmd}}"})
	if err != nil {
		t.Fatal(err)
	}
	if !op.CanInject() {
		t.Error("template with {{cmd}} should be injection-capable")
	}
	line := op.(*command).buildLine(Target{Dir: "/code/a b", EnterCmd: "twig enter"})
	want := "wezterm cli spawn --cwd '/code/a b' -- bash -c 'cd '\\''/code/a b'\\'' && twig enter'"
	if line != want {
		t.Errorf("buildLine =\n  %s\nwant\n  %s", line, want)
	}

	plain, err := newCommand("cursor", config.Opener{Kind: "command", Command: "cursor {{dir}}"})
	if err != nil {
		t.Fatal(err)
	}
	if plain.CanInject() {
		t.Error("template without {{cmd}} must not claim injection")
	}
	if got := plain.(*command).buildLine(Target{Dir: "/code/app"}); got != "cursor '/code/app'" {
		t.Errorf("buildLine = %q", got)
	}
}

func TestCommandOpenerRejectsCurrentTab(t *testing.T) {
	op, err := newCommand("cursor", config.Opener{Kind: "command", Command: "cursor {{dir}}"})
	if err != nil {
		t.Fatal(err)
	}
	if err := op.Open(Target{Dir: "/code/app", Mode: ModeCurrentTab}); err == nil || !strings.Contains(err.Error(), "current tab") {
		t.Errorf("err = %v, want current-tab rejection", err)
	}
}

func TestFromConfig(t *testing.T) {
	catalog := config.Open{Openers: map[string]config.Opener{
		"cursor": {Kind: "command", Command: "cursor {{dir}}"},
		"broken": {Kind: "teleport"},
		"empty":  {Kind: "command"},
	}}

	if op, err := FromConfig("ghostty", catalog); err != nil || op.Name() != "ghostty" {
		t.Errorf("builtin ghostty: op=%v err=%v", op, err)
	}
	if op, err := FromConfig("cursor", catalog); err != nil || op.Name() != "cursor" {
		t.Errorf("cursor: op=%v err=%v", op, err)
	}
	if _, err := FromConfig("phantom", catalog); err == nil {
		t.Error("undefined opener should error")
	}
	if _, err := FromConfig("broken", catalog); err == nil {
		t.Error("unknown kind should error")
	}
	if _, err := FromConfig("empty", catalog); err == nil {
		t.Error("command kind without command should error")
	}
}
