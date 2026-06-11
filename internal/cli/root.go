// Package cli owns the cobra command tree. Subcommand files in this package
// wire flags and delegate to the internal packages; business logic lives in
// resolve, setup, opener, and friends — not here.
package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/josh-padnick/twig/internal/ui"
)

// Execute runs the root command and returns a process exit code. Errors are
// printed to stderr here (cobra's own printing is silenced) so no failure
// path ever writes to stdout.
func Execute(version, commit string) int {
	root := newRootCmd(version, commit)
	if err := root.Execute(); err != nil {
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
// picker over that repo's worktrees (gw parity).
func newRootCmd(version, commit string) *cobra.Command {
	root := &cobra.Command{
		Use:   "twig [fragment]",
		Short: "Git worktrees, one short command away",
		Long: "twig resolves a fragment — a branch name, branch suffix, directory slug,\n" +
			"or literal path — to a git worktree and enters it: opening your terminal\n" +
			"or editor there and running the worktree's trust-gated setup script.",
		Version:       fmt.Sprintf("%s (commit %s)", version, commit),
		Args:          cobra.ArbitraryArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return fmt.Errorf("opening worktrees lands in M4 — until then: cd \"$(twig cd %s)\"", args[0])
		},
	}
	root.AddCommand(newCdCmd(), newListCmd())
	return root
}
