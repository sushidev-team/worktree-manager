package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

// BranchPickerModel is a Bubbletea model for picking a base branch.
type BranchPickerModel struct {
	list     list.Model
	selected string
	quitting bool
}

// NewBranchPicker creates a new branch picker TUI.
func NewBranchPicker(branches []git.Branch) BranchPickerModel {
	items := make([]list.Item, len(branches))
	for i, b := range branches {
		items[i] = BranchItem{branch: b}
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(Purple).
		BorderLeftForeground(Purple)
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.
		Foreground(White)

	l := list.New(items, delegate, 60, 15)
	l.Title = "Select base branch"
	l.Styles.Title = TitleStyle
	l.SetFilteringEnabled(true)
	l.SetShowStatusBar(true)
	l.SetShowHelp(true)

	return BranchPickerModel{list: l}
}

func (m BranchPickerModel) Init() tea.Cmd {
	return nil
}

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
		case "q", "esc":
			m.quitting = true
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 2)
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
	return "\n" + m.list.View()
}

func (m BranchPickerModel) Selected() string {
	return m.selected
}

// PickBranch runs an interactive branch picker and returns the selected branch.
func PickBranch() (string, error) {
	branches, err := git.ListBranches()
	if err != nil {
		return "", err
	}

	if len(branches) == 0 {
		return "", fmt.Errorf("no branches found")
	}

	model := NewBranchPicker(branches)
	p := tea.NewProgram(model, tea.WithAltScreen())
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

// PickBranchWithDefault returns default branch if it exists, otherwise prompts.
func PickBranchWithDefault() (string, error) {
	defaultBranch := git.DefaultBranch()

	fmt.Printf("  Base branch: %s (press Enter to confirm, or type to search)\n",
		lipgloss.NewStyle().Bold(true).Foreground(Cyan).Render(defaultBranch))
	fmt.Print("  > ")

	// Use the interactive picker
	return PickBranch()
}
