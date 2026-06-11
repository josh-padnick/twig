# twig

Git worktrees, one short command away.

`twig <fragment>` resolves a fragment — a branch name, a branch suffix like
`claude/foo-bar` → `foo-bar`, a directory slug, or a literal path — to a git
worktree and enters it: opening your terminal (and any other tools you
configure) there, after running the worktree's trust-gated setup script.

It knows where agent tools keep their worktrees out of the box: Claude Code
desktop (`<repo>/.claude/worktrees/`), Conductor (`~/conductor/workspaces/`),
and cloud sessions that exist only as remote branches.

> **Status:** pre-v0.1, under active development. Docs, packaging, and a
> Homebrew tap are on the way.

## License

[MIT](LICENSE)
