// `twig cd` resolves a fragment and prints the worktree path — the only
// thing this command ever writes to stdout, so shell functions can rely on
// `cd "$(twig cd frag)"`. Prompts and the picker use the terminal directly.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newCdCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cd [fragment]",
		Short: "Resolve a fragment and print the worktree path",
		Long: "Resolves a fragment to a worktree directory and prints the path on stdout.\n" +
			"Designed for command substitution: cd \"$(twig cd foo)\". With no fragment,\n" +
			"inside a repo, offers a picker over that repo's worktrees.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			frag := ""
			if len(args) == 1 {
				frag = args[0]
			}
			remoteFlag, _ := cmd.Flags().GetBool("remote")
			c, err := resolveFragmentOrRemote(frag, remoteFlag)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), c.Path)
			return nil
		},
	}
	cmd.Flags().BoolP("remote", "r", false, "search remote branches when nothing matches locally")
	return cmd
}
