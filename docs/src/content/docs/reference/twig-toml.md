---
title: twig.toml
description: The per-repo manifest reference for setup, run, and opener overrides.
---

`twig.toml` is looked up in the worktree root first, with the main repo
root as fallback. Commit it once at the main root and it governs every
worktree. Nothing in it takes effect until the manifest is
[trusted](../guides/trust.md).

```toml
[setup]
# Runs on first entry into each worktree (and again when this file or a
# watched file changes). bash -eu -o pipefail, cwd = the worktree.
# Make it idempotent.
run = """
bun install
goose up
"""
# Re-run setup when any of these files change (content-hashed, relative
# to the worktree; creating or deleting one counts as a change).
watch = ["package.json", "bun.lockb", "go.mod"]

[run]
# Started in the foreground by `twig <frag> --run` / `twig enter --run`,
# after setup succeeds. Ctrl-C goes to this process; its exit code becomes
# twig's.
run = "bun dev"

# Optional: override which openers run when entering this repo's
# worktrees, and define repo-specific openers. Honored only when this
# manifest is trusted.
[open]
default = ["ghostty", "cursor", "dev-browser"]

[open.openers.dev-browser]
kind = "command"
command = "open http://localhost:5173"
```

## Script environment

| Variable | Value |
| --- | --- |
| `TWIG_WORKTREE` | the worktree directory |
| `TWIG_BRANCH` | checked-out branch (unset when detached) |
| `TWIG_REPO_ROOT` | the repo's main worktree |

## Semantics worth knowing

- A worktree-local `twig.toml` completely shadows the main-root one.
- Unknown keys warn (typo protection) but don't block.
- Setup success is recorded per worktree in its git dir; removing the
  worktree removes the state.
- Editing this file re-trips the trust gate and also re-runs setup after
  re-approval, since the manifest hash is part of the skip check.
