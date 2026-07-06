package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/sushi/worktree-manager/internal/git"
)

var syncIgnoredCmd = &cobra.Command{
	Use:     "sync-ignored [name]",
	Aliases: []string{"sync"},
	Short:   "Copy gitignored files from the main worktree",
	Long: `Copy all gitignored files (.env, config, node_modules, etc.) from the main
worktree into a target worktree.

With no argument, files are copied into the current worktree. Pass a name to
fuzzy-match and target another worktree instead.

These files are excluded from git, so a fresh worktree never receives them;
this command replicates them so the new worktree is ready to run.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		src, err := git.MainWorktreePath()
		if err != nil {
			return err
		}

		var dst, label string
		if len(args) == 1 {
			wt, err := git.FindWorktree(args[0])
			if err != nil {
				return err
			}
			dst = wt.Path
			label = wt.Name
		} else {
			dst, err = os.Getwd()
			if err != nil {
				return err
			}
			label = "current worktree"
		}

		if sameDir(src, dst) {
			return fmt.Errorf("target is the main worktree; nothing to sync into it")
		}

		fmt.Fprintf(os.Stderr, "Copying gitignored files into %s...\n", label)
		n, err := git.SyncIgnoredFiles(src, dst)
		if err != nil {
			return err
		}
		if n == 0 {
			fmt.Fprintln(os.Stderr, "No gitignored files to copy.")
		} else {
			fmt.Fprintf(os.Stderr, "Copied %d ignored %s into %s.\n", n, plural(n, "entry", "entries"), label)
		}
		return nil
	},
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// sameDir reports whether two paths point at the same directory, resolving
// symlinks the way the rest of the worktree detection does.
func sameDir(a, b string) bool {
	ra, err := filepath.EvalSymlinks(a)
	if err != nil {
		ra = a
	}
	rb, err := filepath.EvalSymlinks(b)
	if err != nil {
		rb = b
	}
	return ra == rb
}
