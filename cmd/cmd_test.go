package cmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sushi/worktree-manager/internal/git"
	"github.com/sushi/worktree-manager/internal/gittest"
)

// capture redirects os.Stdout and os.Stderr for the duration of f and returns
// what was written to each. The commands print the cd-path to stdout and all UI
// to stderr, so tests assert on both streams.
func capture(t *testing.T, f func() error) (stdout, stderr string, err error) {
	t.Helper()
	oldOut, oldErr := os.Stdout, os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout, os.Stderr = wOut, wErr

	outCh, errCh := make(chan string, 1), make(chan string, 1)
	go func() { b, _ := io.ReadAll(rOut); outCh <- string(b) }()
	go func() { b, _ := io.ReadAll(rErr); errCh <- string(b) }()

	err = f()
	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout, os.Stderr = oldOut, oldErr
	return <-outCh, <-errCh, err
}

func TestInitShellCommand(t *testing.T) {
	out, _, err := capture(t, func() error {
		initShellCmd.Run(initShellCmd, nil)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "wt() {") || !strings.Contains(out, `cd "$result"`) {
		t.Fatalf("init-shell output looks wrong:\n%s", out)
	}
}

func TestListCommand(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Chdir(t, dir)
	if _, err := git.AddWorktree(git.WorktreeSpec{Name: "feature", Branch: "feature", CreateFrom: "main"}); err != nil {
		t.Fatal(err)
	}

	out, _, err := capture(t, func() error { return listCmd.RunE(listCmd, nil) })
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"NAME", "BRANCH", "COMMIT", "STATUS", "PATH", "feature", "main"} {
		if !strings.Contains(out, want) {
			t.Errorf("list output missing %q:\n%s", want, out)
		}
	}
}

func TestUseCommand(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Chdir(t, dir)
	path, err := git.AddWorktree(git.WorktreeSpec{Name: "feature", Branch: "feature", CreateFrom: "main"})
	if err != nil {
		t.Fatal(err)
	}

	out, _, err := capture(t, func() error { return useCmd.RunE(useCmd, []string{"feature"}) })
	if err != nil {
		t.Fatal(err)
	}
	if out != path {
		t.Fatalf("use printed %q, want the worktree path %q", out, path)
	}
}

func TestUseCommandNotFound(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Chdir(t, dir)
	_, _, err := capture(t, func() error { return useCmd.RunE(useCmd, []string{"missing"}) })
	if err == nil {
		t.Fatal("use should error on an unknown worktree")
	}
}

func TestAddCommand(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Chdir(t, dir)

	addBase = "main"
	addSync = false
	t.Cleanup(func() { addBase = ""; addSync = false })

	out, _, err := capture(t, func() error { return addCmd.RunE(addCmd, []string{"newwt"}) })
	if err != nil {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(out); statErr != nil {
		t.Fatalf("add should print an existing worktree path, got %q (%v)", out, statErr)
	}
	// The CLI keeps the convention that the branch is named after the worktree.
	branch := strings.TrimSpace(gittest.Git(t, out, "rev-parse", "--abbrev-ref", "HEAD"))
	if branch != "newwt" {
		t.Fatalf("add created branch %q, want newwt", branch)
	}
}

func TestRemoveCommandForce(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Chdir(t, dir)
	path, err := git.AddWorktree(git.WorktreeSpec{Name: "gone", Branch: "gone", CreateFrom: "main"})
	if err != nil {
		t.Fatal(err)
	}

	removeForce = true
	t.Cleanup(func() { removeForce = false })

	_, _, err = capture(t, func() error { return removeCmd.RunE(removeCmd, []string{"gone"}) })
	if err != nil {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("worktree should be removed")
	}
}

func TestRemoveCommandRejectsMain(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Chdir(t, dir)
	// Add a second worktree so "repo" uniquely substring-matches only main.
	if _, err := git.AddWorktree(git.WorktreeSpec{Name: "side", Branch: "side", CreateFrom: "main"}); err != nil {
		t.Fatal(err)
	}

	removeForce = true
	t.Cleanup(func() { removeForce = false })

	_, _, err := capture(t, func() error { return removeCmd.RunE(removeCmd, []string{"repo"}) })
	if err == nil || !strings.Contains(err.Error(), "main worktree") {
		t.Fatalf("removing main should error, got %v", err)
	}
}

func TestRemoveCommandRejectsCurrent(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Chdir(t, dir)
	path, err := git.AddWorktree(git.WorktreeSpec{Name: "here", Branch: "here", CreateFrom: "main"})
	if err != nil {
		t.Fatal(err)
	}
	// Switch into the worktree so it counts as current.
	gittest.Chdir(t, path)

	removeForce = true
	t.Cleanup(func() { removeForce = false })

	_, _, err = capture(t, func() error { return removeCmd.RunE(removeCmd, []string{"here"}) })
	if err == nil || !strings.Contains(err.Error(), "current worktree") {
		t.Fatalf("removing current worktree should error, got %v", err)
	}
}

func TestSyncIgnoredCommand(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.WriteFile(t, dir, ".gitignore", ".env\n")
	gittest.Git(t, dir, "add", ".gitignore")
	gittest.Git(t, dir, "commit", "-m", "gitignore")
	gittest.WriteFile(t, dir, ".env", "SECRET=1\n")
	gittest.Chdir(t, dir)

	path, err := git.AddWorktree(git.WorktreeSpec{Name: "fresh", Branch: "fresh", CreateFrom: "main"})
	if err != nil {
		t.Fatal(err)
	}
	// Run the command from within the fresh worktree (no arg = current).
	gittest.Chdir(t, path)
	_, stderr, err := capture(t, func() error { return syncIgnoredCmd.RunE(syncIgnoredCmd, nil) })
	if err != nil {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(filepath.Join(path, ".env")); statErr != nil {
		t.Fatalf(".env should have been copied in: %v", statErr)
	}
	if !strings.Contains(stderr, "Copied") {
		t.Errorf("expected a 'Copied' status on stderr, got %q", stderr)
	}
}

func TestSyncIgnoredCommandRejectsMain(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Chdir(t, dir)
	_, _, err := capture(t, func() error { return syncIgnoredCmd.RunE(syncIgnoredCmd, nil) })
	if err == nil || !strings.Contains(err.Error(), "main worktree") {
		t.Fatalf("sync into main should error, got %v", err)
	}
}

// --- pure helpers -----------------------------------------------------------

func TestPlural(t *testing.T) {
	if got := plural(1, "entry", "entries"); got != "entry" {
		t.Errorf("plural(1) = %q", got)
	}
	if got := plural(0, "entry", "entries"); got != "entries" {
		t.Errorf("plural(0) = %q", got)
	}
	if got := plural(2, "entry", "entries"); got != "entries" {
		t.Errorf("plural(2) = %q", got)
	}
}

func TestSameDir(t *testing.T) {
	dir := t.TempDir()
	if !sameDir(dir, dir) {
		t.Error("identical paths should be the same dir")
	}
	other := t.TempDir()
	if sameDir(dir, other) {
		t.Error("different paths should not be the same dir")
	}
}

func TestSetVersion(t *testing.T) {
	SetVersion("1.2.3", "abcdef")
	if version != "1.2.3" {
		t.Errorf("version = %q, want 1.2.3", version)
	}
	if rootCmd.Version != "1.2.3 (abcdef)" {
		t.Errorf("rootCmd.Version = %q", rootCmd.Version)
	}
}

func TestExtractBinaryFromTarGz(t *testing.T) {
	want := []byte("#!/bin/sh\necho hi\n")
	archive := makeTarGz(t, map[string][]byte{"README": []byte("x"), "wt": want})

	got, err := extractBinaryFromTarGz(bytes.NewReader(archive), "wt")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("extracted %q, want %q", got, want)
	}
}

func TestExtractBinaryFromTarGzMissing(t *testing.T) {
	archive := makeTarGz(t, map[string][]byte{"other": []byte("x")})
	if _, err := extractBinaryFromTarGz(bytes.NewReader(archive), "wt"); err == nil {
		t.Fatal("expected an error when the binary is absent from the archive")
	}
}

func makeTarGz(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		hdr := &tar.Header{Name: name, Mode: 0o755, Size: int64(len(content))}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
