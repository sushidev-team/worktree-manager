package tui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/list"
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

// BranchPickerModel is a Bubbletea model for picking a base branch.
type BranchPickerModel struct {
	list     list.Model
	selected string
	quitting bool
}

// NewBranchPicker creates a new branch picker TUI.
func NewBranchPicker(branches []git.Branch) BranchPickerModel {
	return BranchPickerModel{list: newBranchList(branches, 60, 15)}
}

func (m BranchPickerModel) Init() tea.Cmd { return nil }

func (m BranchPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
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
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
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
	return "\n" + header + "\n\n" + m.list.View()
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
