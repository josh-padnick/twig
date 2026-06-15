// `twig open` (and bare `twig <fragment>`): resolve, gate trust in the
// invoking terminal, then run the opener set. Injection-capable terminal
// openers carry `twig enter` into the new shell so setup output and [run]
// servers live where the user lands; with only non-terminal openers
// (editors, browsers) setup runs here first.
package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/josh-padnick/twig/internal/config"
	"github.com/josh-padnick/twig/internal/opener"
	"github.com/josh-padnick/twig/internal/setup"
	"github.com/josh-padnick/twig/internal/ui"
)

// openFlags are shared between the root command and `twig open`.
type openFlags struct {
	tab        bool
	runAfter   bool
	forceSetup bool
	noSetup    bool
	remote     bool
	verbose    bool
	with       []string
}

func addOpenFlags(cmd *cobra.Command, f *openFlags) {
	cmd.Flags().BoolVarP(&f.tab, "tab", "t", false, "enter in the current tab instead of a new window")
	cmd.Flags().BoolVar(&f.runAfter, "run", false, "run the [run] script after setup succeeds")
	cmd.Flags().BoolVar(&f.forceSetup, "setup", false, "force the setup script to re-run")
	cmd.Flags().BoolVar(&f.noSetup, "no-setup", false, "skip setup entirely")
	cmd.Flags().BoolVarP(&f.remote, "remote", "r", false, "search remote branches when nothing matches locally")
	cmd.Flags().BoolVarP(&f.verbose, "verbose", "v", false, "narrate each resolution step and scan location checked")
	cmd.Flags().StringSliceVar(&f.with, "with", nil, "openers to run (overrides the configured set)")
	cmd.MarkFlagsMutuallyExclusive("setup", "no-setup")
}

func newOpenCmd() *cobra.Command {
	var f openFlags
	cmd := &cobra.Command{
		Use:   "open [fragment]",
		Short: "Resolve a fragment and open the worktree with your configured tools",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			frag := ""
			if len(args) == 1 {
				frag = args[0]
			}
			return runOpen(frag, f)
		},
	}
	addOpenFlags(cmd, &f)
	return cmd
}

// runOpen is the shared implementation behind `twig open` and bare
// `twig <fragment>`.
func runOpen(frag string, f openFlags) error {
	c, err := resolveFragmentOrRemote(frag, f.remote, f.verbose)
	if err != nil {
		// First-run experience: an interactive miss with no config file on
		// disk is almost always missing roots — offer the wizard, then
		// retry once (the config now exists, so this can't recurse).
		if offerInitWizard(err) {
			return runOpen(frag, f)
		}
		return err
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	engine, err := newEngine()
	if err != nil {
		return err
	}

	// Trust is settled here, in the invoking terminal, so the repo's [open]
	// override can be honored and the injected `twig enter` never prompts.
	ld, lerr := setup.Load(c.Path)
	if lerr != nil && !errors.Is(lerr, setup.ErrNoManifest) {
		return lerr
	}
	trusted := false
	if ld != nil {
		if terr := engine.EnsureTrusted(ld, c.Path); terr == nil {
			trusted = true
		} else {
			if f.runAfter {
				return terr
			}
			ui.Warnf("opening with global defaults, setup skipped: %v", terr)
		}
	}
	if f.runAfter && ld == nil {
		return fmt.Errorf("nothing to run: no twig.toml governs %s", c.Path)
	}

	names, catalog := openerSet(f.with, cfg.Open, ld, trusted)
	if len(names) == 0 {
		return errors.New("no openers configured — set [open] default in config.toml")
	}
	var ops []opener.Opener
	for _, name := range names {
		op, err := opener.FromConfig(name, catalog)
		if err != nil {
			return err
		}
		ops = append(ops, op)
	}

	mode := opener.ModeWindow
	if f.tab {
		mode = opener.ModeCurrentTab
		if !anyOp(ops, opener.Opener.CanCurrentTab) {
			return errors.New("-t needs an opener that can enter the current tab (ghostty) — for an in-place cd use the tw shell function (twig shell-init)")
		}
	}

	// Setup placement: an injector hosts `twig enter` in its terminal;
	// otherwise (editors/browsers only) setup runs right here, first.
	enterCmd := ""
	if trusted || ld == nil {
		if anyOp(ops, opener.Opener.CanInject) {
			if ld != nil {
				enterCmd = buildEnterCmd(f)
			}
		} else {
			if f.runAfter {
				return fmt.Errorf("--run needs a terminal opener that can host it; none of [%s] can inject a command (add ghostty or a {{cmd}} template)", strings.Join(names, ", "))
			}
			if ld != nil {
				err := engine.Enter(setup.EnterOptions{
					Dir:        c.Path,
					ForceSetup: f.forceSetup,
					SkipSetup:  f.noSetup || (!cfg.SetupAuto() && !f.forceSetup),
				})
				if err != nil {
					return err
				}
			}
		}
	}

	// When a terminal opener will enter the current tab and actually change
	// directory, fold the other openers' launches into its single entry
	// line so the kept session runs cd → launches → enter in order, rather
	// than opening them in parallel while the cd waits for twig to exit.
	host := currentTabHost(ops, mode, c.Path)

	var failures []string
	var fold []string
	for i, op := range ops {
		ui.Stepf("opening %s with %s", ui.Tilde(c.Path), op.Name())
		if host >= 0 {
			if i == host {
				continue // opened last, hosting the folded launches
			}
			if cmd, ok := op.LaunchCmd(opener.Target{Dir: c.Path}); ok {
				fold = append(fold, cmd)
				continue
			}
		}
		// Non-host openers carry no entry command — the host runs it.
		enter := enterCmd
		if host >= 0 {
			enter = ""
		}
		t := opener.Target{Dir: c.Path, EnterCmd: enter, Mode: mode}
		if !op.CanInject() {
			t.EnterCmd = ""
		}
		if !op.CanCurrentTab() {
			t.Mode = opener.ModeWindow
		}
		if err := op.Open(t); err != nil {
			failures = append(failures, err.Error())
		}
	}
	if host >= 0 {
		t := opener.Target{Dir: c.Path, EnterCmd: enterCmd, Mode: mode, Fold: fold}
		if err := ops[host].Open(t); err != nil {
			failures = append(failures, err.Error())
		}
	}
	if len(failures) > 0 {
		return errors.New(strings.Join(failures, "; "))
	}
	return nil
}

// currentTabHost returns the index of the opener that will enter the current
// tab and so should host the others' launch commands, or -1 when none will.
// Folding only helps when there's a real cd to lead with: a new window is
// already a clean slate, and reusing the tab we're already in has no cd, so
// those keep opening each opener independently.
func currentTabHost(ops []opener.Opener, mode opener.TargetMode, dir string) int {
	if opener.SameDir(dir) {
		return -1
	}
	for i, op := range ops {
		if op.EntersCurrentTab(mode) {
			return i
		}
	}
	return -1
}

// openerSet computes which openers run and against which catalog:
// --with beats a trusted repo's [open].default, which beats the global
// default. A trusted repo's opener definitions overlay the global catalog;
// an untrusted manifest contributes nothing (its openers are arbitrary
// commands too).
func openerSet(with []string, global config.Open, ld *setup.Loaded, trusted bool) ([]string, config.Open) {
	catalog := config.Open{Default: global.Default, Openers: map[string]config.Opener{}}
	for name, spec := range global.Openers {
		catalog.Openers[name] = spec
	}
	if trusted && ld != nil {
		for name, spec := range ld.Manifest.Open.Openers {
			catalog.Openers[name] = spec
		}
		if len(ld.Manifest.Open.Default) > 0 {
			catalog.Default = ld.Manifest.Open.Default
		}
	}
	names := catalog.Default
	if len(with) > 0 {
		names = with
	}
	return names, catalog
}

// buildEnterCmd composes the `twig enter` command injected into terminals,
// using the running binary's own path because fresh windows may not have
// twig on PATH yet.
func buildEnterCmd(f openFlags) string {
	exe, err := os.Executable()
	if err != nil {
		exe = "twig"
	}
	parts := []string{opener.Quote(exe), "enter"}
	if f.runAfter {
		parts = append(parts, "--run")
	}
	if f.forceSetup {
		parts = append(parts, "--setup")
	}
	if f.noSetup {
		parts = append(parts, "--no-setup")
	}
	return strings.Join(parts, " ")
}

func anyOp(ops []opener.Opener, pred func(opener.Opener) bool) bool {
	for _, op := range ops {
		if pred(op) {
			return true
		}
	}
	return false
}
