package main

import (
	"context"
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/list"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/zopu/tracey/internal/xray"
)

type model struct {
	error         mo.Option[string]
	traces        []xray.TraceSummary
	cursor        int // which to-do list item our cursor is pointing at
	selected      mo.Option[int]
	traceSelected mo.Option[xray.TraceDetails]
}

func initialModel() model {
	return model{
		traces: []xray.TraceSummary{},
	}
}

type TraceSummaryMsg struct {
	traces []xray.TraceSummary
}

type TraceDetailsMsg struct {
	trace *xray.TraceDetails
}

type ErrorMsg struct {
	Msg string
}

func fetchTraceSummaries() tea.Msg {
	summary, err := xray.FetchTraceSummaries(context.Background())
	if err != nil {
		return ErrorMsg{Msg: err.Error()}
	}
	return TraceSummaryMsg{traces: summary}
}

func fetchTraceDetails(traceID string) tea.Cmd {
	return func() tea.Msg {
		details, err := xray.FetchTraceDetails(context.Background(), traceID)
		if err != nil {
			return ErrorMsg{Msg: err.Error()}
		}
		return TraceDetailsMsg{trace: details}
	}
}

func (m model) Init() tea.Cmd {
	return fetchTraceSummaries
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ErrorMsg:
		m.error = mo.Some(msg.Msg)

	case TraceSummaryMsg:
		m.traces = msg.traces

	case TraceDetailsMsg:
		m.traceSelected = mo.Some(*msg.trace)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "ctrl+u":
			m.cursor -= 10
			if m.cursor < 0 {
				m.cursor = 0
			}
		case "down", "j":
			if m.cursor < len(m.traces)-1 {
				m.cursor++
			}
		case "ctrl+d":
			m.cursor += 10
			if m.cursor > len(m.traces)-1 {
				m.cursor = len(m.traces) - 1
			}

		case "enter", " ":
			m.selected = mo.Some(m.cursor)
			return m, fetchTraceDetails(m.traces[m.cursor].ID())
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.error.IsPresent() {
		return "Error: " + m.error.MustGet() + "\n\n"
	}

	if len(m.traces) == 0 {
		return "Looking for traces...\n\n"
	}
	s := "Select a trace to view\n\n"
	if m.traceSelected.IsPresent() {
		s = "Selected trace: " + m.traceSelected.MustGet().String() + "\n\n"
	}

	enumeratorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("99")).MarginRight(1)

	tIDs := lo.Map(m.traces, func(t xray.TraceSummary, _ int) string {
		return t.Title()
	})

	start := max(0, m.cursor-10)
	end := min(len(tIDs), start+20)
	l := list.New(tIDs[start:end]).
		EnumeratorStyle(enumeratorStyle).
		ItemStyleFunc(func(_ list.Items, i int) lipgloss.Style {
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("#c6d0f5")).MarginRight(1)
			if m.cursor == i+start {
				style = style.Background(lipgloss.Color("#303446"))
			}
			if sel, ok := m.selected.Get(); ok && sel == i+start {
				style = style.Background(lipgloss.Color("#414559"))
			}
			if m.traces[i+start].HasError() {
				style = style.Foreground(lipgloss.Color("#e78284"))
			}
			if m.traces[i+start].HasFault() {
				style = style.Foreground(lipgloss.Color("#e78284"))
			}
			return style
		})

	enumerator := func(_ list.Items, i int) string {
		prefix := ""
		if m.cursor == i+start {
			prefix += "→"
		}
		return prefix + " "
	}
	s += l.Enumerator(enumerator).String()

	m.traceSelected.ForEach(func(td xray.TraceDetails) {
		s += fmt.Sprintf("\n\nSegments: %d\n", len(td.Segments()))
		for _, segment := range td.Segments() {
			s += fmt.Sprintf("Document: %s\n", *segment.Document)
		}
	})
	return s
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Alas, there's been an error: %v", err)
	}
}
