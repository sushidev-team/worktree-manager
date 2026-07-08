package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sushi/worktree-manager/internal/gittest"
)

func TestSiblingPath(t *testing.T) {
	got := SiblingPath("/home/me/myrepo", "feature")
	want := filepath.Join("/home/me", "myrepo--feature")
	if got != want {
		t.Fatalf("SiblingPath = %q, want %q", got, want)
	}
}

func TestDeriveName(t *testing.T) {
	repoRoot := "/home/me/myrepo"
	cases := map[string]string{
		"/home/me/myrepo":          "myrepo (main)",
		"/home/me/myrepo--feature": "feature",
		"/home/me/unrelated":       "unrelated",
	}
	for path, want := range cases {
		if got := deriveName(path, repoRoot); got != want {
			t.Errorf("deriveName(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestRepoRootFromSubdir(t *testing.T) {
	dir := gittest.Repo(t)
	sub := filepath.Join(dir, "nested", "deep")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	gittest.Chdir(t, sub)

	root, err := RepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	if root != dir {
		t.Fatalf("RepoRoot() = %q, want %q", root, dir)
	}
}

func TestListWorktreesSingle(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Chdir(t, dir)

	wts, err := ListWorktrees()
	if err != nil {
		t.Fatal(err)
	}
	if len(wts) != 1 {
		t.Fatalf("want 1 worktree, got %d", len(wts))
	}
	m := wts[0]
	if !m.IsMain {
		t.Error("only worktree should be main")
	}
	if !m.IsCurrent {
		t.Error("main should be current when cwd is the repo root")
	}
	if m.Branch != "main" {
		t.Errorf("branch = %q, want main", m.Branch)
	}
	if m.Name != "repo (main)" {
		t.Errorf("name = %q, want 'repo (main)'", m.Name)
	}
	if m.Head == "" {
		t.Error("head hash should be populated")
	}
}

func TestListWorktreesMultipleAndDirty(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Chdir(t, dir)

	wtPath, err := AddWorktree(WorktreeSpec{Name: "feature", Branch: "feature", CreateFrom: "main"})
	if err != nil {
		t.Fatal(err)
	}
	// Make the new worktree dirty with an untracked file.
	if err := os.WriteFile(filepath.Join(wtPath, "scratch.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	wts, err := ListWorktrees()
	if err != nil {
		t.Fatal(err)
	}
	if len(wts) != 2 {
		t.Fatalf("want 2 worktrees, got %d", len(wts))
	}

	feat := worktreeByName(t, wts, "feature")
	if feat.IsMain {
		t.Error("feature worktree should not be main")
	}
	if feat.Branch != "feature" {
		t.Errorf("feature branch = %q", feat.Branch)
	}
	if !feat.IsDirty || feat.DirtyCount < 1 {
		t.Errorf("feature worktree should be dirty, got dirty=%v count=%d", feat.IsDirty, feat.DirtyCount)
	}
}

func TestListWorktreesAheadCount(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.AddRemote(t, dir) // sets upstream main -> origin/main
	// A local commit that is not pushed → 1 ahead.
	gittest.WriteFile(t, dir, "new.txt", "content\n")
	gittest.Git(t, dir, "add", ".")
	gittest.Git(t, dir, "commit", "-m", "local work")
	gittest.Chdir(t, dir)

	wts, err := ListWorktrees()
	if err != nil {
		t.Fatal(err)
	}
	m := wts[0]
	if !m.HasUpstream {
		t.Fatal("main should have an upstream")
	}
	if m.Ahead != 1 || m.Behind != 0 {
		t.Fatalf("ahead/behind = %d/%d, want 1/0", m.Ahead, m.Behind)
	}
}

func TestAddWorktreeNewBranch(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Chdir(t, dir)

	path, err := AddWorktree(WorktreeSpec{Name: "feat", Branch: "feat", CreateFrom: "main"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("worktree path missing: %v", err)
	}
	if b := currentBranch(t, path); b != "feat" {
		t.Fatalf("worktree on branch %q, want feat", b)
	}
}

func TestAddWorktreeCheckoutExisting(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Git(t, dir, "branch", "existing", "main")
	gittest.Chdir(t, dir)

	path, err := AddWorktree(WorktreeSpec{Name: "wt", Branch: "existing"})
	if err != nil {
		t.Fatal(err)
	}
	if b := currentBranch(t, path); b != "existing" {
		t.Fatalf("worktree on branch %q, want existing", b)
	}
}

func TestAddWorktreeFromRemoteTracks(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.AddRemote(t, dir)
	gittest.RemoteOnlyBranch(t, dir, "colleague", "main")
	gittest.Chdir(t, dir)

	path, err := AddWorktree(WorktreeSpec{Name: "col", Branch: "colleague", CreateFrom: "origin/colleague"})
	if err != nil {
		t.Fatal(err)
	}
	if b := currentBranch(t, path); b != "colleague" {
		t.Fatalf("worktree on branch %q, want colleague", b)
	}
	up := strings.TrimSpace(gittest.Git(t, path, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}"))
	if up != "origin/colleague" {
		t.Fatalf("upstream = %q, want origin/colleague", up)
	}
}

func TestAddWorktreeDuplicateErrors(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Chdir(t, dir)
	if _, err := AddWorktree(WorktreeSpec{Name: "dup", Branch: "dup", CreateFrom: "main"}); err != nil {
		t.Fatal(err)
	}
	if _, err := AddWorktree(WorktreeSpec{Name: "dup", Branch: "dup", CreateFrom: "main"}); err == nil {
		t.Fatal("expected error creating a duplicate worktree, got nil")
	}
}

func TestRemoveWorktree(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Chdir(t, dir)
	path, err := AddWorktree(WorktreeSpec{Name: "gone", Branch: "gone", CreateFrom: "main"})
	if err != nil {
		t.Fatal(err)
	}
	if err := RemoveWorktree(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("worktree dir should be gone, stat err = %v", err)
	}
}

func TestForceRemoveWorktreeDirty(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Chdir(t, dir)
	path, err := AddWorktree(WorktreeSpec{Name: "dirty", Branch: "dirty", CreateFrom: "main"})
	if err != nil {
		t.Fatal(err)
	}
	// Modify a tracked file so a plain remove would refuse.
	if err := os.WriteFile(filepath.Join(path, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RemoveWorktree(path); err == nil {
		t.Fatal("plain remove should refuse a dirty worktree")
	}
	if err := ForceRemoveWorktree(path); err != nil {
		t.Fatalf("force remove failed: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("worktree dir should be gone after force remove")
	}
}

func TestFindWorktree(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Chdir(t, dir)
	if _, err := AddWorktree(WorktreeSpec{Name: "feature-a", Branch: "feature-a", CreateFrom: "main"}); err != nil {
		t.Fatal(err)
	}
	if _, err := AddWorktree(WorktreeSpec{Name: "feature-b", Branch: "feature-b", CreateFrom: "main"}); err != nil {
		t.Fatal(err)
	}
	if _, err := AddWorktree(WorktreeSpec{Name: "docs", Branch: "docs", CreateFrom: "main"}); err != nil {
		t.Fatal(err)
	}

	// Exact.
	if wt, err := FindWorktree("feature-a"); err != nil || wt.Name != "feature-a" {
		t.Fatalf("exact match failed: wt=%v err=%v", wt, err)
	}
	// Unique substring.
	if wt, err := FindWorktree("docs"); err != nil || wt.Name != "docs" {
		t.Fatalf("substring match failed: err=%v", err)
	}
	// Ambiguous prefix.
	if _, err := FindWorktree("feature"); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguous error, got %v", err)
	}
	// Not found.
	if _, err := FindWorktree("nope"); err == nil {
		t.Fatalf("expected not-found error")
	}
}

func TestMainWorktreePath(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Chdir(t, dir)
	if _, err := AddWorktree(WorktreeSpec{Name: "side", Branch: "side", CreateFrom: "main"}); err != nil {
		t.Fatal(err)
	}
	p, err := MainWorktreePath()
	if err != nil {
		t.Fatal(err)
	}
	if p != dir {
		t.Fatalf("MainWorktreePath = %q, want %q", p, dir)
	}
}

// --- helpers ---------------------------------------------------------------

func worktreeByName(t *testing.T, wts []Worktree, name string) Worktree {
	t.Helper()
	for _, wt := range wts {
		if wt.Name == name {
			return wt
		}
	}
	t.Fatalf("no worktree named %q in %+v", name, wts)
	return Worktree{}
}

func currentBranch(t *testing.T, dir string) string {
	t.Helper()
	return strings.TrimSpace(gittest.Git(t, dir, "rev-parse", "--abbrev-ref", "HEAD"))
}
