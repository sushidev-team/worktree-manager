package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/sushi/worktree-manager/internal/git"
	"github.com/sushi/worktree-manager/internal/tui"
)

var addBase string

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

		fmt.Print(path)
		return nil
	},
}

func init() {
	addCmd.Flags().StringVarP(&addBase, "base", "b", "", "Base branch (interactive picker if omitted)")
}
