package main

import (
	"context"
	"log"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/samber/mo"
	"github.com/zopu/tracey/internal/config"
	"github.com/zopu/tracey/internal/ui"
	"github.com/zopu/tracey/internal/xray"
)

type model struct {
	config      config.App
	error       mo.Option[string]
	list        ui.TraceList
	detailsPane ui.DetailsPane
}

func initialModel(config config.App) model {
	return model{
		config: config,
		list: ui.TraceList{
			Traces: []xray.TraceSummary{},
		},
		detailsPane: ui.DetailsPane{
			LogFields: config.ParsedLogFields,
		},
	}
}

type TraceSummaryMsg struct {
	traces []xray.TraceSummary
}

type TraceDetailsMsg struct {
	trace       *xray.TraceDetails
	logsQueryID *xray.LogQueryID
}

type TraceLogsMsg struct {
	logs *xray.LogData
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

func fetchTraceDetails(id xray.TraceID, logGroupName string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		details, err := xray.FetchTraceDetails(ctx, id)
		if err != nil {
			return ErrorMsg{Msg: err.Error()}
		}
		logsQueryID, err := xray.StartLogsQuery(ctx, logGroupName, id)
		if err != nil {
			return ErrorMsg{Msg: err.Error()}
		}
		return TraceDetailsMsg{trace: details, logsQueryID: logsQueryID}
	}
}

func fetchLogs(id xray.LogQueryID, delay time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(delay)
		logs, err := xray.FetchLogs(context.Background(), id)
		if err != nil {
			return ErrorMsg{Msg: err.Error()}
		}
		return TraceLogsMsg{logs: logs}
	}
}

func (m model) Init() tea.Cmd {
	return fetchTraceSummaries
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.Width = msg.Width
		m.detailsPane.Width = msg.Width
		m.detailsPane.Height = msg.Height - 14

	case ErrorMsg:
		m.error = mo.Some(msg.Msg)

	case TraceSummaryMsg:
		m.list.Traces = msg.traces

	case TraceDetailsMsg:
		m.detailsPane.Details = mo.Some(*msg.trace)
		m.detailsPane.Logs = mo.None[xray.LogData]()
		return m, fetchLogs(*msg.logsQueryID, time.Second)

	case TraceLogsMsg:
		m.detailsPane.Logs = mo.Some(*msg.logs)

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
			m.detailsPane.Details = mo.None[xray.TraceDetails]()
			return m, fetchTraceDetails(id, m.config.LogGroupName)
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.error.IsPresent() {
		return "Error: " + m.error.MustGet() + "\n\n"
	}

	s := m.list.View()
	s += "\n\n"
	s += m.detailsPane.View()
	return s
}

func main() {
	config, err := config.Parse()
	if err != nil {
		log.Fatalf("Error reading config: %s", err)
	}
	p := tea.NewProgram(initialModel(*config), tea.WithAltScreen())
	if _, err = p.Run(); err != nil {
		log.Fatalf("Alas, there's been an error: %v", err)
	}
}
