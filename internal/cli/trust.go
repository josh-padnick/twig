// `twig trust` approves a worktree's twig.toml for execution (direnv-style):
// it prints the manifest so the user sees exactly what they're approving,
// then records the path+content hash. --list, --revoke, and --show manage
// and inspect approvals.
package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/josh-padnick/twig/internal/setup"
	"github.com/josh-padnick/twig/internal/trust"
	"github.com/josh-padnick/twig/internal/ui"
)

func newTrustCmd() *cobra.Command {
	var listFlag, revokeFlag, showFlag bool
	cmd := &cobra.Command{
		Use:   "trust [fragment]",
		Short: "Approve a worktree's twig.toml for execution",
		Long: "Shows the twig.toml governing the given worktree (default: the current\n" +
			"directory) and records your approval. twig refuses to run setup or open\n" +
			"scripts from a manifest you haven't approved; any edit to the file\n" +
			"requires re-approval.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := trust.NewStore()
			if err != nil {
				return err
			}
			if listFlag {
				return printTrustList(cmd, store)
			}

			dir, err := os.Getwd()
			if err != nil {
				return err
			}
			if len(args) == 1 {
				c, err := resolveFragment(args[0])
				if err != nil {
					return err
				}
				dir = c.Path
			}
			manifest, err := setup.FindManifest(dir)
			if err != nil {
				return err
			}

			if revokeFlag {
				n, err := store.Revoke(manifest)
				if err != nil {
					return err
				}
				ui.Infof("revoked %d approval(s) for %s", n, manifest)
				return nil
			}

			content, err := os.ReadFile(manifest)
			if err != nil {
				return err
			}
			if showFlag {
				fmt.Fprintf(cmd.OutOrStdout(), "# %s\n%s", manifest, content)
				return nil
			}
			ui.Infof("%s:\n", manifest)
			fmt.Fprintf(os.Stderr, "%s\n", content)
			if err := store.Approve(manifest, content); err != nil {
				return err
			}
			ui.Infof("twig: trusted %s", manifest)
			return nil
		},
	}
	cmd.Flags().BoolVar(&listFlag, "list", false, "list all approved manifests")
	cmd.Flags().BoolVar(&revokeFlag, "revoke", false, "revoke approval for the manifest")
	cmd.Flags().BoolVar(&showFlag, "show", false, "print the manifest without approving it")
	cmd.MarkFlagsMutuallyExclusive("list", "revoke", "show")
	return cmd
}

// printTrustList renders the approval store, newest first.
func printTrustList(cmd *cobra.Command, store *trust.Store) error {
	entries, err := store.List()
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		ui.Infof("no trusted manifests")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "MANIFEST\tAPPROVED")
	for _, e := range entries {
		fmt.Fprintf(w, "%s\t%s\n", e.Path, e.ApprovedAt.Local().Format("2006-01-02 15:04"))
	}
	return w.Flush()
}
