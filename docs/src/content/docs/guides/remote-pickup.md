---
title: Remote branch pickup
description: Resolve branches that only exist on GitHub by fetching them into a local worktree.
---

Cloud sessions (Claude Code web, Codex) leave their work as a branch on
GitHub that no local checkout knows about. Remote pickup bridges that gap:

```sh
twig -r fix-login
# twig: no local match — searching remote branches of 3 repo(s)…
# branch claude/fix-login-9a8b7c found on origin of ~/Code/myorg/app — fetch it and create a worktree? [Y/n]
# twig: created worktree ~/Code/myorg/app/.claude/worktrees/fix-login-9a8b7c (branch claude/fix-login-9a8b7c)
```

## How it works

1. Pickup runs only after local resolution finds nothing, and only with
   `-r` or `auto = true`. Typos shouldn't hit the network.
2. twig queries `git ls-remote --heads` against the remotes of repos
   already on disk: the current repo first, then repos directly under
   your `roots`. There's no GitHub API and no tokens involved, so it
   works with any git host.
3. Matches use the same [branch tiers](https://joshpadnick.com/twig/guides/resolution/) as local
   resolution; several equal matches open the picker.
4. After you confirm, twig fetches the branch, creates a tracking worktree
   at the configured location, and continues the normal open/cd flow,
   [setup](https://joshpadnick.com/twig/guides/setup-scripts/) included.

## Configuration

```toml
# ~/.config/twig/config.toml
[remote]
auto = false                          # set true to search on every local miss
dir = ".claude/worktrees/{{slug}}"    # where fetched branches get worktrees,
                                      # relative to the repo's main worktree
```

`{{slug}}` is the branch's last path segment; `{{branch}}` is the full
name with `/` replaced by `-`.

## Out of scope (for now)

twig will not clone a repo that isn't on disk at all; that requires
host-API repo discovery and auth. If the repo exists locally anywhere
under your roots, pickup finds its branches.
