// Shared resolve→pick glue used by every command that takes a fragment.
package cli

import (
	"os"

	"github.com/josh-padnick/twig/internal/pick"
	"github.com/josh-padnick/twig/internal/resolve"
)

// newResolver constructs the resolver for the invoking process. Roots and
// provider selection come from user config starting in M2; until then all
// builtin providers are active and no custom roots exist.
func newResolver() (*resolve.Resolver, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return &resolve.Resolver{Cwd: cwd, Home: home, Providers: resolve.Builtin}, nil
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
	return pick.One(res.Candidates)
}
