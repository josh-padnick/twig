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

## Out of scope (for now)

twig resolves a session by its **id**, not its thread name — names aren't
stable handles, and the id is what Codex puts in the shareable link.
