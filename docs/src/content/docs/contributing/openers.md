---
title: Contributing openers and providers
description: How to add support for a new terminal, editor, or worktree-creating tool.
---

twig is designed so that supporting a new tool is a small, well-bounded
PR. Often it's no code at all.

## First: do you need Go?

Anything launchable from a shell already works as a `command` opener in
config, with no contribution needed:

```toml
[open.openers.zed]
kind = "command"
command = "zed {{dir}}"
```

A built-in opener is worth it when the tool needs real logic: an
AppleScript dance, window-vs-tab modes, or quirks a one-line template
can't express. The Ghostty opener exists because
`open -na Ghostty --args ...` silently drops arguments when an instance is
already running.

## Adding a built-in opener

In [`internal/opener/`](https://github.com/josh-padnick/twig/tree/main/internal/opener):

1. Create `<name>_<os>.go` (build-tagged if platform-specific, with a
   stub for other platforms that returns a helpful error). Implement:

   ```go
   type Opener interface {
       Name() string
       CanInject() bool       // can it type a command into its terminal?
       CanCurrentTab() bool   // does it support -t?
       Available() error      // doctor diagnostics
       Open(t Target) error
   }
   ```

2. Register the kind in `FromConfig` in `opener.go`.
3. Never interpolate paths or commands into script source. Pass them as
   argv (see the Ghostty opener) and quote with `Quote`/`EntryLine`.
4. Add tests for the pure parts (script/line building), build-tagged to
   match.

## Adding a provider

A provider is one entry in
[`internal/resolve/providers.go`](https://github.com/josh-padnick/twig/blob/main/internal/resolve/providers.go):
a name plus a `Parents(home, roots)` function returning the directories
whose immediate children are worktree candidates. Nonexistent paths are
filtered automatically; candidates must contain a `.git` entry to match.

Include in the PR: the tool's actual on-disk layout (with a real example
path) and a fake-dir test in `scan_test.go`. The existing Conductor
symlink test shows the pattern.

## Running the suite

```sh
go vet ./... && go test ./...
```

Tests create real git repos in temp dirs; no network, no global git config.
