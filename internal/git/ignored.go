package git

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// MainWorktreePath returns the filesystem path of the main worktree.
// This is the canonical source for gitignored files (.env, config, etc.).
func MainWorktreePath() (string, error) {
	worktrees, err := ListWorktrees()
	if err != nil {
		return "", err
	}
	for i := range worktrees {
		if worktrees[i].IsMain {
			return worktrees[i].Path, nil
		}
	}
	return "", fmt.Errorf("no main worktree found")
}

// ListIgnoredEntries returns the top-level gitignored paths (files and
// directories) in dir, relative to dir. Fully-ignored directories are
// collapsed to a single entry (e.g. "node_modules") rather than every file
// inside them.
func ListIgnoredEntries(dir string) ([]string, error) {
	cmd := exec.Command("git", "ls-files", "-z",
		"--others", "--ignored", "--exclude-standard", "--directory")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files (ignored): %w", err)
	}

	var entries []string
	for e := range strings.SplitSeq(string(out), "\x00") {
		e = strings.TrimSuffix(e, "/")
		if e == "" || e == ".git" {
			continue
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// SyncIgnoredFiles copies every gitignored file and directory from src into
// dst, preserving relative paths and permissions. Existing files in dst are
// overwritten. Returns the number of top-level entries copied.
func SyncIgnoredFiles(src, dst string) (int, error) {
	entries, err := ListIgnoredEntries(src)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, rel := range entries {
		if err := copyPath(filepath.Join(src, rel), filepath.Join(dst, rel)); err != nil {
			return count, fmt.Errorf("copy %q: %w", rel, err)
		}
		count++
	}
	return count, nil
}

// copyPath copies a file, directory, or symlink from src to dst.
func copyPath(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	switch {
	case info.Mode()&os.ModeSymlink != 0:
		return copySymlink(src, dst)
	case info.IsDir():
		return copyDir(src, dst, info)
	default:
		return copyFile(src, dst, info)
	}
}

func copyDir(src, dst string, info os.FileInfo) error {
	if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := copyPath(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string, info os.FileInfo) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func copySymlink(src, dst string) error {
	target, err := os.Readlink(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	os.Remove(dst)
	return os.Symlink(target, dst)
}
