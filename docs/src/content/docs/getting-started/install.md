---
title: Install
description: Install twig with Homebrew or go install.
---

## Homebrew (macOS, Linux)

```sh
brew install josh-padnick/tap/twig
```

## go install

```sh
go install github.com/josh-padnick/twig@latest
```

## From a release

Download the tarball for your platform from the
[releases page](https://github.com/josh-padnick/twig/releases), unpack, and
put `twig` on your `PATH`.

## Shell integration (recommended)

The `tw` function cd's into a worktree in your current shell — something a
plain binary cannot do. Add to your shell config:

```sh
# ~/.zshrc
eval "$(twig shell-init zsh)"

# ~/.bashrc
eval "$(twig shell-init bash)"

# ~/.config/fish/config.fish
twig shell-init fish | source
```

Prefer a different name? `twig shell-init zsh --cmd j` emits `j()` instead.

## Verify

```sh
twig doctor
```

Doctor reports git, config, providers, openers, and the trust store, each
as an `ok`/`warn`/`fail` line.
