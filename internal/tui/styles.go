package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	Purple    = lipgloss.Color("#7C3AED")
	Green     = lipgloss.Color("#10B981")
	Red       = lipgloss.Color("#EF4444")
	Yellow    = lipgloss.Color("#F59E0B")
	Gray      = lipgloss.Color("#6B7280")
	DimGray   = lipgloss.Color("#374151")
	White     = lipgloss.Color("#F9FAFB")
	Cyan      = lipgloss.Color("#06B6D4")

	// Styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Purple).
			MarginBottom(1)

	SelectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(White).
			Background(Purple).
			Padding(0, 1)

	NormalStyle = lipgloss.NewStyle().
			Foreground(White).
			Padding(0, 1)

	CurrentStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Green)

	DirtyStyle = lipgloss.NewStyle().
			Foreground(Yellow)

	BranchStyle = lipgloss.NewStyle().
			Foreground(Cyan)

	HashStyle = lipgloss.NewStyle().
			Foreground(Gray)

	HelpStyle = lipgloss.NewStyle().
			Foreground(Gray).
			MarginTop(1)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(Red).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(Green).
			Bold(true)

	DimStyle = lipgloss.NewStyle().
			Foreground(DimGray)
)
