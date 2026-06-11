// `twig enter` is the on-entry primitive: trust check, setup-if-needed,
// optional [run] — executed in the current terminal. Every arrival path
// converges here: `twig open` injects it into the new window, the `tw`
// shell function calls it after cd, and users can run it by hand after a
// manual cd (direnv's shell hook, made explicit).
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/josh-padnick/twig/internal/setup"
	"github.com/josh-padnick/twig/internal/trust"
	"github.com/josh-padnick/twig/internal/ui"
)

func newEnterCmd() *cobra.Command {
	var runAfter, forceSetup, noSetup bool
	cmd := &cobra.Command{
		Use:   "enter [dir]",
		Short: "Run the on-entry steps (trust check + setup) in the current terminal",
		Long: "Locates the twig.toml governing dir (default: the current directory),\n" +
			"enforces the trust gate, runs [setup] when its inputs changed, and with\n" +
			"--run starts the [run] script in the foreground.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return err
			}
			if len(args) == 1 {
				dir, err = filepath.Abs(args[0])
				if err != nil {
					return err
				}
				if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
					return fmt.Errorf("not a directory: %s", dir)
				}
			}
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			engine, err := newEngine()
			if err != nil {
				return err
			}
			return engine.Enter(setup.EnterOptions{
				Dir:        dir,
				ForceSetup: forceSetup,
				SkipSetup:  noSetup || (!cfg.SetupAuto() && !forceSetup),
				RunAfter:   runAfter,
			})
		},
	}
	cmd.Flags().BoolVar(&runAfter, "run", false, "run the [run] script after setup succeeds")
	cmd.Flags().BoolVar(&forceSetup, "setup", false, "force the setup script to re-run")
	cmd.Flags().BoolVar(&noSetup, "no-setup", false, "skip setup entirely")
	cmd.MarkFlagsMutuallyExclusive("setup", "no-setup")
	return cmd
}

// newEngine wires the setup engine with the trust store and an interactive
// confirm bound to the controlling terminal.
func newEngine() (*setup.Engine, error) {
	store, err := trust.NewStore()
	if err != nil {
		return nil, err
	}
	return &setup.Engine{
		Trust:   store,
		Version: buildVersion,
		Confirm: func(prompt string) (bool, error) { return ui.ConfirmYN(prompt, false) },
	}, nil
}
