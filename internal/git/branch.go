package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// Branch represents a git branch.
type Branch struct {
	Name      string
	IsCurrent bool
	IsDefault bool
	IsRemote  bool
}

// ListBranches returns local branches sorted with default branch first.
func ListBranches() ([]Branch, error) {
	defaultBranch := DefaultBranch()

	out, err := exec.Command("git", "branch", "--format=%(refname:short) %(HEAD)").Output()
	if err != nil {
		return nil, fmt.Errorf("git branch: %w", err)
	}

	var branches []Branch
	localNames := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		name := parts[0]
		isCurrent := len(parts) > 1 && parts[1] == "*"
		localNames[name] = true
		branches = append(branches, Branch{
			Name:      name,
			IsCurrent: isCurrent,
			IsDefault: name == defaultBranch,
		})
	}

	// Append remote-tracking branches that have no local counterpart, so a new
	// worktree can be based on a branch a colleague pushed but that was never
	// checked out here. The full ref (e.g. "origin/feature") is kept as the name
	// so git can resolve it as a start point, and so the new local branch tracks it.
	rout, err := exec.Command("git", "branch", "-r", "--format=%(refname:short)").Output()
	if err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(rout)), "\n") {
			line = strings.TrimSpace(line)
			// Skip blanks and symbolic refs like "origin/HEAD -> origin/main"
			// (short form is the bare remote name "origin", i.e. no slash).
			if line == "" || strings.Contains(line, "->") || !strings.Contains(line, "/") {
				continue
			}
			short := line[strings.IndexByte(line, '/')+1:]
			if localNames[short] {
				continue
			}
			branches = append(branches, Branch{
				Name:     line,
				IsRemote: true,
			})
		}
	}

	// Sort: default branch first, then current, then locals, then remotes,
	// each group alphabetical.
	sortBranches(branches)
	return branches, nil
}

// Fetch updates remote-tracking branches from all remotes, pruning refs for
// branches that were deleted upstream. Lets the picker surface branches a
// colleague pushed after the TUI was opened.
func Fetch() error {
	cmd := exec.Command("git", "fetch", "--all", "--prune")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git fetch: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// DefaultBranch detects the default branch (main or master).
func DefaultBranch() string {
	// Try symbolic ref of origin/HEAD
	out, err := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD").Output()
	if err == nil {
		ref := strings.TrimSpace(string(out))
		return strings.TrimPrefix(ref, "refs/remotes/origin/")
	}

	// Check if main exists
	if err := exec.Command("git", "rev-parse", "--verify", "main").Run(); err == nil {
		return "main"
	}

	// Fallback to master
	if err := exec.Command("git", "rev-parse", "--verify", "master").Run(); err == nil {
		return "master"
	}

	return "main"
}

func sortBranches(branches []Branch) {
	// Simple bubble sort since branch lists are small
	n := len(branches)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if branchLess(branches[j], branches[i]) {
				branches[i], branches[j] = branches[j], branches[i]
			}
		}
	}
}

func branchLess(a, b Branch) bool {
	if a.IsDefault != b.IsDefault {
		return a.IsDefault
	}
	if a.IsCurrent != b.IsCurrent {
		return a.IsCurrent
	}
	if a.IsRemote != b.IsRemote {
		return !a.IsRemote // local branches before remote-only ones
	}
	return a.Name < b.Name
}
