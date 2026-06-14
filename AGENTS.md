# AGENTS.md

Guidance for AI agents (and humans) working in this repo. Keep it short and
high-signal; add to it when a change teaches a non-obvious lesson.

## What twig is

A Go CLI that resolves a fragment — a branch name, branch suffix, directory
slug, or literal path — to a git worktree and "enters" it: opening terminals
or editors, running trust-gated setup, optionally starting a `[run]` script.
Entry point is `main.go`; all logic lives under `internal/`.

## Build, test, format

- `go build ./...` — compile.
- `go test ./...` — full suite. CI runs this on **ubuntu + macos**; many tests
  shell out to a real `git`, so keep `git` on PATH.
- `go vet ./...` — CI gate (see `.github/workflows/ci.yml`).
- `gofmt -l internal/` must print nothing — run it before you commit.
- Docs (`docs/`, Astro + Starlight): `cd docs && bun run build`.

## Package map

- `cli/` — cobra commands; thin, delegate to the packages below.
- `config/` — user config (`~/.config/twig/config.toml`).
- `resolve/` — fragment → worktree resolution; pure, never prompts.
- `remote/` — remote-branch pickup for cloud sessions (fetch + worktree).
- `gitx/` — thin wrapper over the system `git` binary.
- `setup/` · `trust/` · `opener/` · `pick/` · `ui/` · `initwiz/` · `shell/` · `xdg/`.

## Gotcha: a `config.toml` option lives in FOUR places — keep them in sync

Adding or changing a key under `~/.config/twig/config.toml` isn't done until
all of these agree. The wizard template is the easy, silent miss:

1. **Struct + default** — `internal/config/config.go`: the field's `toml:"…"`
   tag, and the default in `Default()` when the zero value isn't the default.
   If the value is constrained (enum-like), also add a check in `validate()`.
2. **`twig init` template** — `internal/initwiz/generate.go` writes the starter
   config; its commented hints should mention the new option.
3. **Reference** — `docs/src/content/docs/reference/config.md`.
4. **Guide** — the feature's guide, e.g. `docs/.../guides/remote-pickup.md`
   for `[remote]` keys.

`internal/config/config_test.go` decodes a full TOML and asserts on the parsed
fields — extend it when you add a key.

## Convention: all user-facing output goes through `internal/ui`

Everything twig says goes to **stderr** (or `/dev/tty` for prompts) so that
`twig cd` can keep **stdout** clean for command substitution
(`cd "$(twig cd foo)"`). Pick the right helper:

- `ui.Stepf` — twig narrating one step of its own work (dimmed `twig:`
  prefix). Use this for operational steps, **not** `Infof`.
- `ui.Infof` — raw, unprefixed text (wizard prose, listings).
- `ui.Warnf` / `ui.Errorf` — highlighted severity prefixes.
- `ui.Tilde(path)` — shorten a home-prefixed path for **display only**. Never
  feed its result anywhere that needs a real path.
