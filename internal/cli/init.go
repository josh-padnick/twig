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

	if reportDetectedTools(home, chosen) {
		// A distinct beat: let the user read the detection summary before
		// the opener questions start.
		_, _ = ui.ReadLine("\nPress ENTER to continue ")
	}

	editors, err := chooseEditors()
	if err != nil {
		return err
	}

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

	return pick.Checklist(
		"Which folders should twig search for worktrees?",
		cands,
		func(c initwiz.RootCandidate) string {
			label := fmt.Sprintf("%s (%d repos", displayPath(home, c.Path), c.RepoCount)
			if c.ClaudeCount > 0 {
				label += fmt.Sprintf(", %d with Claude Code worktrees", c.ClaudeCount)
			}
			return label + ")"
		},
		func(c initwiz.RootCandidate) bool { return c.ClaudeCount > 0 },
	)
}

// reportDetectedTools tells the user, concretely, which worktree-creating
// tools twig found and where it will look for their work. Returns whether
// anything was detected and printed (silent otherwise), so the caller can
// pause for reading.
func reportDetectedTools(home string, chosen []initwiz.RootCandidate) bool {
	var lines []string
	if initwiz.HasConductor(home) {
		lines = append(lines, "  Conductor: workspaces in "+displayPath(home, filepath.Join(home, "conductor", "workspaces")))
	}
	var claudeRoots []string
	for _, c := range chosen {
		if c.ClaudeCount > 0 {
			claudeRoots = append(claudeRoots, c.Path)
		}
	}
	if len(claudeRoots) > 0 {
		lines = append(lines, "  Claude Code: worktrees in repos under "+displayPaths(home, claudeRoots))
	}
	if initwiz.HasCodex(home) {
		lines = append(lines, "  Codex: cloud sessions live on GitHub as branches; fetch one with `twig -r <fragment>`")
	}
	if len(lines) == 0 {
		return false
	}
	ui.Infof("\nI detected the following tools. In addition to the above paths, twig")
	ui.Infof("will also look for git worktrees in the following paths:")
	for _, line := range lines {
		ui.Infof("%s", line)
	}
	return true
}

// chooseEditors multi-selects among detected editors with the same
// checklist UX as the roots step; selecting none is fine (a terminal
// still opens via the default ghostty opener).
func chooseEditors() ([]initwiz.Editor, error) {
	editors := initwiz.DetectEditors()
	if len(editors) == 0 {
		return nil, nil
	}
	ui.Infof("")
	return pick.Checklist(
		"Which tools should I automatically open when you enter a worktree?",
		editors,
		func(e initwiz.Editor) string { return e.DisplayName },
		nil,
	)
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
