// `twig shell-init` prints the tw() shell function to stdout for eval'ing
// from shell config — the in-place-cd companion to `twig open`.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/josh-padnick/twig/internal/shell"
)

func newShellInitCmd() *cobra.Command {
	var cmdName string
	cmd := &cobra.Command{
		Use:   "shell-init <zsh|bash|fish>",
		Short: "Emit the tw shell function for in-place cd",
		Long: "Prints a small shell function that cd's into a resolved worktree in the\n" +
			"current shell and runs the on-entry steps. Install it from shell config:\n\n" +
			"  eval \"$(twig shell-init zsh)\"      # ~/.zshrc\n" +
			"  eval \"$(twig shell-init bash)\"     # ~/.bashrc\n" +
			"  twig shell-init fish | source       # ~/.config/fish/config.fish",
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"zsh", "bash", "fish"},
		RunE: func(cmd *cobra.Command, args []string) error {
			twigPath, err := os.Executable()
			if err != nil {
				twigPath = "twig"
			}
			out, err := shell.Render(args[0], cmdName, twigPath)
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), out)
			return nil
		},
	}
	cmd.Flags().StringVar(&cmdName, "cmd", "tw", "name of the emitted shell function")
	return cmd
}
