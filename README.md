# wt — Git Worktree Manager

A fast, interactive CLI for managing git worktrees. Create, switch, list, and remove worktrees with fuzzy search and a beautiful TUI.

![Go](https://img.shields.io/github/go-mod/go-version/sushidev-team/worktree-manager)
![License](https://img.shields.io/github/license/sushidev-team/worktree-manager)
![Release](https://img.shields.io/github/v/release/sushidev-team/worktree-manager)

## Install

### Quick Install (macOS / Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/sushidev-team/worktree-manager/main/install.sh | sh
```

Or specify a custom install directory:

```bash
INSTALL_DIR=~/.local/bin curl -fsSL https://raw.githubusercontent.com/sushidev-team/worktree-manager/main/install.sh | sh
```

### Homebrew (macOS)

```bash
brew install sushidev-team/tap/wt
```

### Go

```bash
go install github.com/sushidev-team/worktree-manager@latest
```

### Binary Download

Download the latest binary for your platform from [Releases](https://github.com/sushidev-team/worktree-manager/releases).

## Shell Setup

`wt` needs a shell wrapper to change directories. Add this to your `~/.zshrc` or `~/.bashrc`:

```bash
eval "$(wt init-shell)"
```

Then restart your shell or run `source ~/.zshrc`.

## Usage

### Interactive Mode

```bash
wt
```

Opens a full-screen interactive TUI with all your worktrees. Features:

- **Fuzzy search** — type `/` to filter worktrees
- **Enter** — switch to the selected worktree
- **a** — add a new worktree (with branch picker)
- **d** — delete the selected worktree
- **q** — quit

### Create a Worktree

```bash
# Interactive branch picker
wt add my-feature

# Specify base branch directly
wt add my-feature --base main

# Also copy gitignored files (.env, config, node_modules, …) into the new worktree
wt add my-feature --base main --sync-ignored
```

Creates a new worktree as a sibling directory and switches to it. For a repo at `~/code/myrepo`, the worktree is created at `~/code/myrepo--my-feature`.

In the interactive TUI, the add flow asks whether to copy gitignored files before creating the worktree.

### Copy Gitignored Files

Git worktrees only check out tracked files, so a fresh worktree is missing anything git ignores — `.env` secrets, local config, `node_modules`, build output. `sync-ignored` copies those from the main worktree so the new one is ready to run:

```bash
# Copy gitignored files from the main worktree into the current one
wt sync-ignored

# Or target another worktree by name (fuzzy match)
wt sync-ignored my-feature
```

`sync` works as an alias: `wt sync`. Existing files in the target are overwritten.

### Switch to a Worktree

```bash
wt use my-feature

# Fuzzy matching works — just type enough to be unique
wt use feat
```

### List Worktrees

```bash
wt list
```

```
NAME                  BRANCH        COMMIT   STATUS              PATH
myrepo (main)         main          a1b2c3d  ● current           ~/code/myrepo
my-feature            my-feature    d4e5f6a  ✱ dirty             ~/code/myrepo--my-feature
bugfix                fix/login     b7c8d9e                      ~/code/myrepo--bugfix
```

`ls` works as an alias: `wt ls`

### Remove a Worktree

```bash
wt remove my-feature

# Skip confirmation
wt remove my-feature --force
```

`rm` works as an alias: `wt rm my-feature`

## Commands

| Command | Description |
|---|---|
| `wt` | Interactive TUI — browse, switch, add, delete |
| `wt add <name> [-b branch] [-s]` | Create a new worktree (`-s` copies gitignored files) |
| `wt use <name>` | Switch to a worktree (fuzzy match) |
| `wt list` | List all worktrees |
| `wt remove <name> [-f]` | Remove a worktree |
| `wt sync-ignored [name]` | Copy gitignored files from the main worktree |
| `wt init-shell` | Print shell wrapper function |
| `wt upgrade` | Self-update to the latest release |

## How It Works

- Worktrees are created as **sibling directories** using a `--` separator: `repo--worktree-name`
- Each worktree gets its own branch (named after the worktree)
- The shell wrapper function captures the path output from `wt` and `cd`s into it
- Dirty worktrees (uncommitted changes) are flagged in both list and interactive views

## License

[MIT](LICENSE)
