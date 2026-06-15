// Remote-branch pickup glue: when local resolution finds nothing and the
// user opted in (-r or remote.auto), search the remotes of repos already on
// disk, confirm, then fetch and create the worktree so the normal open or
// cd flow can continue as if it had always existed locally.
package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/josh-padnick/twig/internal/config"
	"github.com/josh-padnick/twig/internal/pick"
	"github.com/josh-padnick/twig/internal/remote"
	"github.com/josh-padnick/twig/internal/resolve"
	"github.com/josh-padnick/twig/internal/ui"
)

// resolveFragmentOrRemote resolves locally first; on a clean no-match it
// optionally falls through to remote pickup. Every other error (ambiguity,
// stale records, git failures) passes through untouched.
func resolveFragmentOrRemote(frag string, remoteFlag bool) (resolve.Candidate, error) {
	c, err := resolveFragment(frag)
	var noMatch *resolve.NoMatchError
	if err == nil || !errors.As(err, &noMatch) {
		return c, err
	}
	cfg, cfgErr := loadConfig()
	if cfgErr != nil {
		return c, err
	}
	if !remoteFlag && !cfg.Remote.AutoInclude {
		return c, err
	}
	return pickupRemote(frag, err)
}

// pickupRemote runs the search → pick → confirm → fetch+worktree sequence.
// noMatchErr is the local resolution error, returned when remote search
// comes up empty too.
func pickupRemote(frag string, noMatchErr error) (resolve.Candidate, error) {
	var zero resolve.Candidate
	cfg, err := loadConfig()
	if err != nil {
		return zero, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return zero, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return zero, err
	}

	repos := remote.CandidateRepos(cwd, cfg.ExpandedRoots(home))
	if len(repos) == 0 {
		return zero, noMatchErr
	}
	ui.Stepf("no local match — searching remote branches of %d repo(s)…", len(repos))
	matches := remote.Search(frag, repos)
	if len(matches) == 0 {
		return zero, fmt.Errorf("%v; remote branches had no match either", noMatchErr)
	}

	m, err := pick.OneOf(matches, remote.DisplayMatch)
	if err != nil {
		return zero, err
	}
	ui.Stepf("found %s on %s of %s", m.Branch, m.Remote, ui.Tilde(m.RepoDir))

	// If the branch is already checked out, there's nothing to fetch or
	// create — just enter it. No confirmation: this is non-destructive and
	// is exactly what the user asked for by naming the branch.
	if path, ok := remote.ExistingCheckout(m); ok {
		ui.Stepf("%s is already checked out — entering %s", m.Branch, ui.Tilde(path))
		return resolve.Candidate{Path: path, Branch: m.Branch, Source: resolve.SourceRemote}, nil
	}

	// Ask before any fetch touches the network or disk, unless the user
	// turned the prompt off with remote.confirm_before_fetch = false.
	if cfg.RemoteConfirmBeforeFetch() {
		yes, err := ui.ConfirmYN(fmt.Sprintf("fetch %s and create a worktree?", m.Branch), true)
		if err != nil {
			return zero, fmt.Errorf("remote pickup needs a terminal to confirm: %w", err)
		}
		if !yes {
			return zero, errors.New("cancelled")
		}
	} else {
		ui.Stepf("confirm_before_fetch is off — fetching without asking")
	}

	path, reused, err := remote.CreateWorktree(m, cfg.Remote.Dir)
	if err != nil {
		return zero, err
	}
	if reused {
		ui.Stepf("%s is already checked out — entering %s", m.Branch, ui.Tilde(path))
	} else {
		ui.Stepf("fetched %s — created worktree at %s", m.Branch, ui.Tilde(path))
	}
	return resolve.Candidate{Path: path, Branch: m.Branch, Source: resolve.SourceRemote}, nil
}

// offerInitWizard reports whether it ran the init wizard to completion in
// response to a no-match failure. Only fires interactively, only for a
// clean no-match, and only when no config file exists yet.
func offerInitWizard(resolveErr error) bool {
	var noMatch *resolve.NoMatchError
	if !errors.As(resolveErr, &noMatch) || !ui.HasTTY() {
		return false
	}
	cfgPath, err := config.Path()
	if err != nil {
		return false
	}
	if _, err := os.Stat(cfgPath); err == nil {
		return false // config exists; the miss is real
	}
	ui.Errorf("%v", resolveErr)
	yes, err := ui.ConfirmYN("No twig config exists yet — run the setup wizard now?", true)
	if err != nil || !yes {
		return false
	}
	if err := runInitWizard(false); err != nil {
		ui.Errorf("%v", err)
		return false
	}
	return true
}
