package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sushi/worktree-manager/internal/git"
)

// BranchItem implements list.Item for the branch picker.
type BranchItem struct {
	branch git.Branch
}

func (b BranchItem) Title() string {
	name := b.branch.Name
	if b.branch.IsDefault {
		name += " (default)"
	}
	if b.branch.IsCurrent {
		name += " *"
	}
	if b.branch.IsRemote {
		name += " (remote)"
	}
	return name
}

func (b BranchItem) Description() string { return "" }
func (b BranchItem) FilterValue() string { return b.branch.Name }

// newBranchList builds a styled single-line branch list, shared by the
// standalone picker and the inline add flow.
func newBranchList(branches []git.Branch, width, height int) list.Model {
	items := make([]list.Item, len(branches))
	for i, b := range branches {
		items[i] = BranchItem{branch: b}
	}

	if width <= 0 {
		width = 60
	}
	if height <= 0 {
		height = 15
	}
	l := list.New(items, branchDelegate{}, width, height)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.FilterInput.PromptStyle = BranchStyle
	l.FilterInput.Cursor.Style = BranchStyle
	return l
}

// branchFetchMsg carries the result of an in-picker `git fetch` + reload.
type branchFetchMsg struct {
	list []git.Branch
	err  error
}

func pickerFetchCmd() tea.Msg {
	if err := git.Fetch(); err != nil {
		return branchFetchMsg{err: err}
	}
	bs, err := git.ListBranches()
	return branchFetchMsg{list: bs, err: err}
}

// BranchPickerModel is a Bubbletea model for picking a base branch.
type BranchPickerModel struct {
	list     list.Model
	spinner  spinner.Model
	selected string
	quitting bool
	fetching bool
	err      error
}

// NewBranchPicker creates a new branch picker TUI.
func NewBranchPicker(branches []git.Branch) BranchPickerModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = SpinnerStyle
	return BranchPickerModel{list: newBranchList(branches, 60, 15), spinner: sp}
}

func (m BranchPickerModel) Init() tea.Cmd { return nil }

func (m BranchPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// While fetching, only allow aborting.
		if m.fetching {
			if msg.String() == "ctrl+c" {
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}
		// Don't intercept keys when filtering
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch msg.String() {
		case "enter":
			if item, ok := m.list.SelectedItem().(BranchItem); ok {
				m.selected = item.branch.Name
			}
			m.quitting = true
			return m, tea.Quit
		case "f":
			m.fetching = true
			m.err = nil
			return m, tea.Batch(m.spinner.Tick, pickerFetchCmd)
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	case branchFetchMsg:
		m.fetching = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		items := make([]list.Item, len(msg.list))
		for i, b := range msg.list {
			items[i] = BranchItem{branch: b}
		}
		m.list.SetItems(items)
		return m, nil
	case spinner.TickMsg:
		if !m.fetching {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-2)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m BranchPickerModel) View() string {
	if m.quitting && m.selected != "" {
		return fmt.Sprintf("\n  %s %s\n",
			SuccessStyle.Render("Base branch:"),
			BranchStyle.Render(m.selected))
	}
	header := HeaderBarStyle.Render(" Select base branch ")

	var footer string
	if m.fetching {
		footer = "  " + m.spinner.View() + " " + HelpStyle.Render("Fetching…")
	} else {
		footer = "  " + strings.Join([]string{
			hint("enter", "select"), hint("f", "fetch"), hint("/", "filter"), hint("esc", "cancel"),
		}, HelpStyle.Render("  ·  "))
	}
	if m.err != nil {
		footer = ErrorStyle.Render("  "+m.err.Error()) + "\n" + footer
	}

	return "\n" + header + "\n\n" + m.list.View() + "\n" + footer
}

func (m BranchPickerModel) Selected() string { return m.selected }

// PickBranch runs an interactive branch picker and returns the selected branch.
func PickBranch() (string, error) {
	alignColorToStderr()
	branches, err := git.ListBranches()
	if err != nil {
		return "", err
	}
	if len(branches) == 0 {
		return "", fmt.Errorf("no branches found")
	}

	model := NewBranchPicker(branches)
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithOutput(os.Stderr))
	result, err := p.Run()
	if err != nil {
		return "", err
	}

	selected := result.(BranchPickerModel).Selected()
	if selected == "" {
		return "", fmt.Errorf("no branch selected")
	}
	return selected, nil
}
