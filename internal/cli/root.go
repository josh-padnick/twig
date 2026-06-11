// Package cli owns the cobra command tree. Subcommand files in this package
// wire flags and delegate to the internal packages; business logic lives in
// resolve, setup, opener, and friends — not here.
package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/josh-padnick/twig/internal/gitx"
	"github.com/josh-padnick/twig/internal/setup"
	"github.com/josh-padnick/twig/internal/ui"
)

// buildVersion is the ldflags version, stamped into setup state markers.
var buildVersion = "dev"

// Execute runs the root command and returns a process exit code. Errors are
// printed to stderr here (cobra's own printing is silenced) so no failure
// path ever writes to stdout. A child process's exit code (from [run]
// scripts) passes through unchanged and unannounced — the child already
// spoke for itself.
func Execute(version, commit string) int {
	buildVersion = version
	root := newRootCmd(version, commit)
	if err := root.Execute(); err != nil {
		var exitErr *setup.ExitCodeError
		if errors.As(err, &exitErr) {
			return exitErr.Code
		}
		if !errors.Is(err, errAlreadyReported) {
			ui.Errorf("%v", err)
		}
		return 1
	}
	return 0
}

// errAlreadyReported wraps failures whose details were already printed to
// the user (e.g. a setup script that streamed its own error output); the
// root handler then exits nonzero without printing a redundant line.
var errAlreadyReported = errors.New("already reported")

// newRootCmd builds the root `twig` command. With a fragment argument it
// behaves like `twig open <fragment>`; with none, inside a repo, it offers a
// picker over that repo's worktrees (gw parity), and outside one it shows
// help.
func newRootCmd(version, commit string) *cobra.Command {
	var f openFlags
	root := &cobra.Command{
		Use:   "twig [fragment]",
		Short: "Git worktrees, one short command away",
		Long: "twig resolves a fragment — a branch name, branch suffix, directory slug,\n" +
			"or literal path — to a git worktree and enters it: opening your terminal\n" +
			"or editor there and running the worktree's trust-gated setup script.\n" +
			"A fragment that collides with a subcommand name: use `twig open <fragment>`.",
		Version:       fmt.Sprintf("%s (commit %s)", version, commit),
		Args:          cobra.ArbitraryArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return fmt.Errorf("expected one fragment, got %d arguments", len(args))
			}
			if len(args) == 0 {
				if cwd, err := os.Getwd(); err == nil && gitx.InRepo(cwd) {
					return runOpen("", f)
				}
				return cmd.Help()
			}
			return runOpen(args[0], f)
		},
	}
	addOpenFlags(root, &f)
	root.AddCommand(newCdCmd(), newListCmd(), newTrustCmd(), newEnterCmd(), newOpenCmd(), newRmCmd(), newDoctorCmd(), newShellInitCmd())
	return root
}
