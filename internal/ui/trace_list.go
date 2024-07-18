package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/list"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/zopu/tracey/internal/xray"
)

type TraceList struct {
	Traces   []xray.TraceSummary
	Selected mo.Option[int]
	cursor   int
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

func (tl *TraceList) Select() xray.TraceID {
	tl.Selected = mo.Some(tl.cursor)
	return xray.TraceID(tl.Traces[tl.cursor].ID())
}

func (tl TraceList) View() string {
	if len(tl.Traces) == 0 {
		return "Looking for traces...\n\n"
	}
	enumeratorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("99")).MarginRight(1)

	tIDs := lo.Map(tl.Traces, func(t xray.TraceSummary, _ int) string {
		return t.Title()
	})

	start := max(0, tl.cursor-10)
	end := min(len(tIDs), start+20)
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
	return l.Enumerator(enumerator).String()
}
