package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const shellFunction = `# Add this to your .zshrc or .bashrc
wt() {
  local result
  result="$(command wt "$@")" || { echo "$result" >&2; return $?; }
  if [ -d "$result" ]; then
    cd "$result" || return $?
    echo "Switched to: $(pwd)" >&2
  elif [ -n "$result" ]; then
    echo "$result"
  fi
}
`

var initShellCmd = &cobra.Command{
	Use:   "init-shell",
	Short: "Print shell function for directory switching",
	Long: `Print a shell function that wraps the wt binary to enable directory switching.

Add the output to your .zshrc or .bashrc:

  eval "$(wt init-shell)"

Or manually copy the function.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(shellFunction)
	},
}
