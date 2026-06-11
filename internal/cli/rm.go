// `twig rm` removes a worktree: confirm, `git worktree remove`, then
// `git worktree prune`. The branch is deliberately kept — deleting history
// is git's job, twig only manages directories.
package cli

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/josh-padnick/twig/internal/gitx"
	"github.com/josh-padnick/twig/internal/ui"
)

func newRmCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "rm <fragment>",
		Short: "Remove a worktree (git worktree remove + prune)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := resolveFragment(args[0])
			if err != nil {
				return err
			}
			wts, err := gitx.Worktrees(c.Path)
			if err != nil {
				return fmt.Errorf("%s is not a usable git worktree: %w", c.Path, err)
			}
			mainDir := wts[0].Path // first porcelain entry is the main worktree
			if samePath(c.Path, mainDir) {
				return fmt.Errorf("refusing to remove the main worktree %s", mainDir)
			}
			branch, _ := gitx.CurrentBranch(c.Path)
			if !force {
				// The confirmation calls out anything the user might want
				// to deal with first: unpushed commits, uncommitted
				// changes. Neither is lost by removal (the branch is
				// kept), but both deserve a heads-up.
				details := []string{}
				if branch != "" {
					details = append(details, "branch "+branch)
				}
				if ahead, ok := gitx.AheadOfUpstream(c.Path); ok && ahead > 0 {
					details = append(details, fmt.Sprintf("%d commit(s) not pushed to upstream", ahead))
				}
				if dirty, err := gitx.IsDirty(c.Path); err == nil && dirty {
					details = append(details, "uncommitted changes")
				}
				label := c.Path
				if len(details) > 0 {
					label = fmt.Sprintf("%s (%s)", c.Path, strings.Join(details, ", "))
				}
				yes, err := ui.ConfirmYN("remove worktree "+label+"?", false)
				if err != nil {
					return fmt.Errorf("confirmation needs a terminal; use --force non-interactively (%v)", err)
				}
				if !yes {
					return errors.New("cancelled")
				}
			}
			if err := gitx.RemoveWorktree(mainDir, c.Path, force); err != nil {
				return err
			}
			if err := gitx.Prune(mainDir); err != nil {
				return err
			}
			if branch != "" {
				ui.Infof("twig: removed worktree %s — branch %s kept (delete with: git -C %s branch -d %s)", c.Path, branch, mainDir, branch)
			} else {
				ui.Infof("twig: removed worktree %s", c.Path)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "skip confirmation and remove even when dirty")
	return cmd
}

// samePath compares two paths through symlinks.
func samePath(a, b string) bool {
	ra, errA := filepath.EvalSymlinks(a)
	rb, errB := filepath.EvalSymlinks(b)
	if errA != nil || errB != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	return ra == rb
}
