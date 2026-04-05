package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
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
)

// WorktreeItem implements list.Item for the worktree list.
type WorktreeItem struct {
	wt git.Worktree
}

func (w WorktreeItem) Title() string {
	name := w.wt.Name
	if w.wt.IsCurrent {
		name = "● " + name
	}
	return name
}

func (w WorktreeItem) Description() string {
	parts := []string{}
	if w.wt.Branch != "" {
		parts = append(parts, w.wt.Branch)
	}
	if w.wt.Head != "" {
		parts = append(parts, w.wt.Head)
	}
	if w.wt.IsDirty {
		parts = append(parts, "✱ dirty")
	}
	return strings.Join(parts, " · ")
}

func (w WorktreeItem) FilterValue() string {
	return w.wt.Name + " " + w.wt.Branch
}

// InteractiveModel is the main TUI model for the interactive worktree manager.
type InteractiveModel struct {
	list       list.Model
	mode       mode
	addInput   textinput.Model
	branchList list.Model
	selected   *git.Worktree
	message    string
	err        error
	width      int
	height     int
	quitting   bool
	switchTo   string // path to switch to after quitting
}

// NewInteractive creates the main interactive TUI model.
func NewInteractive(worktrees []git.Worktree) InteractiveModel {
	items := make([]list.Item, len(worktrees))
	for i, wt := range worktrees {
		items[i] = WorktreeItem{wt: wt}
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(Purple).
		BorderLeftForeground(Purple)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(Cyan).
		BorderLeftForeground(Purple)
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.
		Foreground(White)
	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.
		Foreground(Gray)

	l := list.New(items, delegate, 70, 20)
	l.Title = "Git Worktrees"
	l.Styles.Title = TitleStyle
	l.SetFilteringEnabled(true)
	l.SetShowStatusBar(true)
	l.SetShowHelp(false) // We'll show our own help
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "switch")),
			key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
			key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		}
	}

	ti := textinput.New()
	ti.Placeholder = "worktree-name"
	ti.CharLimit = 100
	ti.Width = 40

	return InteractiveModel{
		list:     l,
		addInput: ti,
	}
}

func (m InteractiveModel) Init() tea.Cmd {
	return nil
}

func (m InteractiveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 4)
		if m.mode == modeBranchPick {
			m.branchList.SetWidth(msg.Width)
			m.branchList.SetHeight(msg.Height - 6)
		}
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m.updateCurrentMode(msg)
}

func (m InteractiveModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeList:
		return m.handleListKey(msg)
	case modeAdd:
		return m.handleAddKey(msg)
	case modeConfirmDelete:
		return m.handleDeleteKey(msg)
	case modeBranchPick:
		return m.handleBranchPickKey(msg)
	}
	return m, nil
}

func (m InteractiveModel) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Don't intercept keys when filtering
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
		}
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
		// Move to branch picking
		branches, err := git.ListBranches()
		if err != nil {
			m.err = err
			m.mode = modeList
			return m, nil
		}
		items := make([]list.Item, len(branches))
		for i, b := range branches {
			items[i] = BranchItem{branch: b}
		}
		delegate := list.NewDefaultDelegate()
		delegate.ShowDescription = false
		delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
			Foreground(Purple).
			BorderLeftForeground(Purple)
		bl := list.New(items, delegate, m.width, m.height-6)
		bl.Title = fmt.Sprintf("Base branch for '%s'", name)
		bl.Styles.Title = TitleStyle
		bl.SetFilteringEnabled(true)
		m.branchList = bl
		m.mode = modeBranchPick
		return m, nil
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
			var err error
			if m.selected.IsDirty {
				err = git.ForceRemoveWorktree(m.selected.Path)
			} else {
				err = git.RemoveWorktree(m.selected.Path)
			}
			if err != nil {
				m.err = err
			} else {
				m.message = fmt.Sprintf("Removed worktree '%s'", m.selected.Name)
				// Refresh the list
				return m.refreshList()
			}
		}
		m.mode = modeList
		m.selected = nil
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
			name := strings.TrimSpace(m.addInput.Value())
			path, err := git.AddWorktree(name, item.branch.Name)
			if err != nil {
				m.err = err
				m.mode = modeList
				return m, nil
			}
			m.switchTo = path
			m.quitting = true
			return m, tea.Quit
		}
	case "esc":
		m.mode = modeAdd
		return m, m.addInput.Cursor.BlinkCmd()
	}

	var cmd tea.Cmd
	m.branchList, cmd = m.branchList.Update(msg)
	return m, cmd
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

func (m InteractiveModel) refreshList() (tea.Model, tea.Cmd) {
	worktrees, err := git.ListWorktrees()
	if err != nil {
		m.err = err
		m.mode = modeList
		return m, nil
	}
	items := make([]list.Item, len(worktrees))
	for i, wt := range worktrees {
		items[i] = WorktreeItem{wt: wt}
	}
	m.list.SetItems(items)
	m.mode = modeList
	m.selected = nil
	return m, nil
}

func (m InteractiveModel) View() string {
	if m.quitting {
		if m.switchTo != "" {
			// Print only the path for shell function to cd
			return m.switchTo
		}
		return ""
	}

	var b strings.Builder

	switch m.mode {
	case modeList:
		b.WriteString(m.list.View())
	case modeAdd:
		b.WriteString("\n")
		b.WriteString(TitleStyle.Render("New Worktree"))
		b.WriteString("\n\n")
		b.WriteString("  Name: ")
		b.WriteString(m.addInput.View())
		b.WriteString("\n\n")
		b.WriteString(HelpStyle.Render("  enter: confirm · esc: cancel"))
	case modeConfirmDelete:
		b.WriteString("\n")
		b.WriteString(m.list.View())
		b.WriteString("\n")
		warn := fmt.Sprintf("  Delete worktree '%s'?", m.selected.Name)
		if m.selected.IsDirty {
			warn += " (has uncommitted changes!)"
		}
		b.WriteString(ErrorStyle.Render(warn))
		b.WriteString("\n")
		b.WriteString(HelpStyle.Render("  y: yes · n: no"))
	case modeBranchPick:
		b.WriteString(m.branchList.View())
	}

	if m.message != "" {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(Yellow).Padding(0, 2).Render(m.message))
	}
	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(ErrorStyle.Padding(0, 2).Render(m.err.Error()))
	}

	// Help bar for list mode
	if m.mode == modeList {
		b.WriteString("\n")
		b.WriteString(HelpStyle.Padding(0, 2).Render(
			"enter: switch · a: add · d: delete · /: filter · q: quit"))
	}

	return b.String()
}

// SwitchTo returns the path to switch to, or empty string.
func (m InteractiveModel) SwitchTo() string {
	return m.switchTo
}

// RunInteractive launches the full interactive TUI.
func RunInteractive() (string, error) {
	worktrees, err := git.ListWorktrees()
	if err != nil {
		return "", err
	}

	model := NewInteractive(worktrees)
	p := tea.NewProgram(model, tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		return "", err
	}

	return result.(InteractiveModel).SwitchTo(), nil
}
