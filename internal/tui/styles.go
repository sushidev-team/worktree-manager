package tui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// alignColorToStderr makes lipgloss detect color support from stderr, where the
// TUI is rendered, instead of the default stdout. The shell wrapper captures
// stdout via $(...), so stdout is a pipe and stdout-based detection would
// disable color entirely — rendering the whole UI plain white. Must be called
// before any styled string is rendered.
func alignColorToStderr() {
	lipgloss.SetColorProfile(termenv.NewOutput(os.Stderr).EnvColorProfile())
}

var (
	// Palette
	Purple  = lipgloss.Color("#7C3AED")
	Green   = lipgloss.Color("#10B981")
	Red     = lipgloss.Color("#EF4444")
	Yellow  = lipgloss.Color("#F59E0B")
	Gray    = lipgloss.Color("#6B7280")
	DimGray = lipgloss.Color("#374151")
	White   = lipgloss.Color("#F9FAFB")
	Cyan    = lipgloss.Color("#06B6D4")

	// HeaderBarStyle is the full-width title bar at the top of the screen.
	HeaderBarStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(White).
			Background(Purple).
			Padding(0, 1)

	// CountStyle dims the "(N)" item counter in the header.
	CountStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#C4B5FD"))

	// FooterStyle / HelpStyle render the bottom key hints.
	HelpStyle = lipgloss.NewStyle().Foreground(Gray)
	KeyStyle  = lipgloss.NewStyle().Foreground(Cyan)

	// Row selection: normal rows are muted, the selected row is bold + bright
	// with a colored marker, so the cursor is unmistakable.
	SoftGray        = lipgloss.Color("#9CA3AF")
	CurrentDotStyle = lipgloss.NewStyle().Foreground(Green)
	SelectedName    = lipgloss.NewStyle().Bold(true).Foreground(White)
	NormalName      = lipgloss.NewStyle().Foreground(SoftGray)
	BarStyle        = lipgloss.NewStyle().Foreground(Purple).Bold(true)
	PointerStyle    = lipgloss.NewStyle().Foreground(Cyan).Bold(true)
	DefaultTag      = lipgloss.NewStyle().Foreground(Cyan)
	CurrentTag      = lipgloss.NewStyle().Foreground(Green)
	RemoteTag       = lipgloss.NewStyle().Foreground(Yellow)
	BranchStyle     = lipgloss.NewStyle().Foreground(Cyan)
	HashStyle       = lipgloss.NewStyle().Foreground(Gray)
	MetaStyle       = lipgloss.NewStyle().Foreground(Gray)
	AheadStyle      = lipgloss.NewStyle().Foreground(Green)
	BehindStyle     = lipgloss.NewStyle().Foreground(Yellow)
	DirtyStyle      = lipgloss.NewStyle().Foreground(Yellow)
	SubjectStyle    = lipgloss.NewStyle().Foreground(SoftGray).Italic(true)

	// ModalStyle boxes the add / confirm / sync prompts.
	ModalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Purple).
			Padding(1, 3)

	ModalTitle = lipgloss.NewStyle().Bold(true).Foreground(Purple).MarginBottom(1)

	SpinnerStyle = lipgloss.NewStyle().Foreground(Purple)

	ErrorStyle   = lipgloss.NewStyle().Foreground(Red).Bold(true)
	SuccessStyle = lipgloss.NewStyle().Foreground(Green).Bold(true)
	WarnStyle    = lipgloss.NewStyle().Foreground(Yellow)
)
