---
title: CLI commands
description: Every twig command and flag.
---

## twig [fragment]

Bare `twig <fragment>` resolves and opens the worktree with your
configured [opener set](https://joshpadnick.com/twig/guides/openers/). Bare `twig` inside a repo
fuzzy-picks among that repo's worktrees; outside one it shows help.

| Flag | Meaning |
| --- | --- |
| `-t, --tab` | enter in the current tab instead of a new window |
| `-r, --remote` | search remote branches when nothing matches locally |
| `--run` | run the `[run]` script after setup succeeds |
| `--setup` | force the setup script to re-run |
| `--no-setup` | skip setup entirely |
| `--with a,b` | openers to run, overriding the configured set |

A fragment that collides with a subcommand name: use `twig open <fragment>`.

## twig open [fragment]

Identical to the bare form, as an explicit subcommand.

## twig init [--force]

The first-run wizard. It discovers likely roots (ranked by repo count and
Claude-worktree signal), detects installed editors to offer as openers,
writes a commented config, and offers to add the `tw` function to your
shell config. Refuses to overwrite an existing config without `--force`.
An interactive `twig <fragment>` miss with no config file offers this
wizard automatically.

## twig cd [fragment]

Prints the resolved worktree path on stdout and nothing else, for
`cd "$(twig cd foo)"`. Supports `-r`. With no fragment, picks among the
current repo's worktrees.

## twig enter [dir] [--run] [--setup] [--no-setup]

Runs the on-entry steps in the current terminal for `dir` (default: cwd):
trust gate, setup-if-needed, optional `[run]`. This is the primitive that
`twig open` injects into new windows and the `tw` function calls after
cd.

## twig list

Every worktree twig can see (current repo, providers, roots) as
`PATH  BRANCH  STATUS`, where status is `clean`, `dirty`, `stale` (record
without directory), or `broken` (git can't read it).

## twig rm <fragment> [--force]

Confirms, then `git worktree remove` + `git worktree prune`. The branch is
kept. `--force` skips confirmation and removes dirty trees. Refuses to
remove the main worktree.

## twig trust [fragment] [--show|--list|--revoke]

Approve (or inspect/revoke approval of) the `twig.toml` governing a
worktree. See [the trust model](https://joshpadnick.com/twig/guides/trust/).

## twig shell-init <zsh|bash|fish> [--cmd tw]

Emits the [`tw` shell function](https://joshpadnick.com/twig/guides/shell-integration/).

## twig doctor

Diagnoses git, config (including warnings), roots, providers, openers,
the trust store, and terminal availability.

## Exit codes

twig exits `0` on success and `1` on errors. The exception is `--run`,
where the run script's own exit code passes through.
