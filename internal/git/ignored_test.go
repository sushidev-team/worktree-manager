package git

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/sushi/worktree-manager/internal/gittest"
)

// ignoredRepo builds a repo whose .gitignore hides an env file, a whole
// directory, and a symlink, then materializes those ignored entries.
func ignoredRepo(t *testing.T) string {
	t.Helper()
	dir := gittest.Repo(t)
	gittest.WriteFile(t, dir, ".gitignore", ".env\n/node_modules/\nlink\n")
	gittest.Git(t, dir, "add", ".gitignore")
	gittest.Git(t, dir, "commit", "-m", "add gitignore")

	gittest.WriteFile(t, dir, ".env", "SECRET=1\n")
	gittest.WriteFile(t, dir, "node_modules/pkg/index.js", "module.exports={}\n")
	if err := os.Symlink("README.md", filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestListIgnoredEntries(t *testing.T) {
	dir := ignoredRepo(t)

	entries, err := ListIgnoredEntries(dir)
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(entries)
	want := []string{".env", "link", "node_modules"}
	if len(entries) != len(want) {
		t.Fatalf("entries = %v, want %v", entries, want)
	}
	for i := range want {
		if entries[i] != want[i] {
			t.Fatalf("entries = %v, want %v", entries, want)
		}
	}
}

func TestSyncIgnoredFiles(t *testing.T) {
	src := ignoredRepo(t)
	gittest.Chdir(t, src)

	// A fresh worktree never receives ignored files.
	dst, err := AddWorktree(WorktreeSpec{Name: "fresh", Branch: "fresh", CreateFrom: "main"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dst, ".env")); !os.IsNotExist(err) {
		t.Fatalf(".env should not exist in a fresh worktree yet")
	}

	n, err := SyncIgnoredFiles(src, dst)
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("copied %d entries, want 3", n)
	}

	// The regular file and the nested directory file are copied through.
	assertContent(t, filepath.Join(dst, ".env"), "SECRET=1\n")
	assertContent(t, filepath.Join(dst, "node_modules/pkg/index.js"), "module.exports={}\n")

	// The symlink is copied as a symlink, not dereferenced.
	fi, err := os.Lstat(filepath.Join(dst, "link"))
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("link should be copied as a symlink")
	}
	target, err := os.Readlink(filepath.Join(dst, "link"))
	if err != nil || target != "README.md" {
		t.Fatalf("symlink target = %q err=%v, want README.md", target, err)
	}
}

func TestSyncIgnoredFilesOverwrites(t *testing.T) {
	src := ignoredRepo(t)
	gittest.Chdir(t, src)
	dst, err := AddWorktree(WorktreeSpec{Name: "over", Branch: "over", CreateFrom: "main"})
	if err != nil {
		t.Fatal(err)
	}
	// Pre-existing stale copy should be overwritten by the sync.
	gittest.WriteFile(t, dst, ".env", "STALE=1\n")
	if _, err := SyncIgnoredFiles(src, dst); err != nil {
		t.Fatal(err)
	}
	assertContent(t, filepath.Join(dst, ".env"), "SECRET=1\n")
}

func TestSyncIgnoredFilesNone(t *testing.T) {
	src := gittest.Repo(t) // no .gitignore, nothing ignored
	gittest.Chdir(t, src)
	dst, err := AddWorktree(WorktreeSpec{Name: "empty", Branch: "empty", CreateFrom: "main"})
	if err != nil {
		t.Fatal(err)
	}
	n, err := SyncIgnoredFiles(src, dst)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("copied %d, want 0", n)
	}
}

func assertContent(t *testing.T, path, want string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	if string(b) != want {
		t.Fatalf("%s = %q, want %q", path, string(b), want)
	}
}
