// Package gittest provides helpers for building throwaway git repositories in
// tests. It is only imported from _test.go files, so it never ends up in the
// shipped binary.
package gittest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Git runs a git command in dir and fails the test on error, returning the
// combined output.
func Git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	// Keep the environment deterministic regardless of the developer's config.
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v (in %s) failed: %v\n%s", args, dir, err, out)
	}
	return string(out)
}

// WriteFile writes content to dir/rel, creating parent directories.
func WriteFile(t *testing.T, dir, rel, content string) string {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// Repo creates a fresh repository on branch main with a single commit, inside a
// subdirectory of a temp dir so worktree siblings stay within the cleaned-up
// tree. The returned path is symlink-resolved (macOS temp dirs are symlinks),
// which matters because worktree detection compares resolved paths.
func Repo(t *testing.T) string {
	t.Helper()
	return repoOnBranch(t, "main")
}

// RepoOnBranch is like Repo but lets the caller choose the initial branch name
// (used to exercise the master fallback in DefaultBranch).
func RepoOnBranch(t *testing.T, branch string) string {
	t.Helper()
	return repoOnBranch(t, branch)
}

func repoOnBranch(t *testing.T, branch string) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "repo")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	Git(t, dir, "init", "-b", branch)
	Git(t, dir, "config", "commit.gpgsign", "false")
	WriteFile(t, dir, "README.md", "hello\n")
	Git(t, dir, "add", ".")
	Git(t, dir, "commit", "-m", "initial commit")
	return dir
}

// AddRemote creates a bare remote, wires it up as origin, pushes the current
// branch with tracking, and points origin/HEAD at it. Returns the remote path.
func AddRemote(t *testing.T, dir string) string {
	t.Helper()
	remote := dir + "-remote.git"
	Git(t, dir, "init", "--bare", remote)
	Git(t, dir, "remote", "add", "origin", remote)
	branch := strings.TrimSpace(Git(t, dir, "rev-parse", "--abbrev-ref", "HEAD"))
	Git(t, dir, "push", "-u", "origin", branch)
	// Set origin/HEAD explicitly; `-a` can't resolve it on a bare repo whose
	// own HEAD still points at its (unborn) default branch.
	Git(t, dir, "remote", "set-head", "origin", branch)
	return remote
}

// RemoteOnlyBranch creates a branch from base, pushes it to origin, then deletes
// the local branch, leaving only the origin/<name> remote-tracking ref.
func RemoteOnlyBranch(t *testing.T, dir, name, base string) {
	t.Helper()
	Git(t, dir, "branch", name, base)
	Git(t, dir, "push", "origin", name)
	Git(t, dir, "branch", "-D", name)
}

// Chdir switches the process into dir and restores the previous directory when
// the test ends. Tests that use it must not call t.Parallel(), since the working
// directory is process-global.
func Chdir(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
}
