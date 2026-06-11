---
title: Shell integration
description: The tw function — cd into worktrees in your current shell, with setup applied.
---

A binary cannot change its parent shell's directory, so twig ships a tiny
shell function instead (the direnv/zoxide trick):

```sh
# ~/.zshrc
eval "$(twig shell-init zsh)"
```

Now:

```sh
tw matsumoto      # cd's this shell into the worktree, then runs `twig enter`
tw                # inside a repo: fuzzy-pick a worktree, then cd
tw -r fix-login   # flags pass through to resolution (remote pickup here)
```

The function is three lines:

```sh
tw() {
  local dir
  dir="$(twig cd "$@")" || return $?
  cd "$dir" || return $?
  twig enter
}
```

`twig cd` prints **only the path** on stdout — prompts, pickers, and
errors use the terminal directly — which is what makes the command
substitution safe. `twig enter` then applies the same trust-gated setup
as every other arrival path.

Supported shells: `zsh`, `bash`, `fish`. Rename the function with
`--cmd`:

```sh
eval "$(twig shell-init zsh --cmd j)"
```

## Three ways in, one behavior

| You do | What happens |
| --- | --- |
| `twig foo` | new terminal window (plus other openers), setup inside it |
| `tw foo` | current shell cd's there, setup runs right here |
| `cd …` by hand, then `twig enter` | same setup logic, fully manual |

`setup.auto = false` in config makes all of them skip setup unless you
pass `--setup`.
