package ui

import (
	"context"
	"regexp"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/list"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/zopu/tracey/internal/aws"
	"github.com/zopu/tracey/internal/store"
)

type TraceSummaryMsg struct {
	NextToken       mo.Option[string]
	Traces          []aws.TraceSummary
	ShouldFetchMore bool
}

func FetchTraceSummaries(store *store.Store, pathFilters []regexp.Regexp, nextToken mo.Option[string]) tea.Msg {
	result, err := aws.FetchTraceSummaries(context.Background(), nextToken)
	if err != nil {
		return ErrorMsg{Msg: err.Error()}
	}
	store.AddTraceSummaries(result.Summaries)
	summaries := store.GetTraceSummaries()

	// Filter out traces that match any exclude regex
	filtered := make([]aws.TraceSummary, 0)
outer:
	for _, trace := range summaries {
		for _, exclude := range pathFilters {
			if exclude.MatchString(trace.Path()) {
				continue outer
			}
		}
		filtered = append(filtered, trace)
	}

	shouldFetchMore := result.NextToken.IsPresent() && store.Size() < 20

	return TraceSummaryMsg{
		Traces:          filtered,
		NextToken:       result.NextToken,
		ShouldFetchMore: shouldFetchMore,
	}
}

type TraceList struct {
	Traces    []aws.TraceSummary
	NextToken mo.Option[string]
	Width     int
	selected  mo.Option[int]
	focused   bool
	cursor    int
}

func NewTraceList() TraceList {
	return TraceList{
		Traces: []aws.TraceSummary{},
	}
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

type ListSelectionMsg struct {
	ID aws.TraceID
}

type ListAtEndMsg struct{}

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
			tl.selected = mo.Some(tl.cursor)
			return func() tea.Msg {
				return ListSelectionMsg{ID: aws.TraceID(tl.Traces[tl.cursor].ID())}
			}

		case "tab":
			return func() tea.Msg {
				return SelectNextPaneMsg{}
			}
		}
	}

	if tl.cursor == len(tl.Traces)-1 {
		return func() tea.Msg {
			return ListAtEndMsg{}
		}
	}
	return nil
}

func (tl TraceList) View() string {
	if tl.focused {
		return tl.ViewFocused()
	}
	if len(tl.Traces) == 0 {
		return "Looking for traces...\n"
	}

	s := "No trace selected"
	if tl.selected.IsPresent() {
		s = listEnumeratorStyle().Render("  ")
		title := tl.Traces[tl.selected.MustGet()].Title()
		s += tl.StyleItem(tl.selected.MustGet()).Render(title)
	}

	style := lipgloss.NewStyle().
		Width(tl.Width - 2).
		Height(1).
		BorderStyle(lipgloss.NormalBorder())
	return style.Render(s)
}

func (tl TraceList) ViewFocused() string {
	if len(tl.Traces) == 0 {
		return "Looking for traces...\n\n"
	}

	tIDs := lo.Map(tl.Traces, func(summary aws.TraceSummary, _ int) string {
		// I'd expect lipgloss inline styling to truncate these to the width, but it doesn't,
		// so we have to do it here.
		t := summary.Title()
		maxLen := tl.Width - 6
		if len(t) > maxLen {
			return t[:maxLen-3] + "..."
		}
		return t
	})

	start := max(0, tl.cursor-5)
	end := min(len(tIDs), start+10)
	l := list.New(tIDs[start:end]).
		EnumeratorStyle(listEnumeratorStyle()).
		ItemStyleFunc(func(_ list.Items, i int) lipgloss.Style {
			return tl.StyleItem(i + start)
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
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("63"))
	return style.Render(s)
}

func listEnumeratorStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("99")).MarginRight(1)
}

func (tl TraceList) StyleItem(index int) lipgloss.Style {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("#c6d0f5")).MarginRight(1)
	if tl.cursor == index {
		style = style.Background(lipgloss.Color("#303446"))
	}
	if sel, ok := tl.selected.Get(); ok && sel == index {
		style = style.Background(lipgloss.Color("#414559"))
	}
	if tl.Traces[index].HasError() {
		style = style.Foreground(lipgloss.Color("#e78284"))
	}
	if tl.Traces[index].HasFault() {
		style = style.Foreground(lipgloss.Color("#e78284"))
	}
	return style
}
