package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// Worktree represents a git worktree.
type Worktree struct {
	Path        string
	Branch      string
	Head        string // short commit hash
	IsBare      bool
	IsMain      bool // true if this is the main worktree
	IsCurrent   bool
	IsDirty     bool
	Name        string // derived display name
	DirtyCount  int    // number of changed entries (git status --porcelain lines)
	Ahead       int    // commits ahead of upstream
	Behind      int    // commits behind upstream
	HasUpstream bool   // whether the branch tracks an upstream
	LastCommit  string // relative time of last commit, e.g. "2 hours ago"
	Subject     string // last commit subject
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

	// Mark main worktree and current, derive names. enrich() shells out ~3
	// times per worktree, so run those concurrently to keep startup snappy.
	repoRoot, _ := RepoRoot()
	var wg sync.WaitGroup
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
		wg.Add(1)
		go func() {
			defer wg.Done()
			enrich(wt)
		}()
	}
	wg.Wait()

	return worktrees, nil
}

// WorktreeSpec describes a worktree to create.
type WorktreeSpec struct {
	Name   string // worktree (directory) name; the sibling suffix
	Branch string // branch the worktree ends up on
	// CreateFrom, when non-empty, creates Branch as a new local branch starting
	// at this ref (a branch or a remote-tracking ref like "origin/foo", which
	// also sets up tracking). When empty, Branch must already exist and is
	// checked out directly.
	CreateFrom string
}

// AddWorktree creates a new worktree as a sibling directory.
func AddWorktree(spec WorktreeSpec) (string, error) {
	repoRoot, err := RepoRoot()
	if err != nil {
		return "", err
	}

	wtPath := SiblingPath(repoRoot, spec.Name)

	var args []string
	if spec.CreateFrom != "" {
		// Create a new branch from the given start-point.
		args = []string{"worktree", "add", "-b", spec.Branch, wtPath, spec.CreateFrom}
	} else {
		// Check out an existing branch.
		args = []string{"worktree", "add", wtPath, spec.Branch}
	}

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

// gitOut runs git in dir and returns trimmed stdout.
func gitOut(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// enrich fills the derived status fields for a worktree: dirty count,
// ahead/behind vs upstream, and last-commit info. Failures are non-fatal —
// a field just stays at its zero value.
func enrich(wt *Worktree) {
	if out, err := gitOut(wt.Path, "status", "--porcelain"); err == nil && out != "" {
		wt.DirtyCount = len(strings.Split(out, "\n"))
	}
	wt.IsDirty = wt.DirtyCount > 0

	if out, err := gitOut(wt.Path, "log", "-1", "--format=%cr%x1f%s"); err == nil {
		if parts := strings.SplitN(out, "\x1f", 2); len(parts) == 2 {
			wt.LastCommit = parts[0]
			wt.Subject = parts[1]
		}
	}

	// left-right count of upstream...HEAD: left = behind, right = ahead.
	if out, err := gitOut(wt.Path, "rev-list", "--left-right", "--count", "@{u}...HEAD"); err == nil {
		if f := strings.Fields(out); len(f) == 2 {
			wt.HasUpstream = true
			wt.Behind, _ = strconv.Atoi(f[0])
			wt.Ahead, _ = strconv.Atoi(f[1])
		}
	}
}
