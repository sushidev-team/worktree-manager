# wt — Git Worktree Manager

A fast, interactive CLI for managing git worktrees. Create, switch, list, and remove worktrees with fuzzy search and a beautiful TUI.

![Go](https://img.shields.io/github/go-mod/go-version/sushidev-team/worktree-manager)
![License](https://img.shields.io/github/license/sushidev-team/worktree-manager)
![Release](https://img.shields.io/github/v/release/sushidev-team/worktree-manager)

## Install

### Homebrew (macOS / Linux)

```bash
brew tap sushidev-team/tap
brew install wt
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
```

Creates a new worktree as a sibling directory and switches to it. For a repo at `~/code/myrepo`, the worktree is created at `~/code/myrepo--my-feature`.

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
| `wt add <name> [-b branch]` | Create a new worktree |
| `wt use <name>` | Switch to a worktree (fuzzy match) |
| `wt list` | List all worktrees |
| `wt remove <name> [-f]` | Remove a worktree |
| `wt init-shell` | Print shell wrapper function |

## How It Works

- Worktrees are created as **sibling directories** using a `--` separator: `repo--worktree-name`
- Each worktree gets its own branch (named after the worktree)
- The shell wrapper function captures the path output from `wt` and `cd`s into it
- Dirty worktrees (uncommitted changes) are flagged in both list and interactive views

## License

[MIT](LICENSE)
