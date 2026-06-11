package shell

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderSubstitutions(t *testing.T) {
	for _, sh := range []string{"zsh", "bash", "fish"} {
		out, err := Render(sh, "tw", "/usr/local/bin/twig")
		if err != nil {
			t.Fatalf("%s: %v", sh, err)
		}
		if strings.Contains(out, "%CMD%") || strings.Contains(out, "%TWIG%") {
			t.Errorf("%s: unsubstituted placeholders:\n%s", sh, out)
		}
		if !strings.Contains(out, "'/usr/local/bin/twig'") {
			t.Errorf("%s: twig path not quoted in:\n%s", sh, out)
		}
	}
}

func TestRenderCustomCommandName(t *testing.T) {
	out, err := Render("zsh", "j", "/bin/twig")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "j() {") {
		t.Errorf("custom name missing:\n%s", out)
	}
}

func TestRenderUnsupportedShell(t *testing.T) {
	if _, err := Render("powershell", "tw", "/bin/twig"); err == nil {
		t.Fatal("expected error for unsupported shell")
	}
}

func TestRenderQuotesAwkwardPaths(t *testing.T) {
	out, err := Render("zsh", "tw", "/Users/it's me/bin/twig")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `'/Users/it'\''s me/bin/twig'`) {
		t.Errorf("posix quoting wrong:\n%s", out)
	}
	out, err = Render("fish", "tw", "/Users/it's me/bin/twig")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `'/Users/it\'s me/bin/twig'`) {
		t.Errorf("fish quoting wrong:\n%s", out)
	}
}

// Smoke test: each shell, when present, must accept the snippet and define
// the function.
func TestRenderedSnippetDefinesFunction(t *testing.T) {
	cases := []struct {
		shell string
		check string
	}{
		{"zsh", "source %f && type tw"},
		{"bash", "source %f && type tw"},
		{"fish", "source %f; and type tw"},
	}
	for _, tc := range cases {
		t.Run(tc.shell, func(t *testing.T) {
			bin, err := exec.LookPath(tc.shell)
			if err != nil {
				t.Skipf("%s not installed", tc.shell)
			}
			out, err := Render(tc.shell, "tw", "/usr/local/bin/twig")
			if err != nil {
				t.Fatal(err)
			}
			f := filepath.Join(t.TempDir(), "init."+tc.shell)
			if err := os.WriteFile(f, []byte(out), 0o644); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(bin, "-c", strings.ReplaceAll(tc.check, "%f", f))
			if outBytes, err := cmd.CombinedOutput(); err != nil {
				t.Errorf("%s rejected the snippet: %v\n%s", tc.shell, err, outBytes)
			}
		})
	}
}
