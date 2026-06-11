---
title: Quickstart
description: From zero to opening worktrees by fragment in two minutes.
---

## Open a worktree by fragment

From inside any repo that has worktrees:

```sh
twig matsumoto          # opens a new Ghostty window in the matching worktree
twig                    # no argument: fuzzy-pick among this repo's worktrees
twig -t matsumoto       # enter in the current tab instead of a new window
```

A fragment can be a full branch (`claude/competent-matsumoto-493452`), a
branch suffix (`matsumoto`), a directory slug, a hex suffix (`493452`), or a
literal path. Exact matches win without a picker; ties open a fuzzy finder.

## cd instead of opening windows

With [shell integration](/twig/getting-started/install/#shell-integration-recommended)
installed:

```sh
tw matsumoto            # cd's your current shell into the worktree
```

Or script it yourself — `twig cd` prints only the path:

```sh
cd "$(twig cd matsumoto)"
```

## Add a setup script

Commit a `twig.toml` at your repo root and it governs every worktree:

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

First entry into each worktree runs setup (after you approve the manifest
with `twig trust`); later entries skip it until a watched file or the
manifest changes. `twig foo --run` starts the `[run]` script after setup.

## See everything twig sees

```sh
twig list               # all worktrees: current repo + providers + roots
twig rm <fragment>      # remove a worktree (confirms; branch is kept)
```

## Point twig at your projects

Worktrees in `~/conductor/workspaces` and `<repo>/.claude/worktrees` are
found automatically. For anything else, declare roots in
`~/.config/twig/config.toml`:

```toml
roots = ["~/Code/myorg"]
```
