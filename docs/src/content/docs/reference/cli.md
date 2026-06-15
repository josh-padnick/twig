---
title: CLI commands
description: Every twig command and flag.
---

## twig [fragment]

Bare `twig <fragment>` resolves and opens the worktree with your
configured [opener set](../guides/openers.md). Bare `twig` inside a repo
opens a [picker](../guides/resolution.md#ranking-exact-beats-substring-branches-beat-directories)
over that repo's worktrees; outside one it shows help.

The fragment can also be a **GitHub pull request URL** —
`twig https://github.com/org/app/pull/140` (or any URL inside that PR, like
`…/pull/140/files`). twig maps it to the PR's head branch and picks it up
exactly as `twig -r <branch>` would, no `-r` needed. See
[remote pickup](../guides/remote-pickup.md#pull-request-urls).

The fragment can also be a **Codex thread link** —
`twig codex://threads/<id>` (or a bare thread id) opens the directory that
local Codex session ran in. See
[Codex local sessions](../guides/codex-sessions.md).

| Flag | Meaning |
| --- | --- |
| `-t, --tab` | enter in the current tab instead of a new window |
| `-r, --remote` | search remote branches when nothing matches locally |
| `-v, --verbose` | narrate each resolution step and scan location checked |
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
`cd "$(twig cd foo)"`. Supports `-r`, `-v`, and PR-URL fragments. With no
fragment, picks among the current repo's worktrees.

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
worktree. See [the trust model](../guides/trust.md).

## twig shell-init <zsh|bash|fish> [--cmd tw]

Emits the [`tw` shell function](../guides/shell-integration.md).

## twig doctor

Diagnoses git, config (including warnings), roots, providers, openers,
the trust store, and terminal availability.

## Exit codes

twig exits `0` on success and `1` on errors. The exception is `--run`,
where the run script's own exit code passes through.
