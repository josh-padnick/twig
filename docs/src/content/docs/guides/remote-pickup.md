---
title: Remote branch pickup
description: Resolve branches that only exist on GitHub by fetching them into a local worktree.
---

Cloud sessions (Claude Code web, Codex) leave their work as a branch on
GitHub that no local checkout knows about. Remote pickup bridges that gap:

```sh
twig -r fix-login
# twig: no local match — searching remote branches of 3 repo(s)…
# twig: found claude/fix-login-9a8b7c on origin of ~/Code/myorg/app
# fetch claude/fix-login-9a8b7c and create a worktree? [Y/n]
# twig: fetched claude/fix-login-9a8b7c — created worktree at ~/Code/myorg/app/.claude/worktrees/fix-login-9a8b7c
```

## How it works

1. Pickup runs only after local resolution finds nothing, and only with
   `-r` or `auto = true`. Typos shouldn't hit the network.
2. twig queries `git ls-remote --heads` against the remotes of repos
   already on disk: the current repo first, then repos directly under
   your `roots`. There's no GitHub API and no tokens involved, so it
   works with any git host.
3. Matches use the same [branch tiers](../guides/resolution.md) as local
   resolution; several equal matches open the picker.
4. After you confirm, twig fetches the branch, creates a tracking worktree
   at the configured location, and continues the normal open/cd flow,
   [setup](../guides/setup-scripts.md) included.

If the branch is **already checked out** in another worktree — say a
cloud session you picked up earlier from a different directory — twig
doesn't fail with git's `already used by worktree` error. It detects the
existing worktree, lands you there, and runs the usual open/setup/run
flow against it:

```sh
twig -r fix-login
# twig: no local match — searching remote branches of 3 repo(s)…
# twig: found claude/fix-login-9a8b7c on origin of ~/Code/myorg/app
# fetch claude/fix-login-9a8b7c and create a worktree? [Y/n]
# twig: claude/fix-login-9a8b7c is already checked out — entering ~/Code/myorg/app/.claude/worktrees/cranky-benz-b3b706
```

## Configuration

```toml
# ~/.config/twig/config.toml
[remote]
auto = false                          # set true to search on every local miss
confirm_before_fetch = true           # set false to fetch + create the
                                      # worktree without the y/N prompt
dir = ".claude/worktrees/{{slug}}"    # where fetched branches get worktrees,
                                      # relative to the repo's main worktree
```

`{{slug}}` is the branch's last path segment; `{{branch}}` is the full
name with `/` replaced by `-`.

The two flags are independent: `auto` controls *when* pickup runs (every
local miss vs. only with `-r`), and `confirm_before_fetch` controls
*whether it asks first*. For a fully hands-off `twig <fragment>` that
picks up cloud branches with no flag and no prompt, set both:

```toml
[remote]
auto = true
confirm_before_fetch = false
```

(An already-checked-out branch is entered directly regardless of either
flag — there's nothing to fetch, so there's nothing to confirm.)

## Out of scope (for now)

twig will not clone a repo that isn't on disk at all; that requires
host-API repo discovery and auth. If the repo exists locally anywhere
under your roots, pickup finds its branches.
