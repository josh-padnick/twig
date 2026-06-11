---
title: config.toml
description: The complete user configuration reference.
---

twig reads `~/.config/twig/config.toml` (respecting `XDG_CONFIG_HOME`).
No config file is required: every setting has a default, and `twig init`
generates one interactively. Unknown keys and dangling references are
warnings (shown by every command and by `twig doctor`), never hard
failures.

```toml
# Extra places to scan for worktrees, besides providers and the current
# repo. Each root is scanned at <root>/*, <root>/*/*, and the Claude Code
# patterns. ~ expands.
roots = []

# Built-in tool layouts to scan; active when present on disk.
# Known: "conductor", "claude-code".
providers = ["conductor", "claude-code"]

[open]
# The opener set run on entry. "ghostty" works without a definition.
default = ["ghostty"]

[open.openers.ghostty]
kind = "ghostty"     # built-in kinds: ghostty | command
clear = true         # inject `clear` after cd
delay_ms = 300       # AppleScript delay before typing into the new window

# Define any other tool as a command template:
# [open.openers.cursor]
# kind = "command"
# command = "cursor {{dir}}"          # {{dir}} = quoted worktree path
#                                     # {{cmd}} = quoted entry command
#                                     #   (makes a terminal injection-capable)

[remote]
auto = false                          # search remotes on every local miss
dir = ".claude/worktrees/{{slug}}"    # worktree location for fetched
                                      # branches, relative to the main
                                      # worktree; {{branch}} = name with /→-

[setup]
auto = true          # run setup logic on entry; false = only with --setup
```

## Related files

| Path | Contents |
| --- | --- |
| `~/.config/twig/config.toml` | this file |
| `~/.local/share/twig/trust/` | [trust approvals](../guides/trust.md) |
| `<gitdir>/twig-setup.json` | per-worktree [setup state](../guides/setup-scripts.md) |
