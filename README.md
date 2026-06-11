# twig

Git worktrees, one short command away.

```sh
twig gould          # finds claude/xenodochial-gould-7bf514, opens Ghostty there
tw gould            # ...or cd your current shell into it
```

twig is a small CLI that makes it easy to work with the many git worktrees
created by AI coding tools like Claude Code, Codex, and Conductor.

The idea: while you're in, say, the Claude Code desktop UI, you glance at
the worktree or branch name (`claude/xenodochial-gould-7bf514`) and note
some memorable part of it (`gould`). One command then finds where that
directory actually lives on your machine, opens a terminal there or cd's
your current shell, and runs the repo's optional setup and run commands.
twig can also open other tools in the same directory: Cursor, VS Code, a
browser tab pointed at your dev server.

**[Full documentation → joshpadnick.com/twig](https://joshpadnick.com/twig/)**

## Where worktrees hide

Every tool has its own ideas about where work lives. Claude Code desktop
checks sessions out inside the repo under `.claude/worktrees/`, with
generated names nobody types twice. Conductor keeps workspaces at
`~/conductor/workspaces/<project>/<name>`. Cloud sessions leave a branch
on GitHub and no local directory at all.

twig knows all of these out of the box. The first two are scanned
automatically. For the third, `twig -r <fragment>` searches the remotes of
repos you already have, fetches the branch, and builds the worktree.

## Install

```sh
brew install --cask josh-padnick/tap/twig
# or
go install github.com/josh-padnick/twig@latest
```

Then add the shell function for in-place cd (recommended):

```sh
eval "$(twig shell-init zsh)"     # ~/.zshrc; bash and fish also supported
```

## Quickstart

```sh
twig init              # interactive first-run setup (roots, openers, shell)
twig <fragment>        # open the matching worktree (new Ghostty window)
twig                   # inside a repo: fuzzy-pick among its worktrees
twig -t <fragment>     # enter in the current tab instead
twig -r <fragment>     # also search remote branches (cloud sessions)
tw <fragment>          # cd in place (via shell-init)
twig list              # everything twig can see, with branch + status
twig rm <fragment>     # remove a worktree (confirms; keeps the branch)
twig doctor            # diagnose config, providers, openers, trust store
```

Commit a `twig.toml` at your repo root and every worktree runs setup on
first entry. Re-entries skip it until the manifest or a watched file
changes:

```toml
[setup]
run = """
bun install
goose up
"""
watch = ["package.json", "bun.lockb", "go.mod"]

[run]
run = "bun dev"        # started by `twig <fragment> --run`
```

## The trust model

Running repo-declared scripts on entry is an attack vector: checking out a
PR branch must never execute attacker code. twig copies direnv's answer.
The first time it sees a given `twig.toml`, it refuses, shows you the
file, and asks for approval (`twig trust`, or an interactive y/N).
Approvals are keyed by path and content hash, so any edit or move re-trips
the gate. An untrusted manifest contributes nothing, not even its `[open]`
opener overrides.
[Details.](https://joshpadnick.com/twig/guides/trust/)

## Openers

Entering a worktree can open more than a terminal:

```toml
# ~/.config/twig/config.toml
[open]
default = ["ghostty", "cursor"]

[open.openers.cursor]
kind = "command"
command = "cursor {{dir}}"
```

The built-in `ghostty` opener drives Ghostty via AppleScript, opening a
new window or, with `-t`, entering the current tab. The `command` kind
covers everything else: editors, browsers, other terminals. Trusted repos
can override the set in their own `twig.toml`.
[Details.](https://joshpadnick.com/twig/guides/openers/)

## Future work

- Clone-from-GitHub when a remote branch's repo isn't on disk at all
  (today, remote pickup needs the repo checked out somewhere under your
  roots).

## Contributing

New openers and providers are small, well-bounded PRs; see the
[contributing guide](https://joshpadnick.com/twig/contributing/openers/).
Run the suite with `go vet ./... && go test ./...`. Tests create real git
repos in temp dirs, with no network and no global git config.

## License

[MIT](LICENSE)
