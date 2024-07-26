package main

import (
	"context"
	"log"
	"regexp"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/samber/mo"
	"github.com/zopu/tracey/internal/config"
	"github.com/zopu/tracey/internal/store"
	"github.com/zopu/tracey/internal/ui"
	"github.com/zopu/tracey/internal/xray"
)

const (
	PaneList = iota
	PaneDetails
)

type Pane interface {
	SetFocus(bool)
	Update(tea.Msg) tea.Cmd
}

type model struct {
	config       config.App
	store        *store.Store
	error        mo.Option[string]
	list         ui.TraceList
	detailsPane  ui.DetailsPane
	selectedPane int
}

func initialModel(config config.App) model {
	st := store.New()
	m := model{
		config: config,
		list: ui.TraceList{
			Traces: []xray.TraceSummary{},
		},
		detailsPane: ui.DetailsPane{
			LogFields: config.ParsedLogFields,
		},
		selectedPane: PaneList,
		store:        &st,
	}
	m.list.SetFocus(true)
	return m
}

type TraceSummaryMsg struct {
	NextToken       mo.Option[string]
	traces          []xray.TraceSummary
	ShouldFetchMore bool
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

func fetchTraceSummaries(store *store.Store, pathFilters []regexp.Regexp, nextToken mo.Option[string]) tea.Msg {
	result, err := xray.FetchTraceSummaries(context.Background(), nextToken)
	if err != nil {
		return ErrorMsg{Msg: err.Error()}
	}
	store.AddTraceSummaries(result.Summaries)
	summaries := store.GetTraceSummaries()

	filtered := make([]xray.TraceSummary, 0)
	// Filter out traces that match any exclude regex
	for _, exclude := range pathFilters {
		for _, trace := range summaries {
			if !exclude.MatchString(trace.Path()) {
				filtered = append(filtered, trace)
			}
		}
	}
	shouldFetchMore := result.NextToken.IsPresent() && store.Size() < 20

	return TraceSummaryMsg{
		traces:          filtered,
		NextToken:       result.NextToken,
		ShouldFetchMore: shouldFetchMore,
	}
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
	return func() tea.Msg {
		return fetchTraceSummaries(m.store, m.config.ParsedExcludePaths, mo.None[string]())
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var pane Pane
	switch m.selectedPane {
	case PaneDetails:
		pane = &m.detailsPane
	default:
		pane = &m.list
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.Width = msg.Width
		m.detailsPane.Width = msg.Width
		m.detailsPane.Height = msg.Height - 14

	case ErrorMsg:
		m.error = mo.Some(msg.Msg)

	case TraceSummaryMsg:
		m.list.Traces = msg.traces
		m.list.NextToken = msg.NextToken
		if msg.ShouldFetchMore {
			return m, func() tea.Msg {
				return fetchTraceSummaries(m.store, m.config.ParsedExcludePaths, msg.NextToken)
			}
		}

	case TraceDetailsMsg:
		m.detailsPane.Details = mo.Some(*msg.trace)
		m.detailsPane.Logs = mo.None[xray.LogData]()
		return m, fetchLogs(*msg.logsQueryID, time.Second)

	case TraceLogsMsg:
		m.detailsPane.Logs = mo.Some(*msg.logs)

	case ui.ListSelectionMsg:
		m.detailsPane.Details = mo.None[xray.TraceDetails]()
		return m, fetchTraceDetails(msg.ID, m.config.LogGroupName)

	case ui.ListAtEndMsg:
		return m, func() tea.Msg {
			return fetchTraceSummaries(m.store, m.config.ParsedExcludePaths, m.list.NextToken)
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			m.selectNextPane()
			return m, nil

		default:
			cmd := pane.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m *model) selectNextPane() {
	switch m.selectedPane {
	case PaneList:
		m.selectedPane = PaneDetails
		m.list.SetFocus(false)
		m.detailsPane.SetFocus(true)
	case PaneDetails:
		m.selectedPane = PaneList
		m.detailsPane.SetFocus(false)
		m.list.SetFocus(true)
	}
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
