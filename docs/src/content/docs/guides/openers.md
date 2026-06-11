---
title: Openers
description: Configure which tools open when you enter a worktree, from terminals to editors to browsers.
---

Entering a worktree can open more than a terminal. Openers are named ways
of opening a directory, and entering runs a configurable set of them.

```toml
# ~/.config/twig/config.toml
[open]
default = ["ghostty", "cursor"]

[open.openers.cursor]
kind = "command"
command = "cursor {{dir}}"
```

`twig foo` now opens a Ghostty window and Cursor in the resolved
worktree. Ad-hoc override: `twig foo --with ghostty`.

## The ghostty kind

The built-in `ghostty` opener (macOS) drives Ghostty via AppleScript. It
opens a window, or with `-t` enters your current tab, and types
`cd <dir> && clear && twig enter` into it, so setup output and `--run`
servers live in the terminal you land in. Tune it:

```toml
[open.openers.ghostty]
kind = "ghostty"
clear = true        # clear after cd
delay_ms = 300      # window-creation race delay
```

## The command kind

`command` runs any template through bash:

- `{{dir}}` is the worktree path, shell-quoted.
- `{{cmd}}` is the full entry command (`cd ... && twig enter ...`),
  shell-quoted. A template with `{{cmd}}` can host setup output and
  `--run`; twig calls these injection-capable.

```toml
[open.openers.wezterm]
kind = "command"
command = "wezterm cli spawn --new-window --cwd {{dir}}"

[open.openers.ghostty-linux]
kind = "command"
command = "ghostty -e bash -c {{cmd}}"
```

## Per-repo overrides

A trusted `twig.toml` can override the set and define its own openers:

```toml
[open]
default = ["ghostty", "cursor", "dev-browser"]

[open.openers.dev-browser]
kind = "command"
command = "open http://localhost:5173"
```

Precedence: the `--with` flag beats a trusted repo's `[open].default`,
which beats the global default. Untrusted manifests contribute nothing
(see [the trust model](https://joshpadnick.com/twig/guides/trust/)).

## Where setup runs

If any opener in the set is injection-capable, it carries `twig enter`
into its terminal. If the set is editors and browsers only, setup runs in
the invoking terminal before the openers launch, and `--run` is an error,
since a dev server needs a terminal to live in.
