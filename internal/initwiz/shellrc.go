// Shell-rc installation for `twig init`: detect the user's shell, locate
// its config file, and (with consent, handled by the CLI) append the
// shell-init line — idempotently, so re-running init never duplicates it.
package initwiz

import (
	"os"
	"path/filepath"
	"strings"
)

// ShellRC describes where and what to install for the user's shell.
type ShellRC struct {
	Shell string // zsh | bash | fish
	Path  string // rc file to append to
	Line  string // the line to add
}

// DetectShellRC inspects $SHELL and returns the matching rc target.
func DetectShellRC(home string) (ShellRC, bool) {
	switch filepath.Base(os.Getenv("SHELL")) {
	case "zsh":
		rcDir := os.Getenv("ZDOTDIR")
		if rcDir == "" {
			rcDir = home
		}
		return ShellRC{Shell: "zsh", Path: filepath.Join(rcDir, ".zshrc"), Line: `eval "$(twig shell-init zsh)"`}, true
	case "bash":
		return ShellRC{Shell: "bash", Path: filepath.Join(home, ".bashrc"), Line: `eval "$(twig shell-init bash)"`}, true
	case "fish":
		return ShellRC{Shell: "fish", Path: filepath.Join(home, ".config", "fish", "config.fish"), Line: "twig shell-init fish | source"}, true
	}
	return ShellRC{}, false
}

// Install appends the shell-init line to the rc file with a marker comment.
// Returns false without writing when the file already references
// twig shell-init (idempotence).
func Install(rc ShellRC) (bool, error) {
	existing, err := os.ReadFile(rc.Path)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	if strings.Contains(string(existing), "twig shell-init") {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(rc.Path), 0o755); err != nil {
		return false, err
	}
	f, err := os.OpenFile(rc.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return false, err
	}
	defer f.Close()
	prefix := "\n"
	if len(existing) == 0 || strings.HasSuffix(string(existing), "\n") {
		prefix = ""
	}
	if _, err := f.WriteString(prefix + "\n# added by twig init\n" + rc.Line + "\n"); err != nil {
		return false, err
	}
	return true, nil
}
