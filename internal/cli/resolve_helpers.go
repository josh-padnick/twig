// Shared resolve→pick glue used by every command that takes a fragment.
package cli

import (
	"fmt"
	"os"

	"github.com/josh-padnick/twig/internal/config"
	"github.com/josh-padnick/twig/internal/pick"
	"github.com/josh-padnick/twig/internal/resolve"
	"github.com/josh-padnick/twig/internal/ui"
)

// loadConfig reads the user config, surfacing warnings on stderr so typos
// in config.toml are visible instead of silently doing nothing.
func loadConfig() (config.Config, error) {
	cfg, warnings, err := config.Load()
	for _, w := range warnings {
		ui.Warnf("%s", w)
	}
	return cfg, err
}

// newResolver constructs the resolver for the invoking process from the
// user config: expanded roots and the selected providers.
func newResolver() (*resolve.Resolver, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	return &resolve.Resolver{
		Cwd:       cwd,
		Home:      home,
		Roots:     cfg.ExpandedRoots(home),
		Providers: resolve.ByNames(cfg.Providers),
	}, nil
}

// resolveFragment resolves frag (possibly empty) to exactly one worktree,
// running the interactive picker when several candidates tie.
func resolveFragment(frag string) (resolve.Candidate, error) {
	r, err := newResolver()
	if err != nil {
		return resolve.Candidate{}, err
	}
	res, err := r.Resolve(frag)
	if err != nil {
		return resolve.Candidate{}, err
	}
	if res.Chosen != nil {
		return *res.Chosen, nil
	}
	return pick.One(res.Candidates, pickHeader(frag, len(res.Candidates)))
}

// pickHeader is the prompt shown above the worktree picker. With no fragment
// (bare `twig` inside a repo) the list is simply this repo's worktrees;
// otherwise it's the several worktrees that tied for the fragment.
func pickHeader(frag string, n int) string {
	if frag == "" {
		return fmt.Sprintf("%d worktrees in this repo — select one to enter:", n)
	}
	return fmt.Sprintf("%d worktrees match %q — select one:", n, frag)
}
