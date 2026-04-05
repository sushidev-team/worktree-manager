package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/sushi/worktree-manager/internal/git"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all worktrees",
	RunE: func(cmd *cobra.Command, args []string) error {
		worktrees, err := git.ListWorktrees()
		if err != nil {
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tBRANCH\tCOMMIT\tSTATUS\tPATH")
		for _, wt := range worktrees {
			status := ""
			if wt.IsCurrent {
				status = "● current"
			}
			if wt.IsDirty {
				if status != "" {
					status += ", "
				}
				status += "✱ dirty"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				wt.Name, wt.Branch, wt.Head, status, wt.Path)
		}
		w.Flush()
		return nil
	},
}
