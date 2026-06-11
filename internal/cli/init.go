// `twig init` — the first-run wizard. Discovers likely project roots on
// disk (ranked by how twig-shaped they look), detects installed editors,
// writes a commented config.toml, and offers shell integration. Also
// invoked as an offer when an interactive open fails with no config file.
package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/josh-padnick/twig/internal/config"
	"github.com/josh-padnick/twig/internal/initwiz"
	"github.com/josh-padnick/twig/internal/pick"
	"github.com/josh-padnick/twig/internal/ui"
)

func newInitCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Interactively generate your twig config",
		Long: "Scans well-known project locations for likely roots, detects installed\n" +
			"editors and your shell, and writes a commented ~/.config/twig/config.toml.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInitWizard(force)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing config file")
	return cmd
}

// runInitWizard drives the interactive flow. It needs a controlling
// terminal; scripts get a clear error instead of a hung prompt.
func runInitWizard(force bool) error {
	if !ui.HasTTY() {
		return errors.New("twig init is interactive — run it from a terminal")
	}
	cfgPath, err := config.Path()
	if err != nil {
		return err
	}
	if _, err := os.Stat(cfgPath); err == nil && !force {
		return fmt.Errorf("config already exists at %s (re-run with --force to overwrite)", cfgPath)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	ui.Infof("twig init — let's find your worktrees.\n")

	roots, err := chooseRoots(home)
	if err != nil {
		return err
	}

	ui.Infof("Providers active automatically when present: conductor (~/conductor/workspaces), claude-code.")

	editors := chooseEditors()

	content := initwiz.Generate(home, initwiz.Answers{Roots: roots, Editors: editors})
	ui.Infof("\nThis will be written to %s:\n", cfgPath)
	fmt.Fprintf(os.Stderr, "%s\n", content)
	yes, err := ui.ConfirmYN("Write it?", true)
	if err != nil {
		return err
	}
	if !yes {
		return errors.New("cancelled — nothing written")
	}
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		return err
	}
	ui.Infof("twig: wrote %s", cfgPath)

	offerShellInit(home)

	ui.Infof("\nDone. Try:  twig <fragment>")
	return nil
}

// chooseRoots discovers candidates and multi-selects, falling back to a
// typed path when nothing was found.
func chooseRoots(home string) ([]string, error) {
	cands := initwiz.Discover(home)
	if len(cands) == 0 {
		line, err := ui.ReadLine("No project roots detected. Enter one to scan (empty to skip): ")
		if err != nil || line == "" {
			return nil, nil
		}
		if line == "~" || strings.HasPrefix(line, "~/") {
			line = filepath.Join(home, strings.TrimPrefix(line[1:], "/"))
		}
		if fi, err := os.Stat(line); err != nil || !fi.IsDir() {
			return nil, fmt.Errorf("not a directory: %s", line)
		}
		return []string{line}, nil
	}

	ui.Infof("Select the roots twig should scan (TAB toggles, Enter confirms):")
	chosen, err := pick.ManyOf(cands, func(c initwiz.RootCandidate) string {
		label := fmt.Sprintf("%s — %d repo(s)", displayPath(home, c.Path), c.RepoCount)
		if c.ClaudeCount > 0 {
			label += fmt.Sprintf(", %d with Claude worktrees", c.ClaudeCount)
		}
		return label
	})
	if err != nil {
		return nil, err
	}
	var roots []string
	for _, c := range chosen {
		roots = append(roots, c.Path)
	}
	return roots, nil
}

// chooseEditors asks per detected editor; declining everything is fine.
func chooseEditors() []initwiz.Editor {
	var chosen []initwiz.Editor
	for _, e := range initwiz.DetectEditors() {
		yes, err := ui.ConfirmYN(fmt.Sprintf("Also open %s when entering a worktree?", e.Name), false)
		if err == nil && yes {
			chosen = append(chosen, e)
		}
	}
	return chosen
}

// offerShellInit detects the shell and offers to append the eval line.
func offerShellInit(home string) {
	rc, ok := initwiz.DetectShellRC(home)
	if !ok {
		return
	}
	yes, err := ui.ConfirmYN(fmt.Sprintf("Add the tw shell function to %s?", displayPath(home, rc.Path)), false)
	if err != nil || !yes {
		ui.Infof("To enable in-place cd later, add:  %s", rc.Line)
		return
	}
	added, err := initwiz.Install(rc)
	switch {
	case err != nil:
		ui.Warnf("could not update %s: %v", rc.Path, err)
	case added:
		ui.Infof("twig: added to %s — restart your shell (or source it) to get `tw`", displayPath(home, rc.Path))
	default:
		ui.Infof("twig: %s already references twig shell-init — nothing to do", displayPath(home, rc.Path))
	}
}

// displayPath shortens home-prefixed paths for prompts.
func displayPath(home, p string) string {
	if rel, ok := strings.CutPrefix(p, home+string(os.PathSeparator)); ok {
		return "~" + string(os.PathSeparator) + rel
	}
	return p
}
