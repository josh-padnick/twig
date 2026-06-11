// Command twig resolves git-worktree fragments (branch names, slugs, paths)
// to worktree directories and "enters" them: opening terminal windows or
// editors, running trust-gated setup scripts, and keeping agent-created
// worktrees (Claude Code, Conductor, cloud sessions) one short command away.
package main

import (
	"os"

	"github.com/josh-padnick/twig/internal/cli"
)

// Populated by goreleaser via -ldflags at release time.
var (
	version = "dev"
	commit  = "none"
)

func main() {
	os.Exit(cli.Execute(version, commit))
}
