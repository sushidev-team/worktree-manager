package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sushi/worktree-manager/internal/git"
	"github.com/sushi/worktree-manager/internal/tui"
)

var (
	addBase string
	addSync bool
)

var addCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Create a new worktree",
	Long:  "Create a new worktree as a sibling directory with a new branch.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		baseBranch := addBase
		if baseBranch == "" {
			// Interactive branch picker
			selected, err := tui.PickBranch()
			if err != nil {
				return err
			}
			baseBranch = selected
		}

		path, err := git.AddWorktree(name, baseBranch)
		if err != nil {
			return err
		}

		if addSync {
			src, err := git.MainWorktreePath()
			if err != nil {
				return err
			}
			fmt.Fprint(os.Stderr, "Copying gitignored files...\n")
			n, err := git.SyncIgnoredFiles(src, path)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Copied %d ignored %s.\n", n, plural(n, "entry", "entries"))
		}

		fmt.Print(path)
		return nil
	},
}

func init() {
	addCmd.Flags().StringVarP(&addBase, "base", "b", "", "Base branch (interactive picker if omitted)")
	addCmd.Flags().BoolVarP(&addSync, "sync-ignored", "s", false, "Copy gitignored files from the main worktree into the new worktree")
}
