package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/sushi/worktree-manager/internal/tui"
)

var rootCmd = &cobra.Command{
	Use:   "wt",
	Short: "Git worktree manager",
	Long:  "A convenient CLI for managing git worktrees with an interactive TUI.",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := tui.RunInteractive()
		if err != nil {
			return err
		}
		if path != "" {
			fmt.Print(path)
		}
		return nil
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

var version = "dev"

// SetVersion sets the version string shown by --version.
func SetVersion(v, commit string) {
	version = v
	rootCmd.Version = fmt.Sprintf("%s (%s)", v, commit)
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(useCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(initShellCmd)
	rootCmd.AddCommand(upgradeCmd)
}

