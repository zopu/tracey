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
	error    mo.Option[string]
	traces   []xray.TraceSummary
	cursor   int // which to-do list item our cursor is pointing at
	selected mo.Option[int]
}

func initialModel() model {
	return model{
		traces: []xray.TraceSummary{},
	}
}

type TraceSummaryMsg struct {
	traces []xray.TraceSummary
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

func (m model) Init() tea.Cmd {
	return fetchTraceSummaries
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ErrorMsg:
		m.error = mo.Some(msg.Msg)

	case TraceSummaryMsg:
		m.traces = msg.traces

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.traces)-1 {
				m.cursor++
			}

		case "enter", " ":
			m.selected = mo.Some(m.cursor)
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

	enumeratorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("99")).MarginRight(1)
	itemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212")).MarginRight(1)

	tIDs := lo.Map(m.traces, func(t xray.TraceSummary, _ int) string {
		return t.ID
	})
	s += fmt.Sprintf("Num tIDs: %v\n", len(tIDs))

	start := max(0, m.cursor-10)
	end := min(len(tIDs), start+20)
	l := list.New(tIDs[start:end]).
		EnumeratorStyle(enumeratorStyle).
		ItemStyle(itemStyle)

	enumerator := func(_ list.Items, i int) string {
		prefix := ""
		if m.cursor == i+start {
			prefix += "→"
		}
		if sel, ok := m.selected.Get(); ok && sel == i+start {
			return prefix + "x"
		}
		return prefix + " "
	}

	return s + l.Enumerator(enumerator).String()
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Alas, there's been an error: %v", err)
	}
}
