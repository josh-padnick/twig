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
   `-r` or `auto_include = true`. Typos shouldn't hit the network.
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

## Pull request URLs

When you have a PR open in the browser, you don't need to hunt for its
branch name. Hand twig the URL and it picks up the head branch for you:

```sh
twig https://github.com/fabricahq/app/pull/140
# twig: resolving PR #140 of fabricahq/app across 3 local repo(s)…
# twig: PR #140 is claude/per-request-identity on origin of ~/Code/fabricahq/app
# fetch claude/per-request-identity and create a worktree? [Y/n]
# twig: fetched claude/per-request-identity — created worktree at ~/Code/fabricahq/app/.claude/worktrees/per-request-identity
```

The URL doesn't have to point at the PR root — any URL inside the PR works,
so you can copy it straight from the address bar:
`…/pull/140/files`, `…/pull/140/changes`, `…/pull/140/commits`, and so on.

This is just remote pickup with the branch handed to it directly, so
everything above applies — the confirm prompt, the already-checked-out
shortcut, `confirm_before_fetch`, the worktree `dir` template. The one
difference is that a PR URL is an explicit request, so it works **without**
`-r` and ignores `auto_include`. `tw <url>` and `twig cd <url>` resolve a PR
URL too.

### How twig finds the branch

twig maps the PR back to its branch using git alone — no GitHub API, no
token:

1. It finds the local checkout whose remote points at the PR's repo
   (`fabricahq/app`), the same set of repos searched by fragment pickup.
2. It asks that remote where the PR head sits with
   `git ls-remote <remote> refs/pull/140/head` — a ref GitHub maintains for
   every PR.
3. It maps that commit back to the branch of the same name on the remote,
   then fetches and creates the worktree as usual.

Because the branch is recovered from the remote's own branch list, the head
branch must live in **the same repository** and still exist. A PR opened
from a fork, or one whose branch was deleted after merge, can't be resolved
this way — fall back to naming the branch with `twig -r <branch>` if it's
still around. And as with all remote pickup, the repository has to be on
disk somewhere under your roots.

## Configuration

```toml
# ~/.config/twig/config.toml
[remote]
auto_include = false                  # include remotes in the search on every local miss
confirm_before_fetch = true           # set false to fetch + create the
                                      # worktree without the y/N prompt
dir = ".claude/worktrees/{{slug}}"    # where fetched branches get worktrees,
                                      # relative to the repo's main worktree
```

`{{slug}}` is the branch's last path segment; `{{branch}}` is the full
name with `/` replaced by `-`.

For a fully hands-off `twig <fragment>` that picks up cloud branches with
no flag and no prompt, set both:

```toml
[remote]
auto_include = true
confirm_before_fetch = false
```

(An already-checked-out branch is entered directly regardless of either
flag — there's nothing to fetch, so there's nothing to confirm.)

## Out of scope (for now)

twig will not clone a repo that isn't on disk at all; that requires
host-API repo discovery and auth. If the repo exists locally anywhere
under your roots, pickup finds its branches.
