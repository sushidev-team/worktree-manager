package main

import (
	"fmt"
	"os"

	"github.com/sushi/worktree-manager/cmd"
)

// Set by goreleaser ldflags.
var (
	version = "dev"
	commit  = "none"
)

func main() {
	cmd.SetVersion(version, commit)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
