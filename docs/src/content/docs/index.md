---
title: Why twig
description: What twig is, and the problem it exists to solve.
---

twig is a small CLI that makes it easy to work with the many git worktrees
created by AI coding tools like Claude Code, Codex, and Conductor.

The main idea: while you're in, say, the Claude Code desktop UI, you glance
at the worktree or branch name (`claude/xenodochial-gould-7bf514`) and note
some memorable part of it (`gould`). One command then does the rest. It
finds where that directory actually lives on your machine, opens a terminal
there (or cd's your current shell), and runs the repo's optional setup and
run commands:

```sh
twig gould
# → opens Ghostty in ~/Code/fabricahq/app/.claude/worktrees/xenodochial-gould-7bf514,
#   with the repo's setup script already run
```

twig can also open other tools in that working directory at the same time:
Cursor, VS Code, a browser tab pointed at your dev server, whatever set of
openers you configure.

## The problem

Git worktrees used to be a niche power tool. Agent tooling changed that.
Worktrees now multiply on their own, and every tool has its own ideas
about where they belong:

- Claude Code desktop checks out each session inside the repo, at
  `<repo>/.claude/worktrees/<generated-slug>`, with names like
  `xenodochial-gould-7bf514` that nobody types twice.
- Conductor keeps workspaces far from the repo, at
  `~/conductor/workspaces/<project>/<name>`, and leaves symlinks behind
  when they're renamed.
- Cloud sessions (Claude Code web, Codex) don't create a local directory
  at all: their work is a branch on GitHub that no checkout on your
  machine has fetched.

So "jump to the worktree where the agent fixed login" becomes
archaeology: `git worktree list`, squint, copy a 70-character path, `cd`.
Multiply by every session, every day.

Arriving isn't enough, either. A fresh worktree is a fresh checkout: no
`node_modules`, no migrations, no env. Each one quietly costs a
`bun install && goose up` that you either remember, or discover halfway
into debugging something that was never broken.

## What twig does about it

One short command works anywhere. `twig <fragment>` checks the current
repo's worktree list first (git already knows every worktree, wherever it
lives on disk), then scans known tool locations and your configured roots.
Exact matches beat substrings, branches beat directory names, and real
ambiguity gets a fuzzy picker. Typing `gould` is enough.
[How resolution works.](/twig/guides/resolution/)

Setup runs itself, but only when needed. A `twig.toml` committed at the
repo root says what a worktree needs (`bun install`, migrations,
whatever). twig runs it on first entry, then skips it until the manifest
or a watched lockfile changes. Every entry path converges on the same
logic, whether you opened a window, cd'd with `tw`, or ran `twig enter`
by hand. [Setup and run scripts.](/twig/guides/setup-scripts/)

Repo-declared code requires consent. Running scripts on "enter" is an
attack vector: checking out a PR branch must never execute attacker code.
twig copies direnv here. Nothing from a `twig.toml` runs until you've
seen it and approved that exact content, and any edit re-trips the gate.
[The trust model.](/twig/guides/trust/)

Entering can open your whole working context, not just a terminal.
Openers run as a set, so one command can open Ghostty, Cursor, and a
browser tab on your dev server, with per-repo overrides.
[Openers.](/twig/guides/openers/)

Branches that only exist remotely still resolve. With `-r`, a local miss
falls through to `git ls-remote` across repos you already have; twig
fetches the branch and builds the worktree for you.
[Remote pickup.](/twig/guides/remote-pickup/)

## What twig deliberately isn't

- A git replacement. Branch management stays git's job. Even `twig rm`
  keeps the branch and tells you the `git branch -d` to run.
- A daemon or shell hijack. It's a single static binary plus an optional
  three-line shell function; nothing runs in the background.
- Configuration-hungry. Conductor and Claude Code layouts work with zero
  config, and `twig init` generates the rest in about thirty seconds.

## Where to start

[Install twig](/twig/getting-started/install/), run `twig init`, then try
the [quickstart](/twig/getting-started/quickstart/).
