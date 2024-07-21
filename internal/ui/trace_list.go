package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/list"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/zopu/tracey/internal/xray"
)

type TraceList struct {
	Traces   []xray.TraceSummary
	Selected mo.Option[int]
	OnSelect func(xray.TraceID) tea.Cmd
	focused  bool
	cursor   int
	Width    int
}

func (tl *TraceList) MoveCursor(amount int) {
	tl.cursor += amount
	if tl.cursor < 0 {
		tl.cursor = 0
	}
	if tl.cursor >= len(tl.Traces) {
		tl.cursor = len(tl.Traces) - 1
	}
}

func (tl *TraceList) SetFocus(focus bool) {
	tl.focused = focus
}

func (tl *TraceList) Update(msg tea.Msg) tea.Cmd {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "up", "k":
			tl.MoveCursor(-1)
		case "ctrl+u":
			tl.MoveCursor(-10)
		case "down", "j":
			tl.MoveCursor(1)
		case "ctrl+d":
			tl.MoveCursor(10)

		case "enter", " ":
			tl.Selected = mo.Some(tl.cursor)
			id := xray.TraceID(tl.Traces[tl.cursor].ID())
			return tl.OnSelect(id)
		}
	}
	return nil
}

func (tl TraceList) View() string {
	if len(tl.Traces) == 0 {
		return "Looking for traces...\n\n"
	}
	enumeratorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("99")).MarginRight(1)

	tIDs := lo.Map(tl.Traces, func(t xray.TraceSummary, _ int) string {
		return t.Title()
	})

	start := max(0, tl.cursor-5)
	end := min(len(tIDs), start+10)
	l := list.New(tIDs[start:end]).
		EnumeratorStyle(enumeratorStyle).
		ItemStyleFunc(func(_ list.Items, i int) lipgloss.Style {
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("#c6d0f5")).MarginRight(1)
			if tl.cursor == i+start {
				style = style.Background(lipgloss.Color("#303446"))
			}
			if sel, ok := tl.Selected.Get(); ok && sel == i+start {
				style = style.Background(lipgloss.Color("#414559"))
			}
			if tl.Traces[i+start].HasError() {
				style = style.Foreground(lipgloss.Color("#e78284"))
			}
			if tl.Traces[i+start].HasFault() {
				style = style.Foreground(lipgloss.Color("#e78284"))
			}
			return style
		})

	enumerator := func(_ list.Items, i int) string {
		prefix := ""
		if tl.cursor == i+start {
			prefix += "→"
		}
		return prefix + " "
	}
	s := l.Enumerator(enumerator).String()

	style := lipgloss.NewStyle().
		Width(tl.Width - 2).
		Height(10).
		BorderStyle(lipgloss.NormalBorder())
	if tl.focused {
		style = style.BorderForeground(lipgloss.Color("63"))
	}
	return style.Render(s)
}
