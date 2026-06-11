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
   scans [provider locations](https://joshpadnick.com/twig/guides/providers/) and configured
   `roots`, accepting only directories that contain a `.git` entry.

If nothing matches and remote pickup is enabled, twig can also
[search remote branches](https://joshpadnick.com/twig/guides/remote-pickup/).

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
with no picker. Several candidates in the same tier open the fuzzy finder.

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
