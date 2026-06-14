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

## Available openers

twig ships two: `ghostty`, and the general-purpose `command` that covers
most other tools without any code. (The `kind` field in config selects
which one an entry uses.)

### ghostty

The built-in `ghostty` opener (macOS) drives Ghostty via AppleScript. It
opens a window, or with `-t` enters your current tab, and types
`cd <dir> && clear && twig enter` into it, so setup output and `--run`
servers live in the terminal you land in. Tune it:

```toml
[open.openers.ghostty]
kind = "ghostty"
clear = true          # clear after cd (new windows only — see below)
delay_ms = 300        # window-creation race delay
reuse_window = true   # if you're already in Ghostty, stay in this window
```

With `reuse_window = true`, twig checks whether it's running inside
Ghostty (via the `TERM_PROGRAM` / `GHOSTTY_RESOURCES_DIR` it sets) and, if
so, enters the worktree in your current window instead of spawning a new
one — making the default open behave like a per-invocation `-t`. Run from
any other terminal, there's no Ghostty window to reuse, so it opens one as
usual. (An explicit `-t` always enters the current tab regardless.)

`clear` only fires when a **new window** is opened. Entering the current
tab — whether via `reuse_window` or `-t` — never clears, because that
scrollback is the session you're already working in; wiping it would
defeat the point of staying put. And if you're *already* in the target
worktree, twig types nothing at all — no redundant `cd` echoed into your
prompt.

Because reusing the current tab works by typing `cd <dir>` into your
shell, that command can only run once twig exits and your shell takes
over — so you'll see it appear at the next prompt. For a fully in-process
`cd` with no typing at all, use the [`tw` shell
function](shell-integration.md), which `cd`s first and then runs the
on-entry steps.

### command

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
(see [the trust model](../guides/trust.md)).

## Where setup runs

If any opener in the set is injection-capable, it carries `twig enter`
into its terminal. If the set is editors and browsers only, setup runs in
the invoking terminal before the openers launch, and `--run` is an error,
since a dev server needs a terminal to live in.
