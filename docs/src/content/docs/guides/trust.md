---
title: The trust model
description: Why twig refuses to run scripts you haven't approved, and how approval works.
---

Executing scripts declared in a repo on entry is an attack vector: check
out a pull-request branch, type `tw pr-branch`, and an attacker-supplied
`twig.toml` would run arbitrary code. twig copies
[direnv](https://direnv.net/)'s answer.

## How it works

- The first time twig sees a given `twig.toml`, it refuses to use it,
  prints the full contents, and asks for approval. Interactively that's a
  y/N prompt; otherwise you get instructions to run `twig trust`.
- Approval is recorded as `sha256(absolute path + file content)`. Any edit
  to the file, or moving it, invalidates the approval and the gate trips
  again.
- The gate covers the whole manifest. An untrusted `twig.toml` contributes
  neither scripts nor its `[open]` section, because repo-defined openers
  are arbitrary commands too.

## Managing approvals

```sh
twig trust                  # approve the manifest governing the current dir
twig trust <fragment>       # approve another worktree's manifest
twig trust --show           # print it without approving
twig trust --list           # every approval, newest first
twig trust --revoke         # withdraw approval
```

Approvals live in `~/.local/share/twig/trust/` (XDG data dir), one JSON
file per approved content version, written atomically.

## What this protects (and what it doesn't)

The gate ensures nothing executes that you haven't read and approved at
that exact content. It does not sandbox approved scripts: approval means
you trust the script with everything your user account can do. Read before
you approve.
