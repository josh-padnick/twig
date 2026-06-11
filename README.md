# twig

Git worktrees, one short command away.

```sh
twig matsumoto      # finds claude/competent-matsumoto-493452, opens Ghostty there
tw matsumoto        # …or cd your current shell into it
```

`twig <fragment>` resolves a fragment — a branch name, a branch suffix
(`claude/foo-bar` → `foo-bar`), a directory slug, a hex suffix, or a literal
path — to a git worktree and enters it: opening your terminal (and any other
tools you configure) there, after running the repo's trust-gated setup
script.

**[Full documentation → joshpadnick.com/twig](https://joshpadnick.com/twig/)**

## Why

Agent tools multiply worktrees: Claude Code desktop keeps them inside the
repo under `.claude/worktrees/`, Conductor under `~/conductor/workspaces/`,
and cloud sessions leave branches that exist only on GitHub. twig knows all
of these out of the box — type any memorable fragment and land in the right
directory with dependencies installed.

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
twig <fragment>        # open the matching worktree (new Ghostty window)
twig                   # inside a repo: fuzzy-pick among its worktrees
twig -t <fragment>     # enter in the current tab instead
twig -r <fragment>     # also search remote branches (cloud sessions)
tw <fragment>          # cd in place (via shell-init)
twig list              # everything twig can see, with branch + status
twig rm <fragment>     # remove a worktree (confirms; keeps the branch)
twig doctor            # diagnose config, providers, openers, trust store
```

Commit a `twig.toml` at your repo root and every worktree gets setup on
first entry — skipped on re-entry until the manifest or a watched file
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

Running repo-declared scripts on entry is an attack vector — checking out a
PR branch must never execute attacker code. twig copies direnv's answer:
the first time it sees a given `twig.toml`, it refuses, shows you the file,
and requires approval (`twig trust` or an interactive y/N). Approvals are
keyed by path **and** content hash, so any edit or move re-trips the gate.
An untrusted manifest contributes nothing — neither scripts nor its
`[open]` opener overrides.
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

The built-in `ghostty` opener drives Ghostty via AppleScript (new window,
or current tab with `-t`); the `command` kind covers everything else —
editors, browsers, other terminals. Trusted repos can override the set in
their own `twig.toml`.
[Details.](https://joshpadnick.com/twig/guides/openers/)

## Future work

- Clone-from-GitHub when a remote branch's repo isn't on disk at all
  (today, remote pickup needs the repo checked out somewhere under your
  roots).

## Contributing

New openers and providers are small, well-bounded PRs — see the
[contributing guide](https://joshpadnick.com/twig/contributing/openers/).
Run the suite with `go vet ./... && go test ./...` (hermetic; real git in
temp dirs, no network).

## License

[MIT](LICENSE)
