---
title: Codex local sessions
description: Jump to the directory a local Codex session ran in, straight from its thread link.
---

Codex comes in two shapes, and twig handles each differently:

- **Codex cloud** leaves its work as a branch on GitHub —
  [remote pickup](../guides/remote-pickup.md) (`twig -r`) fetches it.
- **Codex local** (Codex Desktop / CLI) doesn't create a worktree of its
  own. A session just *runs in* a directory you already have — a main
  checkout, a Conductor workspace, a `.claude` worktree — so there's no new
  layout to scan for. Instead twig reads the session's own log to learn
  which directory that was, and takes you there.

Each local session is a **thread**, addressable as
`codex://threads/<id>`. Hand twig that link (copy it from Codex) and it
jumps to where the session was working:

```sh
twig codex://threads/019eccfa-62f5-7733-8fc0-059abf2ea60b
# twig: Codex session 019eccfa… ran in ~/Code/fabricahq/app
# twig: opening ~/Code/fabricahq/app with ghostty
```

The bare thread id works too, and so does `tw` / `twig cd`:

```sh
twig 019eccfa-62f5-7733-8fc0-059abf2ea60b   # same thing
tw codex://threads/019eccfa-62f5-7733-8fc0-059abf2ea60b   # cd in place
```

## By title: `-s`

When you don't have the link handy, search by the session's title with
`-s`. Quote it, since titles have spaces:

```sh
twig -s "frame pr 142"
# twig: Codex session "Frame PR 142 review" ran in ~/Code/fabricahq/app
# twig: opening ~/Code/fabricahq/app with ghostty
```

Matching is case-insensitive and tiered: an exact title wins outright, then
a substring of the title, then a title containing every word you typed (in
any order). Several matches open the [picker](../guides/resolution.md),
most-recently-updated first, each showing where it lands:

```sh
twig -s frame
# 2 Codex sessions match "frame" — select one:
#   Frame PR 145 review  [~/Code/fabricahq/app]
#   Frame PR 142 review  [~/Code/fabricahq/app]
```

`-s` works on `twig`, `twig cd`, and `tw`. It only searches **local Codex**
session titles (from `~/.codex/session_index.jsonl`) — it doesn't touch
branch or directory resolution.

## How it works

1. A `codex://threads/<id>` URL is an explicit pointer, so it resolves
   straight away — no `-r`, no local-scan detour. A **bare** thread id is
   only tried as a last resort, after normal resolution finds nothing, so
   it can never shadow a real worktree.
2. twig finds the session's rollout log under
   `~/.codex/sessions/<yyyy>/<mm>/<dd>/rollout-<ts>-<id>.jsonl` (the id is
   in the filename, so no logs are read until the right one is found).
3. It reads the log's `session_meta` entry for the `cwd` the session ran
   in, and enters that directory through the usual open/cd flow — trust
   gate, [setup](../guides/setup-scripts.md), openers, the lot.

Because the destination is just a directory, it's whatever the session
used: often a main repo, sometimes a worktree another tool created. If
that directory has since been removed, twig says so rather than entering
nothing.

`-s` reads the same logs, after first matching your text against the
titles in `~/.codex/session_index.jsonl` to pick which session's `cwd` to
resolve.

## Scope

Title search covers **local Codex only**. Claude Code and Conductor
worktrees already have a slug or branch you can name as an ordinary
[fragment](../guides/resolution.md), and Codex *cloud* sessions are
branches reached with [`-r`](../guides/remote-pickup.md).
