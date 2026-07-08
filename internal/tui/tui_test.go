package tui

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sushi/worktree-manager/internal/git"
	"github.com/sushi/worktree-manager/internal/gittest"
)

// --- key-driving helpers ----------------------------------------------------

func send(t *testing.T, m InteractiveModel, msg tea.Msg) InteractiveModel {
	t.Helper()
	nm, _ := m.Update(msg)
	return nm.(InteractiveModel)
}

func press(t *testing.T, m InteractiveModel, typ tea.KeyType) InteractiveModel {
	return send(t, m, tea.KeyMsg{Type: typ})
}

func update(t *testing.T, m InteractiveModel, msg tea.Msg) (InteractiveModel, tea.Cmd) {
	t.Helper()
	nm, cmd := m.Update(msg)
	return nm.(InteractiveModel), cmd
}

func typeRune(t *testing.T, m InteractiveModel, s string) InteractiveModel {
	for _, r := range s {
		m = send(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return m
}

var demoBranches = []git.Branch{
	{Name: "main", IsDefault: true, IsCurrent: true},
	{Name: "feature"},
	{Name: "origin/colleague", IsRemote: true},
}

// atBranchPick returns a model sized and advanced to the first branch-pick step.
func atBranchPick(t *testing.T) InteractiveModel {
	t.Helper()
	m := NewInteractive(nil)
	m = send(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = typeRune(t, m, "a") // open add flow
	m = send(t, m, branchesMsg{list: demoBranches})
	if m.mode != modeBranchPick {
		t.Fatalf("want modeBranchPick, got %d", m.mode)
	}
	return m
}

// --- BranchItem -------------------------------------------------------------

func TestBranchItemTitle(t *testing.T) {
	cases := []struct {
		item BranchItem
		want string
	}{
		{BranchItem{create: true}, createBranchLabel},
		{BranchItem{branch: git.Branch{Name: "main", IsDefault: true}}, "main (default)"},
		{BranchItem{branch: git.Branch{Name: "main", IsCurrent: true}}, "main *"},
		{BranchItem{branch: git.Branch{Name: "origin/x", IsRemote: true}}, "origin/x (remote)"},
	}
	for _, c := range cases {
		if got := c.item.Title(); got != c.want {
			t.Errorf("Title() = %q, want %q", got, c.want)
		}
	}
	if got := (BranchItem{create: true}).FilterValue(); got != createBranchLabel {
		t.Errorf("create FilterValue = %q", got)
	}
	if got := (BranchItem{branch: git.Branch{Name: "feat"}}).FilterValue(); got != "feat" {
		t.Errorf("branch FilterValue = %q", got)
	}
}

// --- newBranchList default selection ----------------------------------------

func TestNewBranchListDefaultSelection(t *testing.T) {
	bs := []git.Branch{{Name: "main", IsDefault: true}, {Name: "feature"}}

	withCreate := newBranchList(bs, true, 80, 20)
	if item, ok := withCreate.Items()[0].(BranchItem); !ok || !item.create {
		t.Fatal("item 0 should be the create entry")
	}
	if withCreate.Index() != 1 {
		t.Fatalf("cursor should start on default (index 1), got %d", withCreate.Index())
	}
	if sel := withCreate.SelectedItem().(BranchItem); sel.branch.Name != "main" {
		t.Fatalf("selected = %q, want main", sel.branch.Name)
	}

	noCreate := newBranchList(bs, false, 80, 20)
	if noCreate.Index() != 0 || noCreate.SelectedItem().(BranchItem).branch.Name != "main" {
		t.Fatalf("base picker should select main at index 0, got %d", noCreate.Index())
	}
}

// --- add flow: three paths --------------------------------------------------

func TestAddFlowCreateNewBranch(t *testing.T) {
	m := atBranchPick(t)
	// Create entry is index 0; it is not the cursor default, so navigate to it.
	m = press(t, m, tea.KeyHome) // ensure top
	m = press(t, m, tea.KeyEnter)
	if m.mode != modeBasePick || !m.creatingBranch {
		t.Fatalf("want modeBasePick+creating, got mode=%d creating=%v", m.mode, m.creatingBranch)
	}
	// Base list (no create entry) starts on default → pick main.
	m = press(t, m, tea.KeyEnter)
	if m.mode != modeNewBranch || m.pendingBase != "main" {
		t.Fatalf("want modeNewBranch base=main, got mode=%d base=%q", m.mode, m.pendingBase)
	}
	m = typeRune(t, m, "shiny")
	m = press(t, m, tea.KeyEnter)
	if m.mode != modeName || m.pendingBranch != "shiny" {
		t.Fatalf("want modeName branch=shiny, got mode=%d branch=%q", m.mode, m.pendingBranch)
	}
	// Empty name defaults to the branch name.
	m = press(t, m, tea.KeyEnter)
	if m.mode != modeSyncPrompt || m.pendingName != "shiny" {
		t.Fatalf("want modeSyncPrompt name=shiny, got mode=%d name=%q", m.mode, m.pendingName)
	}
	if !m.creatingBranch || m.pendingBase != "main" {
		t.Fatalf("final spec wrong: creating=%v base=%q", m.creatingBranch, m.pendingBase)
	}
}

func TestAddFlowCheckoutExistingLocal(t *testing.T) {
	m := atBranchPick(t)
	// Default cursor is on main (index 1). Select it directly.
	m = press(t, m, tea.KeyEnter)
	if m.mode != modeName || m.creatingBranch {
		t.Fatalf("want modeName not-creating, got mode=%d creating=%v", m.mode, m.creatingBranch)
	}
	if m.pendingBranch != "main" || m.pendingBase != "" {
		t.Fatalf("want branch=main base=empty, got branch=%q base=%q", m.pendingBranch, m.pendingBase)
	}
	// A typed name overrides the default.
	m = typeRune(t, m, "custom")
	m = press(t, m, tea.KeyEnter)
	if m.pendingName != "custom" || m.mode != modeSyncPrompt {
		t.Fatalf("want name=custom syncPrompt, got name=%q mode=%d", m.pendingName, m.mode)
	}
}

func TestAddFlowCheckoutRemote(t *testing.T) {
	m := atBranchPick(t)
	m = press(t, m, tea.KeyDown) // main(1) -> feature(2)
	m = press(t, m, tea.KeyDown) // feature(2) -> origin/colleague(3)
	m = press(t, m, tea.KeyEnter)
	if m.pendingBranch != "colleague" || m.pendingBase != "origin/colleague" {
		t.Fatalf("remote checkout: branch=%q base=%q", m.pendingBranch, m.pendingBase)
	}
	if got := defaultWorktreeName(m.pendingBranch); got != "colleague" {
		t.Fatalf("default name = %q, want colleague", got)
	}
}

func TestAddFlowEscNavigation(t *testing.T) {
	m := atBranchPick(t)
	// Into create → base → new branch, then walk all the way back out.
	m = press(t, m, tea.KeyHome)
	m = press(t, m, tea.KeyEnter) // create -> basePick
	m = press(t, m, tea.KeyEnter) // basePick -> newBranch
	if m.mode != modeNewBranch {
		t.Fatalf("setup: want modeNewBranch, got %d", m.mode)
	}
	m = press(t, m, tea.KeyEsc) // newBranch -> basePick
	if m.mode != modeBasePick {
		t.Fatalf("esc from newBranch: want modeBasePick, got %d", m.mode)
	}
	m = press(t, m, tea.KeyEsc) // basePick -> branchPick (create entry present again)
	if m.mode != modeBranchPick {
		t.Fatalf("esc from basePick: want modeBranchPick, got %d", m.mode)
	}
	if item, ok := m.branchList.Items()[0].(BranchItem); !ok || !item.create {
		t.Fatalf("branchPick should restore the create entry")
	}
	m = press(t, m, tea.KeyEsc) // branchPick -> list
	if m.mode != modeList {
		t.Fatalf("esc from branchPick: want modeList, got %d", m.mode)
	}
}

func TestAddFlowEscFromNameGoesBack(t *testing.T) {
	// Existing-branch path: name step should esc back to the branch pick.
	m := atBranchPick(t)
	m = press(t, m, tea.KeyEnter) // select main -> name
	m = press(t, m, tea.KeyEsc)
	if m.mode != modeBranchPick {
		t.Fatalf("existing path esc: want modeBranchPick, got %d", m.mode)
	}

	// Create path: name step should esc back to the new-branch input.
	m = atBranchPick(t)
	m = press(t, m, tea.KeyHome)
	m = press(t, m, tea.KeyEnter) // create -> basePick
	m = press(t, m, tea.KeyEnter) // basePick -> newBranch
	m = typeRune(t, m, "x")
	m = press(t, m, tea.KeyEnter) // newBranch -> name
	m = press(t, m, tea.KeyEsc)
	if m.mode != modeNewBranch {
		t.Fatalf("create path esc: want modeNewBranch, got %d", m.mode)
	}
}

func TestNewBranchRequiresName(t *testing.T) {
	m := atBranchPick(t)
	m = press(t, m, tea.KeyHome)
	m = press(t, m, tea.KeyEnter) // create -> basePick
	m = press(t, m, tea.KeyEnter) // basePick -> newBranch
	m = press(t, m, tea.KeyEnter) // empty name: should stay put
	if m.mode != modeNewBranch {
		t.Fatalf("empty new-branch name should not advance, got mode %d", m.mode)
	}
}

func TestBranchFetchErrorStaysInPicker(t *testing.T) {
	m := atBranchPick(t)
	m.loading = true
	m = send(t, m, branchFetchErrMsg{err: os.ErrPermission})
	if m.mode != modeBranchPick {
		t.Fatalf("fetch error should keep modeBranchPick, got %d", m.mode)
	}
	if m.err == nil {
		t.Fatalf("fetch error should be surfaced")
	}
}

// --- list mode --------------------------------------------------------------

func listModelWith(t *testing.T, wts []git.Worktree) InteractiveModel {
	t.Helper()
	m := NewInteractive(wts)
	return send(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
}

func TestListSwitchOnEnter(t *testing.T) {
	wts := []git.Worktree{
		{Name: "main-wt", Path: "/repo", IsMain: true, IsCurrent: true},
		{Name: "feature", Path: "/repo--feature"},
	}
	m := listModelWith(t, wts)
	m = press(t, m, tea.KeyDown) // move to feature
	m = press(t, m, tea.KeyEnter)
	if !m.quitting || m.SwitchTo() != "/repo--feature" {
		t.Fatalf("enter should switch: quitting=%v switchTo=%q", m.quitting, m.SwitchTo())
	}
}

func TestListSwitchToCurrentIsNoop(t *testing.T) {
	wts := []git.Worktree{{Name: "main-wt", Path: "/repo", IsMain: true, IsCurrent: true}}
	m := listModelWith(t, wts)
	m = press(t, m, tea.KeyEnter) // current worktree
	if m.quitting {
		t.Fatal("switching to the current worktree should not quit")
	}
	if m.message == "" {
		t.Fatal("expected an 'already here' message")
	}
}

func TestListDeleteConfirmAndCancel(t *testing.T) {
	wts := []git.Worktree{
		{Name: "main-wt", Path: "/repo", IsMain: true, IsCurrent: true},
		{Name: "feature", Path: "/repo--feature"},
	}
	m := listModelWith(t, wts)
	m = press(t, m, tea.KeyDown) // select feature
	m = typeRune(t, m, "d")
	if m.mode != modeConfirmDelete || m.selected == nil || m.selected.Name != "feature" {
		t.Fatalf("d should open delete confirm for feature, got mode=%d selected=%v", m.mode, m.selected)
	}
	m = typeRune(t, m, "n") // cancel
	if m.mode != modeList || m.selected != nil {
		t.Fatalf("n should cancel delete, got mode=%d", m.mode)
	}
}

func TestListDeleteMainRejected(t *testing.T) {
	wts := []git.Worktree{{Name: "main-wt", Path: "/repo", IsMain: true, IsCurrent: true}}
	m := listModelWith(t, wts)
	m = typeRune(t, m, "d")
	if m.mode == modeConfirmDelete {
		t.Fatal("deleting the main worktree should be rejected")
	}
	if m.message == "" {
		t.Fatal("expected a rejection message for main worktree")
	}
}

func TestQuitFromList(t *testing.T) {
	m := listModelWith(t, []git.Worktree{{Name: "main-wt", Path: "/repo", IsMain: true}})
	m = typeRune(t, m, "q")
	if !m.quitting || m.SwitchTo() != "" {
		t.Fatalf("q should quit without switching")
	}
}

// --- helpers ----------------------------------------------------------------

func TestDefaultWorktreeName(t *testing.T) {
	cases := map[string]string{
		"feature":     "feature",
		"feature/foo": "feature-foo",
		"a/b/c":       "a-b-c",
	}
	for in, want := range cases {
		if got := defaultWorktreeName(in); got != want {
			t.Errorf("defaultWorktreeName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSyncPromptTitle(t *testing.T) {
	m := InteractiveModel{pendingName: "wt", pendingBranch: "br", creatingBranch: true}
	if got := m.syncPromptTitle(); !strings.Contains(got, "new branch 'br'") {
		t.Errorf("creating title = %q", got)
	}
	m.creatingBranch = false
	if got := m.syncPromptTitle(); !strings.Contains(got, "on branch 'br'") || strings.Contains(got, "new branch") {
		t.Errorf("checkout title = %q", got)
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("hello", 10); got != "hello" {
		t.Errorf("no truncation expected, got %q", got)
	}
	if got := truncate("hello world", 5); got != "hell…" {
		t.Errorf("truncate = %q, want 'hell…'", got)
	}
	if got := truncate("x", 0); got != "" {
		t.Errorf("truncate to 0 = %q, want empty", got)
	}
}

// --- view smoke tests -------------------------------------------------------

func TestViewsRenderWithoutPanic(t *testing.T) {
	m := atBranchPick(t)
	if !strings.Contains(m.View(), "Select branch") {
		t.Errorf("branch pick view missing header")
	}
	// Walk through the create flow, rendering at each stage.
	m = press(t, m, tea.KeyHome)
	m = press(t, m, tea.KeyEnter) // basePick
	if !strings.Contains(m.View(), "Base branch") {
		t.Errorf("base pick view missing header")
	}
	m = press(t, m, tea.KeyEnter) // newBranch
	if !strings.Contains(m.View(), "New branch") {
		t.Errorf("new branch view missing header")
	}
	m = typeRune(t, m, "z")
	m = press(t, m, tea.KeyEnter) // name
	if !strings.Contains(m.View(), "Name the worktree") {
		t.Errorf("name view missing prompt")
	}
	m = press(t, m, tea.KeyEnter) // syncPrompt
	if !strings.Contains(m.View(), "Copy gitignored files") {
		t.Errorf("sync prompt view missing text")
	}
}

func TestListViewShowsWorktree(t *testing.T) {
	wts := []git.Worktree{{Name: "feature", Branch: "feature", Path: "/repo--feature", Head: "abc1234"}}
	m := listModelWith(t, wts)
	out := m.View()
	if !strings.Contains(out, "Git Worktrees") || !strings.Contains(out, "feature") {
		t.Errorf("list view missing expected content:\n%s", out)
	}
}

// --- standalone branch picker ----------------------------------------------

func TestBranchPickerSelect(t *testing.T) {
	m := NewBranchPicker([]git.Branch{{Name: "main", IsDefault: true}, {Name: "feature"}})
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = nm.(BranchPickerModel)
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(BranchPickerModel)
	if m.Selected() != "main" {
		t.Fatalf("selected = %q, want main (default)", m.Selected())
	}
}

func TestBranchPickerFetchState(t *testing.T) {
	m := NewBranchPicker([]git.Branch{{Name: "main", IsDefault: true}})
	nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	m = nm.(BranchPickerModel)
	if !m.fetching || cmd == nil {
		t.Fatalf("f should enter fetching state with a command")
	}
	// Delivering the reload result exits the fetching state and updates items.
	nm, _ = m.Update(branchFetchMsg{list: []git.Branch{{Name: "main", IsDefault: true}, {Name: "new"}}})
	m = nm.(BranchPickerModel)
	if m.fetching {
		t.Fatal("fetch result should clear fetching state")
	}
	if len(m.list.Items()) != 2 {
		t.Fatalf("reloaded list should have 2 items, got %d", len(m.list.Items()))
	}
}

func TestBranchPickerQuitEmpty(t *testing.T) {
	m := NewBranchPicker([]git.Branch{{Name: "main"}})
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = nm.(BranchPickerModel)
	if m.Selected() != "" {
		t.Fatalf("esc should select nothing, got %q", m.Selected())
	}
}

func TestAlignColorToStderrNoPanic(t *testing.T) {
	alignColorToStderr() // must not panic
}

// --- integration: the tui's git-backed commands -----------------------------

func TestLoadBranchesCmdIntegration(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Git(t, dir, "branch", "feature")
	gittest.Chdir(t, dir)

	msg, ok := loadBranchesCmd().(branchesMsg)
	if !ok {
		t.Fatalf("loadBranchesCmd returned %T", msg)
	}
	if msg.err != nil {
		t.Fatal(msg.err)
	}
	var names []string
	for _, b := range msg.list {
		names = append(names, b.Name)
	}
	if len(names) != 2 {
		t.Fatalf("branches = %v, want main+feature", names)
	}
}

func TestCreateWorktreeCmdIntegration(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Chdir(t, dir)

	msg := createWorktreeCmd(git.WorktreeSpec{Name: "made", Branch: "made", CreateFrom: "main"}, false)()
	created, ok := msg.(createdMsg)
	if !ok {
		t.Fatalf("createWorktreeCmd returned %T", msg)
	}
	if created.err != nil {
		t.Fatal(created.err)
	}
	if _, err := os.Stat(created.path); err != nil {
		t.Fatalf("created worktree missing: %v", err)
	}
}

func TestRemoveWorktreeCmdIntegration(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Chdir(t, dir)
	path, err := git.AddWorktree(git.WorktreeSpec{Name: "temp", Branch: "temp", CreateFrom: "main"})
	if err != nil {
		t.Fatal(err)
	}
	wt := git.Worktree{Name: "temp", Path: path}
	msg := removeWorktreeCmd(wt)().(removedMsg)
	if msg.err != nil {
		t.Fatalf("remove cmd error: %v", msg.err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("worktree should be removed")
	}
}

func TestLoadWorktreesCmdIntegration(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Chdir(t, dir)
	msg := loadWorktreesCmd().(worktreesMsg)
	if msg.err != nil {
		t.Fatal(msg.err)
	}
	if len(msg.list) != 1 || !msg.list[0].IsMain {
		t.Fatalf("expected a single main worktree, got %+v", msg.list)
	}
}

func TestFetchBranchesCmdIntegration(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.AddRemote(t, dir)
	gittest.RemoteOnlyBranch(t, dir, "colleague", "main")
	gittest.Chdir(t, dir)

	msg, ok := fetchBranchesCmd()().(branchesMsg)
	if !ok {
		t.Fatalf("fetchBranchesCmd returned %T", msg)
	}
	if msg.err != nil {
		t.Fatal(msg.err)
	}
	var found bool
	for _, b := range msg.list {
		if b.Name == "origin/colleague" {
			found = true
		}
	}
	if !found {
		t.Fatal("fetch+reload should surface origin/colleague")
	}
}

func TestPickerFetchCmdIntegration(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.AddRemote(t, dir)
	gittest.Chdir(t, dir)
	msg, ok := pickerFetchCmd().(branchFetchMsg)
	if !ok {
		t.Fatalf("pickerFetchCmd returned %T", msg)
	}
	if msg.err != nil {
		t.Fatalf("pickerFetchCmd error: %v", msg.err)
	}
}

// --- additional coverage ----------------------------------------------------

func TestWorktreeItemInterface(t *testing.T) {
	it := WorktreeItem{wt: git.Worktree{Name: "feat", Branch: "feature"}}
	if it.Title() != "feat" {
		t.Errorf("Title = %q", it.Title())
	}
	if it.Description() != "feature" {
		t.Errorf("Description = %q", it.Description())
	}
	if it.FilterValue() != "feat feature" {
		t.Errorf("FilterValue = %q", it.FilterValue())
	}
	// Branch items also expose an (empty) Description for the list interface.
	if (BranchItem{}).Description() != "" {
		t.Errorf("BranchItem.Description should be empty")
	}
}

func TestUpdatePassthroughDoesNotPanic(t *testing.T) {
	// A message that Update does not explicitly handle is routed to the active
	// component via updateCurrentMode. Exercise a few modes.
	m := listModelWith(t, []git.Worktree{{Name: "a", Path: "/a", IsMain: true}})
	m, _ = update(t, m, 42) // arbitrary tea.Msg while in modeList

	m = atBranchPick(t)
	m, _ = update(t, m, 42) // modeBranchPick

	m = press(t, m, tea.KeyEnter) // -> modeName
	m, _ = update(t, m, 42)       // modeName input passthrough
}

func TestDeleteConfirmYesStartsRemoval(t *testing.T) {
	wts := []git.Worktree{
		{Name: "main-wt", Path: "/repo", IsMain: true, IsCurrent: true},
		{Name: "feature", Path: "/repo--feature"},
	}
	m := listModelWith(t, wts)
	m = press(t, m, tea.KeyDown)
	m = typeRune(t, m, "d")
	if m.mode != modeConfirmDelete {
		t.Fatalf("setup: want modeConfirmDelete, got %d", m.mode)
	}
	m, cmd := update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if !m.loading || cmd == nil {
		t.Fatalf("y should start a removal (loading + cmd), got loading=%v cmd=%v", m.loading, cmd)
	}
}

func TestSyncPromptKeys(t *testing.T) {
	reach := func() InteractiveModel {
		m := atBranchPick(t)
		m = press(t, m, tea.KeyEnter) // select main -> name
		m = press(t, m, tea.KeyEnter) // empty name -> syncPrompt
		if m.mode != modeSyncPrompt {
			t.Fatalf("setup: want modeSyncPrompt, got %d", m.mode)
		}
		return m
	}

	// esc goes back to the name step.
	m := reach()
	m = press(t, m, tea.KeyEsc)
	if m.mode != modeName {
		t.Fatalf("esc from syncPrompt: want modeName, got %d", m.mode)
	}

	// 'n' skips copying and starts creation.
	m = reach()
	m, cmd := update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if !m.loading || cmd == nil {
		t.Fatalf("n should start worktree creation")
	}

	// 'y' copies ignored files and starts creation.
	m = reach()
	m, cmd = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if !m.loading || cmd == nil {
		t.Fatalf("y should start worktree creation")
	}
}

func TestFullWorktreeRowRenders(t *testing.T) {
	wts := []git.Worktree{{
		Name: "feature", Branch: "feature", Path: "/repo--feature", Head: "abc1234",
		HasUpstream: true, Ahead: 2, Behind: 1, DirtyCount: 3,
		LastCommit: "2 hours ago", Subject: "did some work", IsCurrent: true,
	}}
	m := listModelWith(t, wts)
	out := m.View()
	for _, want := range []string{"feature", "abc1234", "↑2", "↓1", "✱3", "2 hours ago", "did some work"} {
		if !strings.Contains(out, want) {
			t.Errorf("worktree row missing %q:\n%s", want, out)
		}
	}
}

func TestBranchPickerView(t *testing.T) {
	m := NewBranchPicker([]git.Branch{{Name: "main", IsDefault: true}})
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = nm.(BranchPickerModel)
	if !strings.Contains(m.View(), "Select base branch") {
		t.Errorf("picker view missing header")
	}
	// After selecting, the view shows the confirmation line.
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(BranchPickerModel)
	if !strings.Contains(m.View(), "Base branch:") {
		t.Errorf("picker confirmation view missing")
	}
}
