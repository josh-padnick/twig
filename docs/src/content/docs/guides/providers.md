---
title: Providers
description: Zero-config awareness of where agent tools keep their worktrees.
---

Providers are built-in scan locations for tools that create worktrees in
known places. They activate automatically when their directories exist on
disk, with no configuration needed.

| Provider | What it scans | Layout |
| --- | --- | --- |
| `conductor` | `~/conductor/workspaces/*/*` | [Conductor](https://conductor.build) workspaces |
| `claude-code` | `<root>/.claude/worktrees/*` and `<root>/*/.claude/worktrees/*` | Claude Code desktop in-repo worktrees, applied under your configured roots |

When you're inside a repo, its own worktrees are found via git first;
providers matter when resolving from anywhere else. One Conductor quirk
worth knowing: it leaves a renamed workspace behind as a symlink to the
new name. twig follows those and collapses duplicates to the canonical
path.

## Codex and cloud sessions

Codex keeps no local worktree directory. Its cloud sessions, like Claude
Code web sessions, exist as branches on GitHub, so they're covered by
[remote branch pickup](https://joshpadnick.com/twig/guides/remote-pickup/) rather than a
provider.

## Trimming the set

```toml
# ~/.config/twig/config.toml
providers = ["conductor"]    # disable claude-code scanning
providers = []               # disable all providers
```

Unknown names produce a warning, and `twig doctor` shows each provider's
live scan locations.

## Adding a provider

A provider is a name plus a function returning scan-parent directories:
one small entry in `internal/resolve/providers.go`. PRs for new tool
layouts are welcome; see
[Contributing openers and providers](https://joshpadnick.com/twig/contributing/openers/).
