// The setup/run engine: twig's on-entry behavior. Enter enforces the trust
// gate, runs the [setup] script only when its inputs changed (manifest hash
// + watch-file hashes), records success in the git dir, and optionally runs
// the [run] script in the foreground with the child's exit code propagated.
package setup

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"

	"github.com/josh-padnick/twig/internal/gitx"
	"github.com/josh-padnick/twig/internal/trust"
	"github.com/josh-padnick/twig/internal/ui"
)

// ExitCodeError carries a child process's exit code up to main. The child
// already wrote its own output, so the CLI exits with the code silently.
type ExitCodeError struct {
	Code int
}

func (e *ExitCodeError) Error() string {
	return fmt.Sprintf("exited with status %d", e.Code)
}

// Engine runs the on-entry steps for a worktree.
type Engine struct {
	Trust   *trust.Store
	Version string // stamped into the state marker
	// Confirm asks the user a yes/no question; nil means non-interactive,
	// in which case an untrusted manifest is an error instead of a prompt.
	Confirm func(prompt string) (bool, error)
}

// EnterOptions are the flags of one entry.
type EnterOptions struct {
	Dir        string // worktree directory (absolute)
	ForceSetup bool   // --setup: run even when state says unchanged
	SkipSetup  bool   // --no-setup (or setup.auto=false)
	RunAfter   bool   // --run: execute [run] after setup succeeds
}

// Enter is the on-entry primitive every arrival path converges on: trust
// gate, setup-if-needed, optional [run]. A missing twig.toml is a silent
// no-op unless the caller asked for [run].
func (e *Engine) Enter(opts EnterOptions) error {
	ld, err := Load(opts.Dir)
	if errors.Is(err, ErrNoManifest) {
		if opts.RunAfter {
			return err
		}
		return nil
	}
	if err != nil {
		return err
	}
	for _, w := range ld.Warnings {
		ui.Warnf("%s", w)
	}

	if err := e.ensureTrusted(ld, opts.Dir); err != nil {
		return err
	}

	if !opts.SkipSetup && strings.TrimSpace(ld.Manifest.Setup.Run) != "" {
		if opts.ForceSetup || needsSetup(opts.Dir, ld) {
			ui.Infof("twig: running setup from %s", ld.Path)
			if err := e.runScript(opts.Dir, ld.Manifest.Setup.Run, false); err != nil {
				return fmt.Errorf("setup failed: %v — fix it and re-enter, or force a re-run with --setup", err)
			}
			e.recordSuccess(opts.Dir, ld)
		}
	}

	if opts.RunAfter {
		script := ld.Manifest.Run.Run
		if strings.TrimSpace(script) == "" {
			return fmt.Errorf("no [run] script in %s", ld.Path)
		}
		return e.runScript(opts.Dir, script, true)
	}
	return nil
}

// ensureTrusted enforces the direnv-style gate: show the manifest, prompt
// when interactive, otherwise fail with the twig trust instruction.
func (e *Engine) ensureTrusted(ld *Loaded, dir string) error {
	ok, err := e.Trust.Check(ld.Path, ld.Content)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	ui.Infof("twig: %s is not trusted; its scripts are blocked until you approve it:\n", ld.Path)
	fmt.Fprintf(os.Stderr, "%s\n", ld.Content)
	if e.Confirm != nil {
		yes, cerr := e.Confirm("trust this twig.toml and continue?")
		if cerr == nil && yes {
			return e.Trust.Approve(ld.Path, ld.Content)
		}
	}
	return fmt.Errorf("twig.toml not trusted — run: twig trust %s", dir)
}

// recordSuccess writes the state marker; a failure to record is a warning,
// not an error, because setup itself succeeded.
func (e *Engine) recordSuccess(dir string, ld *Loaded) {
	st := State{
		Version:        1,
		ManifestPath:   ld.Path,
		ManifestSHA256: hashBytes(ld.Content),
		Watch:          watchHashes(dir, ld.Manifest.Setup.Watch),
		LastSuccess:    time.Now().UTC(),
		TwigVersion:    e.Version,
	}
	if err := writeState(dir, st); err != nil {
		ui.Warnf("could not record setup state (setup will re-run next entry): %v", err)
	}
}

// runScript executes a manifest script with bash in the worktree,
// streaming output. In foreground mode (the [run] script) twig ignores
// SIGINT — Ctrl-C goes to the child, which is the process the user means
// to stop — and the child's exit code is propagated via ExitCodeError.
func (e *Engine) runScript(dir, script string, foreground bool) error {
	bash, err := exec.LookPath("bash")
	if err != nil {
		bash = "/bin/bash"
	}
	cmd := exec.Command(bash, "-eu", "-o", "pipefail", "-c", script)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), envFor(dir)...)

	if !foreground {
		return cmd.Run()
	}
	signal.Ignore(os.Interrupt)
	defer signal.Reset(os.Interrupt)
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return &ExitCodeError{Code: ee.ExitCode()}
		}
		return err
	}
	return nil
}

// envFor exposes worktree context to manifest scripts.
func envFor(dir string) []string {
	env := []string{"TWIG_WORKTREE=" + dir}
	if branch, err := gitx.CurrentBranch(dir); err == nil && branch != "" {
		env = append(env, "TWIG_BRANCH="+branch)
	}
	if wts, err := gitx.Worktrees(dir); err == nil && len(wts) > 0 {
		env = append(env, "TWIG_REPO_ROOT="+wts[0].Path)
	}
	return env
}
