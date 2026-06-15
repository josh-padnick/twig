// Shared resolve→pick glue used by every command that takes a fragment.
package cli

import (
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
// user config: expanded roots and the selected providers. With verbose set,
// the resolver narrates each step and scan location to stderr via ui.Stepf.
func newResolver(verbose bool) (*resolve.Resolver, error) {
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
	r := &resolve.Resolver{
		Cwd:       cwd,
		Home:      home,
		Roots:     cfg.ExpandedRoots(home),
		Providers: resolve.ByNames(cfg.Providers),
	}
	if verbose {
		r.Trace = func(msg string) { ui.Stepf("%s", msg) }
	}
	return r, nil
}

// resolveFragment resolves frag (possibly empty) to exactly one worktree,
// running the interactive picker when several candidates tie.
func resolveFragment(frag string, verbose bool) (resolve.Candidate, error) {
	r, err := newResolver(verbose)
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
	return pick.One(res.Candidates)
}
