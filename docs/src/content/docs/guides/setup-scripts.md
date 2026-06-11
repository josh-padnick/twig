---
title: Setup and run scripts
description: Declare per-worktree setup in twig.toml; twig runs it once and skips it until something changes.
---

A `twig.toml` declares what it takes to make a fresh worktree usable:

```toml
[setup]
run = """
bun install
goose up
"""
watch = ["package.json", "bun.lockb", "go.mod"]

[run]
run = "bun dev"
```

twig looks for `twig.toml` in the worktree root first, then falls back to
the main repo root. Commit it once at the main root and it governs every
worktree.

## When setup runs

Every arrival path (new window, `tw` cd, `twig enter` by hand) converges on
the same logic:

- First entry into a worktree runs `[setup].run`, streaming output.
- Re-entry skips it, unless the manifest changed or any file listed in
  `watch` changed. Watch files are content-hashed, so creating or deleting
  one counts as a change.
- `--setup` forces a re-run; `--no-setup` skips it.
- A failing setup aborts with the script's exit status and records
  nothing, so the next entry tries again.

Success state lives in the worktree's git directory
(`.git/worktrees/<name>/twig-setup.json`), so it dies with the worktree
and never pollutes your working tree.

## The [run] script

`twig <fragment> --run` (or `twig enter --run`) starts `[run].run` in the
foreground after setup succeeds. A dev server, typically. Ctrl-C goes to
the child process, and its exit code becomes twig's.

## Script environment

Scripts execute with `bash -eu -o pipefail` in the worktree directory,
with three extra variables:

| Variable | Value |
| --- | --- |
| `TWIG_WORKTREE` | the worktree directory |
| `TWIG_BRANCH` | the checked-out branch (unset when detached) |
| `TWIG_REPO_ROOT` | the main worktree of the repo |

Write setup scripts to be idempotent: twig guarantees at-least-once, not
exactly-once.

None of this runs until you approve the manifest. See
[the trust model](../guides/trust.md).
