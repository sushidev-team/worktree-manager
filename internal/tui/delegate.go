package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// worktreeDelegate renders each worktree as two lines: a name line with a
// selection bar and a colored current-marker, and a meta line packing branch,
// hash, ahead/behind, dirty count and last-commit age. The selected row also
// shows the last commit subject.
type worktreeDelegate struct{}

func (d worktreeDelegate) Height() int  { return 2 }
func (d worktreeDelegate) Spacing() int { return 1 }

func (d worktreeDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d worktreeDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	it, ok := item.(WorktreeItem)
	if !ok {
		return
	}
	wt := it.wt
	selected := index == m.Index()
	width := m.Width()
	if width <= 0 {
		width = 80
	}

	// Line 1: selection bar + current dot + name.
	bar := "  "
	if selected {
		bar = BarStyle.Render("▎") + " "
	}
	dot := "  "
	if wt.IsCurrent {
		dot = CurrentDotStyle.Render("●") + " "
	}
	// deriveName already suffixes the main worktree with " (main)".
	name := NormalName.Render(wt.Name)
	if selected {
		name = SelectedName.Render(wt.Name)
	}
	line1 := bar + dot + name

	// Line 2: meta segments.
	var segs []string
	if wt.Branch != "" {
		segs = append(segs, BranchStyle.Render(wt.Branch))
	}
	if wt.Head != "" {
		segs = append(segs, HashStyle.Render(wt.Head))
	}
	if wt.HasUpstream && (wt.Ahead > 0 || wt.Behind > 0) {
		ab := ""
		if wt.Ahead > 0 {
			ab += AheadStyle.Render(fmt.Sprintf("↑%d", wt.Ahead))
		}
		if wt.Behind > 0 {
			if ab != "" {
				ab += " "
			}
			ab += BehindStyle.Render(fmt.Sprintf("↓%d", wt.Behind))
		}
		segs = append(segs, ab)
	}
	if wt.DirtyCount > 0 {
		segs = append(segs, DirtyStyle.Render(fmt.Sprintf("✱%d", wt.DirtyCount)))
	}
	if wt.LastCommit != "" {
		segs = append(segs, MetaStyle.Render(wt.LastCommit))
	}
	meta := strings.Join(segs, MetaStyle.Render(" · "))
	if selected && wt.Subject != "" {
		meta += "\n    " + SubjectStyle.Render(truncate(wt.Subject, width-4))
	}
	line2 := "    " + meta

	out := lipgloss.NewStyle().MaxWidth(width).Render(line1 + "\n" + line2)
	fmt.Fprint(w, out)
}

// branchDelegate renders a branch as a single line with the same selection
// language as the worktree list: a bright pointer + bold name when selected,
// muted otherwise, plus default/current tags.
type branchDelegate struct{}

func (d branchDelegate) Height() int                             { return 1 }
func (d branchDelegate) Spacing() int                            { return 0 }
func (d branchDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d branchDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	it, ok := item.(BranchItem)
	if !ok {
		return
	}
	width := m.Width()
	if width <= 0 {
		width = 80
	}

	selected := index == m.Index()

	pointer := "  "
	if selected {
		pointer = PointerStyle.Render("❯ ")
	}

	// The synthetic "create new branch" entry: a highlighted call-to-action
	// rather than a real branch, so it renders without branch tags.
	if it.create {
		label := DefaultTag.Render(createBranchLabel)
		if selected {
			label = SelectedName.Render(createBranchLabel)
		}
		fmt.Fprint(w, lipgloss.NewStyle().MaxWidth(width).Render("  "+pointer+label))
		return
	}

	b := it.branch
	name := NormalName.Render(b.Name)
	if selected {
		name = SelectedName.Render(b.Name)
	}

	var tags string
	if b.IsDefault {
		tags += " " + DefaultTag.Render("default")
	}
	if b.IsCurrent {
		tags += " " + CurrentTag.Render("current")
	}
	if b.IsRemote {
		tags += " " + RemoteTag.Render("remote")
	}

	line := "  " + pointer + name + tags
	fmt.Fprint(w, lipgloss.NewStyle().MaxWidth(width).Render(line))
}

func truncate(s string, max int) string {
	if max < 1 {
		return ""
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	if max == 1 {
		return "…"
	}
	// Rune-safe trim, then ellipsis.
	r := []rune(s)
	for len(r) > 0 && lipgloss.Width(string(r))+1 > max {
		r = r[:len(r)-1]
	}
	return string(r) + "…"
}
