package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/sushi/worktree-manager/internal/git"
)

var useCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Switch to a worktree",
	Long:  "Fuzzy-match a worktree by name and print its path for the shell function to cd into.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		wt, err := git.FindWorktree(args[0])
		if err != nil {
			return err
		}
		fmt.Print(wt.Path)
		return nil
	},
}
