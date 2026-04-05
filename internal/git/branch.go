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
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		name := parts[0]
		isCurrent := len(parts) > 1 && parts[1] == "*"
		branches = append(branches, Branch{
			Name:      name,
			IsCurrent: isCurrent,
			IsDefault: name == defaultBranch,
		})
	}

	// Sort: default branch first, then current, then alphabetical
	sortBranches(branches)
	return branches, nil
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
	return a.Name < b.Name
}
