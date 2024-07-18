package main

import (
	"context"
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/samber/mo"
	"github.com/zopu/tracey/internal/ui"
	"github.com/zopu/tracey/internal/xray"
)

type model struct {
	error         mo.Option[string]
	list          ui.TraceList
	traceSelected mo.Option[xray.TraceDetails]
}

func initialModel() model {
	return model{
		list: ui.TraceList{
			Traces: []xray.TraceSummary{},
		},
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

func fetchTraceDetails(id xray.TraceID) tea.Cmd {
	return func() tea.Msg {
		details, err := xray.FetchTraceDetails(context.Background(), id)
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
		m.list.Traces = msg.traces

	case TraceDetailsMsg:
		m.traceSelected = mo.Some(*msg.trace)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			m.list.MoveCursor(-1)
		case "ctrl+u":
			m.list.MoveCursor(-10)
		case "down", "j":
			m.list.MoveCursor(1)
		case "ctrl+d":
			m.list.MoveCursor(10)

		case "enter", " ":
			id := m.list.Select()
			m.traceSelected = mo.None[xray.TraceDetails]()
			return m, fetchTraceDetails(id)
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.error.IsPresent() {
		return "Error: " + m.error.MustGet() + "\n\n"
	}

	s := "Select a trace to view\n\n"
	if m.traceSelected.IsPresent() {
		s = "Selected trace: " + m.traceSelected.MustGet().String() + "\n\n"
	}

	s += m.list.View()

	m.traceSelected.ForEach(func(td xray.TraceDetails) {
		s += fmt.Sprintf("\n\nSegments: %d\n", len(td.Segments))
		for _, segment := range td.Segments {
			s += fmt.Sprintf("Name: %s, Subsegments: %d\nValue: %.40v\n\n", segment.Name, len(segment.SubSegments), segment)
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
