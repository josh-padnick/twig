// `twig list` prints every worktree twig can see — the current repo's plus
// everything under providers and roots — with branch and working-tree status.
package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/josh-padnick/twig/internal/gitx"
	"github.com/josh-padnick/twig/internal/resolve"
	"github.com/josh-padnick/twig/internal/ui"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all known worktrees",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := newResolver()
			if err != nil {
				return err
			}
			cands := r.All()
			if len(cands) == 0 {
				ui.Infof("no worktrees found — configure roots in config.toml or run from inside a repo")
				return nil
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "PATH\tBRANCH\tSTATUS")
			for _, c := range cands {
				branch, status := describe(c)
				fmt.Fprintf(w, "%s\t%s\t%s\n", c.Path, branch, status)
			}
			return w.Flush()
		},
	}
}

// describe labels one worktree row: stale records (directory gone), broken
// checkouts (git can't read them, e.g. a worktree whose parent repo was
// pruned), detached HEADs, and the usual dirty/clean.
func describe(c resolve.Candidate) (branch, status string) {
	if c.Stale {
		return valueOr(c.Branch, "-"), "stale"
	}
	branch = c.Branch
	if branch == "" {
		b, err := gitx.CurrentBranch(c.Path)
		if err != nil {
			return "-", "broken"
		}
		branch = b
	}
	if branch == "" {
		branch = "(detached)"
	}
	dirty, err := gitx.IsDirty(c.Path)
	switch {
	case err != nil:
		status = "broken"
	case dirty:
		status = "dirty"
	default:
		status = "clean"
	}
	return branch, status
}

func valueOr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
