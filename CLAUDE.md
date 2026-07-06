# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`wt` — an interactive Git worktree manager CLI built in Go with a Bubble Tea TUI. Create, switch, list, and remove worktrees with fuzzy matching and an interactive picker.

## Commands

```bash
go build -o wt .          # Build the binary
go run . <args>           # Run without installing (e.g. go run . list)
go install .              # Install to $GOPATH/bin
go vet ./...              # Vet
gofmt -l .                # List unformatted files (gofmt -w . to fix)
```

There are no tests in the repo. `go build ./...` and `go vet ./...` are the only automated checks.

## The stdout/stderr contract (most important thing to know)

`wt` cannot change the parent shell's directory itself — a child process can't `cd` its parent. The workaround: `wt` **prints the target worktree path to stdout and nothing else**, and a shell wrapper function (from `wt init-shell`, in `cmd/initshell.go`) captures that stdout and runs `cd` on it.

Consequences that any change must respect:

- **stdout is reserved for the path to cd into.** Commands that switch directories (`add`, `use`, root TUI) print `fmt.Print(path)` to stdout with no trailing newline decoration and nothing else.
- **All UI goes to stderr.** Both TUIs launch with `tea.NewProgram(..., tea.WithOutput(os.Stderr))`. Prompts, status messages, and errors use `fmt.Fprint(os.Stderr, ...)`. `list` is the exception — it prints a table to stdout because it's not meant to be `cd`'d.
- Break this and the shell wrapper will try to `cd` into log text. The wrapper only `cd`s when stdout is a directory (`[ -d "$result" ]`), otherwise it echoes — so stray stdout output surfaces as noise, not a cd.

## Architecture

Three layers, each a package:

- **`cmd/`** — Cobra commands, one file per subcommand (`add`, `use`, `list`, `remove`, `upgrade`, `initshell`, `syncignored`), wired in `root.go`. Commands are thin: parse args, call `internal/git`, print the path or delegate to the TUI. Bare `wt` runs the interactive TUI.
- **`internal/git/`** — all Git interaction, done by shelling out to the `git` binary via `os/exec` (no git library). `worktree.go` parses `git worktree list --porcelain`; `branch.go` lists/sorts branches and detects the default branch; `ignored.go` lists gitignored entries (`git ls-files --others --ignored --exclude-standard --directory`) and copies them between worktrees. This is the only package that touches git state.
- **`internal/tui/`** — Bubble Tea models. `interactive.go` is a single model with a `mode` enum (list / add / confirmDelete / branchPick) driving one Update loop; `branchpicker.go` is a standalone picker reused by `wt add` when `--base` is omitted; `styles.go` holds lipgloss color/style definitions.

### Conventions baked into the code

- **Worktree layout:** worktrees are **sibling directories** named `<repo>--<name>` (see `SiblingPath`/`deriveName` in `worktree.go`). A worktree for `myrepo` named `feature` lives at `../myrepo--feature`. `deriveName` reverses this to show clean names in listings. `wt add <name>` also creates a new branch named `<name>`.
- **Fuzzy resolution:** `FindWorktree` (in `worktree.go`) resolves a name argument by exact → prefix → substring match, erroring on ambiguity. `use`/`remove` rely on it.
- **Current worktree detection** compares symlink-resolved paths (`filepath.EvalSymlinks`) against cwd; the first entry from `git worktree list` is treated as the main worktree and cannot be removed.
- **Gitignored file copying:** the main worktree is the canonical source for gitignored files (`.env`, config, `node_modules`, …), which a fresh worktree never receives. `wt sync-ignored [name]` copies them from main into the current (or named) worktree; `wt add -s` does the same right after creating a worktree, and the interactive add flow asks before creating. Fully-ignored directories are copied wholesale via the `--directory` flag on `git ls-files`. This is an operation, not a navigation command, so it prints status to stderr and nothing to stdout.

## Release flow

Fully automated, no manual version bumps:

- **Conventional Commits** drive versioning. `fix:` → patch, `feat:` → minor, `feat!:`/`BREAKING CHANGE` → major. Commit messages matter — release-please parses them.
- Pushes to `main` trigger [release-please](.github/workflows/release-please.yml), which opens/updates a release PR (bumps `CHANGELOG.md` + version). Merging that PR tags a release and runs **GoReleaser** (`.goreleaser.yaml`), which builds cross-platform binaries, publishes the GitHub release, and updates the Homebrew tap (`sushidev-team/homebrew-tap`).
- `version` and `commit` are injected via ldflags at build time (see `main.go`); they are `"dev"`/`"none"` for local builds.
- `wt upgrade` (`cmd/upgrade.go`) self-updates by downloading the matching `worktree-manager_<ver>_<os>_<arch>.tar.gz` from GitHub releases and atomically replacing the running binary. The archive naming here must stay in sync with `.goreleaser.yaml`'s `name_template`.
