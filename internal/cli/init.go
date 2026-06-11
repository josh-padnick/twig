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

	ui.Infof("twig init: let's find your worktrees.\n")

	chosen, err := chooseRoots(home)
	if err != nil {
		return err
	}
	roots := make([]string, 0, len(chosen))
	for _, c := range chosen {
		roots = append(roots, c.Path)
	}
	if len(roots) == 0 {
		ui.Infof("No roots selected: twig will still resolve inside any repo you're in.")
	} else {
		ui.Infof("Roots to scan: %s", displayPaths(home, roots))
	}

	reportDetectedTools(home, chosen)

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
// typed path when nothing was found. Roots holding Claude worktrees are
// preselected, so Enter alone picks the likeliest answer.
func chooseRoots(home string) ([]initwiz.RootCandidate, error) {
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
		return []initwiz.RootCandidate{{Path: line}}, nil
	}

	return pick.ManyOf(cands,
		func(c initwiz.RootCandidate) string {
			label := fmt.Sprintf("%s (%d repos", displayPath(home, c.Path), c.RepoCount)
			if c.ClaudeCount > 0 {
				label += fmt.Sprintf(", %d with Claude Code worktrees", c.ClaudeCount)
			}
			return label + ")"
		},
		"Which folders should twig search for worktrees? TAB selects, Enter confirms.",
		func(c initwiz.RootCandidate) bool { return c.ClaudeCount > 0 },
	)
}

// reportDetectedTools tells the user, concretely, which worktree-creating
// tools twig found and what that means. Silent when nothing was detected.
func reportDetectedTools(home string, chosen []initwiz.RootCandidate) {
	var lines []string
	if initwiz.HasConductor(home) {
		lines = append(lines, fmt.Sprintf("  Conductor: workspaces in %s will be found automatically",
			displayPath(home, filepath.Join(home, "conductor", "workspaces"))))
	}
	var claudeRoots []string
	for _, c := range chosen {
		if c.ClaudeCount > 0 {
			claudeRoots = append(claudeRoots, c.Path)
		}
	}
	if len(claudeRoots) > 0 {
		lines = append(lines, fmt.Sprintf("  Claude Code: worktrees in repos under %s will be found automatically",
			displayPaths(home, claudeRoots)))
	}
	if len(lines) == 0 {
		return
	}
	ui.Infof("\nDetected tools that create worktrees:")
	for _, line := range lines {
		ui.Infof("%s", line)
	}
}

// chooseEditors asks per detected editor; declining everything is fine.
func chooseEditors() []initwiz.Editor {
	var chosen []initwiz.Editor
	for _, e := range initwiz.DetectEditors() {
		yes, err := ui.ConfirmYN(fmt.Sprintf("Should I open %s when entering a worktree?", e.DisplayName), false)
		if err == nil && yes {
			chosen = append(chosen, e)
		}
	}
	return chosen
}

// offerShellInit explains what the tw function buys, then offers to
// append the eval line.
func offerShellInit(home string) {
	rc, ok := initwiz.DetectShellRC(home)
	if !ok {
		return
	}
	ui.Infof("\ntwig can add a small `tw` function to your shell. It lets you jump")
	ui.Infof("to a worktree inside the current terminal instead of opening a new")
	ui.Infof("window: `tw gould` cd's this shell there and runs setup.")
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

// displayPaths renders several paths for prompts, ~-shortened.
func displayPaths(home string, paths []string) string {
	out := make([]string, len(paths))
	for i, p := range paths {
		out[i] = displayPath(home, p)
	}
	return strings.Join(out, ", ")
}
