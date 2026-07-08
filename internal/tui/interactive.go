package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sushi/worktree-manager/internal/git"
)

type mode int

const (
	modeList mode = iota
	modeAdd
	modeConfirmDelete
	modeBranchPick
	modeSyncPrompt
)

// chromeHeight is the number of rows the header, footer and their spacers
// consume; the list/body gets the rest.
const chromeHeight = 4

// --- async messages ---------------------------------------------------------

type worktreesMsg struct {
	list []git.Worktree
	err  error
}
type branchesMsg struct {
	name string // worktree name the branch is being picked for
	list []git.Branch
	err  error
}
type createdMsg struct {
	path string
	err  error
}

// branchFetchErrMsg reports a failed `git fetch` without leaving the branch
// picker, so a typed worktree name and the current selection are preserved.
type branchFetchErrMsg struct {
	err error
}
type removedMsg struct {
	name string
	err  error
}

func loadWorktreesCmd() tea.Msg {
	wts, err := git.ListWorktrees()
	return worktreesMsg{list: wts, err: err}
}

func loadBranchesCmd(name string) tea.Cmd {
	return func() tea.Msg {
		bs, err := git.ListBranches()
		return branchesMsg{name: name, list: bs, err: err}
	}
}

// fetchBranchesCmd fetches from all remotes, then reloads the branch list so
// newly-fetched remote branches appear. A fetch failure keeps the picker open
// with its existing branches (branchFetchErrMsg); a reload failure falls back
// to the normal branchesMsg error handling.
func fetchBranchesCmd(name string) tea.Cmd {
	return func() tea.Msg {
		if err := git.Fetch(); err != nil {
			return branchFetchErrMsg{err: err}
		}
		bs, err := git.ListBranches()
		return branchesMsg{name: name, list: bs, err: err}
	}
}

func createWorktreeCmd(name, base string, syncIgnored bool) tea.Cmd {
	return func() tea.Msg {
		path, err := git.AddWorktree(name, base)
		if err == nil && syncIgnored {
			if src, e := git.MainWorktreePath(); e == nil {
				_, err = git.SyncIgnoredFiles(src, path)
			} else {
				err = e
			}
		}
		return createdMsg{path: path, err: err}
	}
}

func removeWorktreeCmd(wt git.Worktree) tea.Cmd {
	return func() tea.Msg {
		var err error
		if wt.IsDirty {
			err = git.ForceRemoveWorktree(wt.Path)
		} else {
			err = git.RemoveWorktree(wt.Path)
		}
		return removedMsg{name: wt.Name, err: err}
	}
}

// --- list items -------------------------------------------------------------

// WorktreeItem implements list.Item for the worktree list.
type WorktreeItem struct {
	wt git.Worktree
}

func (w WorktreeItem) Title() string       { return w.wt.Name }
func (w WorktreeItem) Description() string { return w.wt.Branch }
func (w WorktreeItem) FilterValue() string { return w.wt.Name + " " + w.wt.Branch }

// --- model ------------------------------------------------------------------

// InteractiveModel is the main TUI model for the interactive worktree manager.
type InteractiveModel struct {
	list          list.Model
	spinner       spinner.Model
	mode          mode
	addInput      textinput.Model
	branchList    list.Model
	pendingName   string // worktree name awaiting branch + sync choice
	pendingBranch string // base branch chosen, awaiting sync confirmation
	selected      *git.Worktree
	message       string
	err           error
	loading       bool
	loadingMsg    string
	width         int
	height        int
	quitting      bool
	switchTo      string // path to switch to after quitting
}

// NewInteractive creates the main interactive TUI model with a pre-loaded
// worktree list, so the list is visible and interactive on the first frame.
// Mutations (add/remove) load asynchronously with a spinner (see startLoading).
func NewInteractive(worktrees []git.Worktree) InteractiveModel {
	items := make([]list.Item, len(worktrees))
	for i, wt := range worktrees {
		items[i] = WorktreeItem{wt: wt}
	}

	l := list.New(items, worktreeDelegate{}, 70, 20)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.FilterInput.PromptStyle = BranchStyle
	l.FilterInput.Cursor.Style = BranchStyle

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = SpinnerStyle

	ti := textinput.New()
	ti.Placeholder = "worktree-name"
	ti.CharLimit = 100
	ti.Width = 40
	ti.Prompt = ""

	return InteractiveModel{
		list:       l,
		branchList: newBranchList(nil, 70, 20), // valid empty model; replaced on add
		spinner:    sp,
		addInput:   ti,
	}
}

func (m InteractiveModel) Init() tea.Cmd {
	return textinput.Blink
}

// startLoading enters the loading state with a message and kicks off cmd,
// keeping the spinner animating.
func (m InteractiveModel) startLoading(msg string, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	m.loading = true
	m.loadingMsg = msg
	m.message = ""
	m.err = nil
	return m, tea.Batch(m.spinner.Tick, cmd)
}

func (m *InteractiveModel) resize() {
	h := m.height - chromeHeight
	if m.width <= 0 || h <= 0 {
		return
	}
	m.list.SetSize(m.width, h)
	m.branchList.SetSize(m.width, h)
}

func (m InteractiveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		return m, nil

	case spinner.TickMsg:
		if !m.loading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case worktreesMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		items := make([]list.Item, len(msg.list))
		for i, wt := range msg.list {
			items[i] = WorktreeItem{wt: wt}
		}
		m.list.SetItems(items)
		m.mode = modeList
		m.selected = nil
		return m, nil

	case branchesMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.mode = modeList
			return m, nil
		}
		m.branchList = newBranchList(msg.list, m.width, m.height-chromeHeight)
		m.mode = modeBranchPick
		return m, nil

	case branchFetchErrMsg:
		// Stay in the branch picker with the branches we already have.
		m.loading = false
		m.err = msg.err
		return m, nil

	case createdMsg:
		if msg.err != nil {
			m.loading = false
			m.err = msg.err
			m.mode = modeList
			return m, nil
		}
		m.switchTo = msg.path
		m.quitting = true
		return m, tea.Quit

	case removedMsg:
		if msg.err != nil {
			m.loading = false
			m.err = msg.err
			m.mode = modeList
			return m, nil
		}
		m.message = fmt.Sprintf("Removed worktree '%s'", msg.name)
		m.loadingMsg = "Refreshing…"
		return m, loadWorktreesCmd

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m.updateCurrentMode(msg)
}

func (m InteractiveModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// While a git operation runs, only allow quitting.
	if m.loading {
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil
	}

	switch m.mode {
	case modeList:
		return m.handleListKey(msg)
	case modeAdd:
		return m.handleAddKey(msg)
	case modeConfirmDelete:
		return m.handleDeleteKey(msg)
	case modeBranchPick:
		return m.handleBranchPickKey(msg)
	case modeSyncPrompt:
		return m.handleSyncPromptKey(msg)
	}
	return m, nil
}

func (m InteractiveModel) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Don't intercept keys while filtering.
	if m.list.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "enter":
		if item, ok := m.list.SelectedItem().(WorktreeItem); ok {
			if item.wt.IsCurrent {
				m.message = "Already in this worktree"
				return m, nil
			}
			m.switchTo = item.wt.Path
			m.quitting = true
			return m, tea.Quit
		}
	case "a":
		m.mode = modeAdd
		m.addInput.Reset()
		m.addInput.Focus()
		m.message = ""
		m.err = nil
		return m, m.addInput.Cursor.BlinkCmd()
	case "d":
		if item, ok := m.list.SelectedItem().(WorktreeItem); ok {
			if item.wt.IsMain {
				m.message = "Cannot delete the main worktree"
				return m, nil
			}
			m.selected = &item.wt
			m.mode = modeConfirmDelete
			m.message = ""
			m.err = nil
		}
		return m, nil
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m InteractiveModel) handleAddKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		name := strings.TrimSpace(m.addInput.Value())
		if name == "" {
			m.mode = modeList
			return m, nil
		}
		m.pendingName = name
		return m.startLoading("Loading branches…", loadBranchesCmd(name))
	case "esc":
		m.mode = modeList
		return m, nil
	}

	var cmd tea.Cmd
	m.addInput, cmd = m.addInput.Update(msg)
	return m, cmd
}

func (m InteractiveModel) handleDeleteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if m.selected != nil {
			return m.startLoading(
				fmt.Sprintf("Removing '%s'…", m.selected.Name),
				removeWorktreeCmd(*m.selected))
		}
		m.mode = modeList
		return m, nil
	case "n", "N", "esc":
		m.mode = modeList
		m.selected = nil
		return m, nil
	}
	return m, nil
}

func (m InteractiveModel) handleBranchPickKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.branchList.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.branchList, cmd = m.branchList.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "enter":
		if item, ok := m.branchList.SelectedItem().(BranchItem); ok {
			m.pendingBranch = item.branch.Name
			m.mode = modeSyncPrompt
			m.message = ""
			return m, nil
		}
	case "f":
		return m.startLoading("Fetching…", fetchBranchesCmd(m.pendingName))
	case "esc":
		m.mode = modeAdd
		return m, m.addInput.Cursor.BlinkCmd()
	}

	var cmd tea.Cmd
	m.branchList, cmd = m.branchList.Update(msg)
	return m, cmd
}

// handleSyncPromptKey asks whether to copy gitignored files into the new
// worktree, then creates it either way.
func (m InteractiveModel) handleSyncPromptKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		return m.startLoading("Creating worktree…",
			createWorktreeCmd(m.pendingName, m.pendingBranch, true))
	case "n", "N", "enter":
		return m.startLoading("Creating worktree…",
			createWorktreeCmd(m.pendingName, m.pendingBranch, false))
	case "esc":
		m.mode = modeBranchPick
		return m, nil
	}
	return m, nil
}

func (m InteractiveModel) updateCurrentMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.mode {
	case modeList:
		m.list, cmd = m.list.Update(msg)
	case modeAdd:
		m.addInput, cmd = m.addInput.Update(msg)
	case modeBranchPick:
		m.branchList, cmd = m.branchList.Update(msg)
	}
	return m, cmd
}

// --- view -------------------------------------------------------------------

func (m InteractiveModel) View() string {
	if m.quitting || m.height == 0 {
		return ""
	}

	header := m.headerView()
	footer := m.footerView()
	bodyH := m.height - chromeHeight

	body := m.bodyView(bodyH)

	// Status line sits between body and footer, occupying the lower spacer row
	// so the overall height stays constant.
	status := ""
	switch {
	case m.err != nil:
		status = ErrorStyle.Render("  " + m.err.Error())
	case m.message != "":
		status = WarnStyle.Render("  " + m.message)
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, "", body, status, footer)
}

func (m InteractiveModel) headerView() string {
	title := "Git Worktrees"
	switch m.mode {
	case modeAdd:
		title = "New Worktree"
	case modeBranchPick:
		title = fmt.Sprintf("Base branch for '%s'", m.pendingName)
	case modeSyncPrompt:
		title = "New Worktree"
	case modeConfirmDelete:
		title = "Delete Worktree"
	}
	bar := title
	if m.mode == modeList && !m.loading {
		bar += "  " + CountStyle.Render(fmt.Sprintf("(%d)", len(m.list.Items())))
	}
	return HeaderBarStyle.Width(m.width).Render(bar)
}

func (m InteractiveModel) footerView() string {
	var hints []string
	switch {
	case m.loading:
		hints = []string{hint("ctrl+c", "quit")}
	case m.mode == modeList && m.list.FilterState() == list.Filtering:
		hints = []string{hint("enter", "apply"), hint("esc", "clear")}
	case m.mode == modeList:
		hints = []string{hint("enter", "switch"), hint("a", "add"), hint("d", "delete"), hint("/", "filter"), hint("q", "quit")}
	case m.mode == modeAdd:
		hints = []string{hint("enter", "next"), hint("esc", "cancel")}
	case m.mode == modeBranchPick:
		hints = []string{hint("enter", "select"), hint("f", "fetch"), hint("/", "filter"), hint("esc", "back")}
	case m.mode == modeSyncPrompt:
		hints = []string{hint("y", "copy"), hint("n", "skip"), hint("esc", "back")}
	case m.mode == modeConfirmDelete:
		hints = []string{hint("y", "delete"), hint("n", "cancel")}
	}
	return "  " + strings.Join(hints, HelpStyle.Render("  ·  "))
}

func (m InteractiveModel) bodyView(h int) string {
	if m.loading {
		return center(m.width, h, m.spinner.View()+" "+HelpStyle.Render(m.loadingMsg))
	}

	switch m.mode {
	case modeBranchPick:
		return place(m.width, h, m.branchList.View())
	case modeAdd:
		box := ModalStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
			ModalTitle.Render("Name the new worktree"),
			BranchStyle.Render("› ")+m.addInput.View(),
		))
		return center(m.width, h, box)
	case modeSyncPrompt:
		box := ModalStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
			ModalTitle.Render(fmt.Sprintf("Create '%s' from '%s'", m.pendingName, m.pendingBranch)),
			"Copy gitignored files (.env, config, …)",
			"from the main worktree?",
		))
		return center(m.width, h, box)
	case modeConfirmDelete:
		warn := fmt.Sprintf("Delete worktree '%s'?", m.selected.Name)
		lines := []string{ModalTitle.Render(warn)}
		if m.selected.IsDirty {
			lines = append(lines, WarnStyle.Render(fmt.Sprintf("✱ %d uncommitted change(s) will be lost.", m.selected.DirtyCount)))
		}
		box := ModalStyle.BorderForeground(Red).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
		return center(m.width, h, box)
	default:
		return place(m.width, h, m.list.View())
	}
}

// SwitchTo returns the path to switch to, or empty string.
func (m InteractiveModel) SwitchTo() string { return m.switchTo }

// RunInteractive launches the full interactive TUI.
func RunInteractive() (string, error) {
	alignColorToStderr()
	worktrees, err := git.ListWorktrees()
	if err != nil {
		return "", err
	}
	p := tea.NewProgram(NewInteractive(worktrees), tea.WithAltScreen(), tea.WithOutput(os.Stderr))
	result, err := p.Run()
	if err != nil {
		return "", err
	}
	return result.(InteractiveModel).SwitchTo(), nil
}

// --- small view helpers -----------------------------------------------------

func hint(k, desc string) string {
	return KeyStyle.Render(k) + " " + HelpStyle.Render(desc)
}

// place pads content to a fixed region, top-left aligned.
func place(w, h int, content string) string {
	return lipgloss.Place(w, h, lipgloss.Left, lipgloss.Top, content)
}

// center pads content to a fixed region, centered.
func center(w, h int, content string) string {
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, content)
}
