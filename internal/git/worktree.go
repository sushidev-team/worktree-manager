package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Worktree represents a git worktree.
type Worktree struct {
	Path       string
	Branch     string
	Head       string // short commit hash
	IsBare     bool
	IsMain     bool // true if this is the main worktree
	IsCurrent  bool
	IsDirty    bool
	Name       string // derived display name
}

// ListWorktrees returns all worktrees for the current repository.
func ListWorktrees() ([]Worktree, error) {
	out, err := exec.Command("git", "worktree", "list", "--porcelain").Output()
	if err != nil {
		return nil, fmt.Errorf("git worktree list: %w", err)
	}

	currentDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	currentDir, _ = filepath.EvalSymlinks(currentDir)

	var worktrees []Worktree
	var current *Worktree

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "worktree "):
			if current != nil {
				worktrees = append(worktrees, *current)
			}
			path := strings.TrimPrefix(line, "worktree ")
			current = &Worktree{Path: path}
		case strings.HasPrefix(line, "HEAD "):
			if current != nil {
				hash := strings.TrimPrefix(line, "HEAD ")
				if len(hash) > 7 {
					hash = hash[:7]
				}
				current.Head = hash
			}
		case strings.HasPrefix(line, "branch "):
			if current != nil {
				ref := strings.TrimPrefix(line, "branch ")
				current.Branch = strings.TrimPrefix(ref, "refs/heads/")
			}
		case line == "bare":
			if current != nil {
				current.IsBare = true
			}
		case line == "":
			// separator between entries
		}
	}
	if current != nil {
		worktrees = append(worktrees, *current)
	}

	// Mark main worktree and current, derive names
	repoRoot, _ := RepoRoot()
	for i := range worktrees {
		wt := &worktrees[i]
		if i == 0 {
			wt.IsMain = true
		}
		resolved, _ := filepath.EvalSymlinks(wt.Path)
		if resolved == currentDir {
			wt.IsCurrent = true
		}
		wt.Name = deriveName(wt.Path, repoRoot)
		wt.IsDirty = isWorktreeDirty(wt.Path)
	}

	return worktrees, nil
}

// AddWorktree creates a new worktree as a sibling directory.
func AddWorktree(name, baseBranch string) (string, error) {
	repoRoot, err := RepoRoot()
	if err != nil {
		return "", err
	}

	wtPath := SiblingPath(repoRoot, name)

	// Create new branch from base
	branchName := name
	args := []string{"worktree", "add", "-b", branchName, wtPath, baseBranch}
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git worktree add: %s", strings.TrimSpace(string(out)))
	}

	return wtPath, nil
}

// RemoveWorktree removes a worktree by path.
func RemoveWorktree(path string) error {
	cmd := exec.Command("git", "worktree", "remove", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree remove: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// ForceRemoveWorktree removes a worktree forcefully.
func ForceRemoveWorktree(path string) error {
	cmd := exec.Command("git", "worktree", "remove", "--force", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree remove --force: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// RepoRoot returns the top-level directory of the git repository.
func RepoRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		// Might be in a bare repo or worktree; try common dir
		out, err = exec.Command("git", "rev-parse", "--git-common-dir").Output()
		if err != nil {
			return "", fmt.Errorf("not in a git repository")
		}
		commonDir := strings.TrimSpace(string(out))
		if filepath.IsAbs(commonDir) {
			return filepath.Dir(commonDir), nil
		}
		cwd, _ := os.Getwd()
		return filepath.Dir(filepath.Join(cwd, commonDir)), nil
	}
	return strings.TrimSpace(string(out)), nil
}

// SiblingPath computes the worktree path as a sibling directory.
func SiblingPath(repoRoot, name string) string {
	parent := filepath.Dir(repoRoot)
	base := filepath.Base(repoRoot)
	return filepath.Join(parent, base+"--"+name)
}

// FindWorktree finds a worktree by fuzzy-matching on name.
func FindWorktree(name string) (*Worktree, error) {
	worktrees, err := ListWorktrees()
	if err != nil {
		return nil, err
	}

	nameLower := strings.ToLower(name)

	// Exact match first
	for i := range worktrees {
		if strings.ToLower(worktrees[i].Name) == nameLower {
			return &worktrees[i], nil
		}
	}

	// Prefix match
	var matches []*Worktree
	for i := range worktrees {
		if strings.HasPrefix(strings.ToLower(worktrees[i].Name), nameLower) {
			matches = append(matches, &worktrees[i])
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}

	// Contains match
	matches = nil
	for i := range worktrees {
		if strings.Contains(strings.ToLower(worktrees[i].Name), nameLower) {
			matches = append(matches, &worktrees[i])
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		names := make([]string, len(matches))
		for i, m := range matches {
			names[i] = m.Name
		}
		return nil, fmt.Errorf("ambiguous worktree name %q, matches: %s", name, strings.Join(names, ", "))
	}

	return nil, fmt.Errorf("no worktree found matching %q", name)
}

func deriveName(wtPath, repoRoot string) string {
	base := filepath.Base(repoRoot)
	wtBase := filepath.Base(wtPath)

	// If it's the main repo directory
	if wtPath == repoRoot {
		return base + " (main)"
	}

	// If it follows our naming convention: repo--name
	prefix := base + "--"
	if strings.HasPrefix(wtBase, prefix) {
		return strings.TrimPrefix(wtBase, prefix)
	}

	// Fallback to directory name
	return wtBase
}

func isWorktreeDirty(path string) bool {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = path
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(out))) > 0
}
