---
title: How resolution works
description: The exact order and ranking twig uses to turn a fragment into a worktree.
---

`twig <fragment>` resolves in three steps, stopping at the first hit:

1. **Literal path.** If the fragment is an existing directory (absolute,
   relative, or `~/...`), use it. No further matching.
2. **The current repo's worktrees.** Inside a repo, twig parses
   `git worktree list --porcelain`. Git knows every worktree of the repo
   regardless of where it lives on disk, which is what makes Conductor
   workspaces and Claude Code worktrees reachable from the main checkout.
3. **Filesystem scan.** Outside a repo, or when git had no match, twig
   scans [provider locations](../guides/providers.md) and configured
   `roots`, accepting only directories that contain a `.git` entry.

If nothing matches and remote pickup is enabled, twig can also
[search remote branches](../guides/remote-pickup.md). A fragment that is a
[GitHub pull request URL](../guides/remote-pickup.md#pull-request-urls)
skips local resolution entirely and goes straight to the PR's head branch.

## Seeing the search: `-v`

Pass `-v` (or `--verbose`) to watch twig resolve. It narrates each step and
prints every scan location as it checks it, so you can tell exactly which
roots and provider directories were searched and where the match came from:

```sh
twig -v ecstatic-euclid
# twig: resolving "ecstatic-euclid"
# twig: scanning 23 worktree location(s) under your roots and providers
# twig: checking ~/Code/josh-padnick
# twig: checking ~/Code/josh-padnick/twig/.claude/worktrees
# twig: matched 1 worktree(s) by scan
# twig: opening ~/Code/josh-padnick/twig/.claude/worktrees/ecstatic-euclid-8f14c5 with ghostty
```

`twig cd -v` traces to stderr while still printing only the resolved path on
stdout, so `cd "$(twig cd -v foo)"` keeps working.

## Ranking: exact beats substring, branches beat directories

Within a step, every candidate gets a tier, and only the best tier
survives:

| Tier | Meaning | Example for fragment `api` |
| --- | --- | --- |
| 1 | branch equals the fragment | branch `api` |
| 2 | directory basename equals the stripped fragment | dir `api/` |
| 3 | branch's last segment equals the stripped fragment | branch `feat/api` |
| 4 | fragment is a substring of the branch | branch `feat/api-v2` |
| 5 | stripped fragment is a substring of the basename | dir `my-api-thing/` |

So `api` resolves straight to `feat/api` even when `feat/api-v2` exists,
with no picker. Several candidates in the same tier open a picker: a headed,
inline list you drive with the arrow keys (or `j`/`k`), a number key to jump
straight to a row, `enter` to choose, and `q` to cancel. It renders in place
rather than taking over the screen, so the list stays in your scrollback.

The stripped fragment is everything after the last `/`: typing
`claude/foo` matches the same things `foo` does, so full branch names
always work. All matching is case-insensitive.

## Stale worktrees

A worktree record whose directory has been deleted is never matched.
If your fragment matches only stale records, twig says so and points you
at `git worktree prune` instead of silently resolving to something else.

## When a fragment collides with a subcommand

`twig list` always runs the subcommand. To open a worktree whose name
collides, use the explicit form: `twig open list`.
