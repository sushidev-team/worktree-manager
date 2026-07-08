package git

import (
	"testing"

	"github.com/sushi/worktree-manager/internal/gittest"
)

func branchNames(bs []Branch) []string {
	names := make([]string, len(bs))
	for i, b := range bs {
		names[i] = b.Name
	}
	return names
}

func findBranch(bs []Branch, name string) (Branch, bool) {
	for _, b := range bs {
		if b.Name == name {
			return b, true
		}
	}
	return Branch{}, false
}

func TestDefaultBranchMain(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Chdir(t, dir)
	if got := DefaultBranch(); got != "main" {
		t.Fatalf("DefaultBranch() = %q, want main", got)
	}
}

func TestDefaultBranchMasterFallback(t *testing.T) {
	dir := gittest.RepoOnBranch(t, "master")
	gittest.Chdir(t, dir)
	if got := DefaultBranch(); got != "master" {
		t.Fatalf("DefaultBranch() = %q, want master", got)
	}
}

func TestDefaultBranchFromRemoteHead(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.AddRemote(t, dir)
	gittest.Chdir(t, dir)
	if got := DefaultBranch(); got != "main" {
		t.Fatalf("DefaultBranch() = %q, want main", got)
	}
}

func TestListBranchesLocalOnly(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Git(t, dir, "branch", "zeta")
	gittest.Git(t, dir, "branch", "alpha")
	gittest.Chdir(t, dir)

	bs, err := ListBranches()
	if err != nil {
		t.Fatal(err)
	}
	// Default branch first, then remaining locals alphabetically.
	got := branchNames(bs)
	want := []string{"main", "alpha", "zeta"}
	if len(got) != len(want) {
		t.Fatalf("branches = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("branch order = %v, want %v", got, want)
		}
	}

	main, _ := findBranch(bs, "main")
	if !main.IsDefault {
		t.Errorf("main should be marked default")
	}
	if !main.IsCurrent {
		t.Errorf("main should be marked current (checked out)")
	}
}

func TestListBranchesIncludesRemoteOnly(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.AddRemote(t, dir)
	gittest.RemoteOnlyBranch(t, dir, "colleague", "main")
	gittest.Chdir(t, dir)

	bs, err := ListBranches()
	if err != nil {
		t.Fatal(err)
	}

	// origin/colleague has no local counterpart → shown as a remote branch.
	col, ok := findBranch(bs, "origin/colleague")
	if !ok {
		t.Fatalf("expected origin/colleague in %v", branchNames(bs))
	}
	if !col.IsRemote {
		t.Errorf("origin/colleague should have IsRemote=true")
	}

	// origin/main duplicates the local main → deduped away.
	if _, ok := findBranch(bs, "origin/main"); ok {
		t.Errorf("origin/main should be deduped against local main")
	}
	// The bare remote HEAD symref must never appear.
	if _, ok := findBranch(bs, "origin"); ok {
		t.Errorf("bare remote name should be filtered out")
	}

	// Remote branches sort after all local branches.
	names := branchNames(bs)
	lastLocal, firstRemote := -1, len(names)
	for i, b := range bs {
		if b.IsRemote && i < firstRemote {
			firstRemote = i
		}
		if !b.IsRemote {
			lastLocal = i
		}
	}
	if lastLocal > firstRemote {
		t.Errorf("remote branch appears before a local one: %v", names)
	}
}

func TestFetchBringsNewRefAndPrunes(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.AddRemote(t, dir)

	// Publish a branch, then forget it locally (both the branch and the
	// remote-tracking ref) so only a Fetch can rediscover it.
	gittest.Git(t, dir, "branch", "col", "main")
	gittest.Git(t, dir, "push", "origin", "col")
	gittest.Git(t, dir, "branch", "-D", "col")
	gittest.Git(t, dir, "branch", "-rd", "origin/col")

	gittest.Chdir(t, dir)
	if _, ok := findBranch(mustBranches(t), "origin/col"); ok {
		t.Fatalf("origin/col should be unknown before fetch")
	}

	if err := Fetch(); err != nil {
		t.Fatalf("Fetch() error: %v", err)
	}
	if _, ok := findBranch(mustBranches(t), "origin/col"); !ok {
		t.Fatalf("origin/col should be known after fetch")
	}

	// Delete it upstream; a pruning fetch should drop the tracking ref.
	gittest.Git(t, dir, "push", "origin", "--delete", "col")
	if err := Fetch(); err != nil {
		t.Fatalf("Fetch() (prune) error: %v", err)
	}
	if _, ok := findBranch(mustBranches(t), "origin/col"); ok {
		t.Fatalf("origin/col should be pruned after upstream deletion")
	}
}

func mustBranches(t *testing.T) []Branch {
	t.Helper()
	bs, err := ListBranches()
	if err != nil {
		t.Fatal(err)
	}
	return bs
}

func TestSortBranchesOrdering(t *testing.T) {
	bs := []Branch{
		{Name: "origin/x", IsRemote: true},
		{Name: "beta"},
		{Name: "main", IsDefault: true},
		{Name: "alpha", IsCurrent: true},
	}
	sortBranches(bs)
	got := branchNames(bs)
	want := []string{"main", "alpha", "beta", "origin/x"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sortBranches order = %v, want %v", got, want)
		}
	}
}
