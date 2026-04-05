package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sushi/worktree-manager/internal/git"
)

var removeForce bool

var removeCmd = &cobra.Command{
	Use:     "remove <name>",
	Aliases: []string{"rm"},
	Short:   "Remove a worktree",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		wt, err := git.FindWorktree(args[0])
		if err != nil {
			return err
		}

		if wt.IsMain {
			return fmt.Errorf("cannot remove the main worktree")
		}

		if wt.IsCurrent {
			return fmt.Errorf("cannot remove the current worktree (switch to another first)")
		}

		if !removeForce {
			prompt := fmt.Sprintf("Remove worktree '%s'?", wt.Name)
			if wt.IsDirty {
				prompt += " (has uncommitted changes!)"
			}
			prompt += " [y/N] "
			fmt.Fprint(os.Stderr, prompt)

			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))
			if answer != "y" && answer != "yes" {
				fmt.Fprintln(os.Stderr, "Cancelled.")
				return nil
			}
		}

		if wt.IsDirty || removeForce {
			err = git.ForceRemoveWorktree(wt.Path)
		} else {
			err = git.RemoveWorktree(wt.Path)
		}
		if err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "Removed worktree '%s'\n", wt.Name)
		return nil
	},
}

func init() {
	removeCmd.Flags().BoolVarP(&removeForce, "force", "f", false, "Force removal even with uncommitted changes")
}
