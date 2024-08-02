package main

import (
	"context"
	"log"
	"regexp"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/samber/mo"
	"github.com/zopu/tracey/internal/aws"
	"github.com/zopu/tracey/internal/config"
	"github.com/zopu/tracey/internal/store"
	"github.com/zopu/tracey/internal/ui"
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
	config        config.App
	logGroups     []string
	store         *store.Store
	error         mo.Option[string]
	list          ui.TraceList
	detailsPane   ui.DetailsPane
	helpBar       ui.HelpBar
	selectedPane  int
	width, height int
}

func initialModel(config config.App, logGroups []string) model {
	st := store.New()
	m := model{
		config:    config,
		logGroups: logGroups,
		list:      ui.NewTraceList(),
		detailsPane: ui.DetailsPane{
			LogFields: config.Logs.ParsedFields,
		},
		helpBar:      ui.HelpBar{},
		selectedPane: PaneList,
		store:        &st,
	}
	m.list.SetFocus(true)
	return m
}

type TraceSummaryMsg struct {
	NextToken       mo.Option[string]
	traces          []aws.TraceSummary
	ShouldFetchMore bool
}

type TraceDetailsMsg struct {
	trace       *aws.TraceDetails
	logsQueryID *aws.LogQueryID
}

type TraceLogsMsg struct {
	logs *aws.LogData
}

type ErrorMsg struct {
	Msg string
}

func fetchTraceSummaries(store *store.Store, pathFilters []regexp.Regexp, nextToken mo.Option[string]) tea.Msg {
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
		traces:          filtered,
		NextToken:       result.NextToken,
		ShouldFetchMore: shouldFetchMore,
	}
}

func fetchTraceDetails(id aws.TraceID, logGroupNames []string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		details, err := aws.FetchTraceDetails(ctx, id)
		if err != nil {
			return ErrorMsg{Msg: err.Error()}
		}
		var logsQueryID *aws.LogQueryID
		if len(logGroupNames) > 0 {
			logsQueryID, err = aws.StartLogsQuery(ctx, logGroupNames, id)
		}
		if err != nil {
			return ErrorMsg{Msg: err.Error()}
		}
		return TraceDetailsMsg{trace: details, logsQueryID: logsQueryID}
	}
}

func fetchLogs(id aws.LogQueryID, delay time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(delay)
		logs, err := aws.FetchLogs(context.Background(), id)
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
		m.width = msg.Width
		m.height = msg.Height
		m.updatePaneDimensions()

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
		m.detailsPane.Logs = mo.None[aws.LogData]()
		if msg.logsQueryID != nil {
			return m, fetchLogs(*msg.logsQueryID, time.Second)
		}
		return m, nil

	case TraceLogsMsg:
		m.detailsPane.Logs = mo.Some(*msg.logs)

	case ui.ListSelectionMsg:
		m.detailsPane.Details = mo.None[aws.TraceDetails]()
		return m, fetchTraceDetails(msg.ID, m.logGroups)

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
			m.updatePaneDimensions()
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

func (m *model) updatePaneDimensions() {
	m.list.Width = m.width
	m.detailsPane.Width = m.width
	m.helpBar.Width = m.width
	if m.selectedPane == PaneList {
		m.detailsPane.Height = m.height - 12
	} else {
		m.detailsPane.Height = m.height - 3
	}
}

func (m model) View() string {
	if m.error.IsPresent() {
		return "Error: " + m.error.MustGet() + "\n\n"
	}

	s := m.list.View()
	s += "\n\n"
	s += m.detailsPane.View()
	s += m.helpBar.Render()
	return s
}

func main() {
	config, err := config.Parse()
	if err != nil {
		log.Fatalf("Error reading config: %s", err)
	}

	logGroups, err := aws.GetLogGroups(context.Background())
	if err != nil {
		log.Fatalf("Could not load log groups")
	}
	filteredLogGroups := make([]string, 0)
	for _, groupFilter := range config.Logs.Groups {
		re, reErr := regexp.Compile(groupFilter)
		if reErr != nil {
			log.Fatalf("Could not compile log group regexp: %s", reErr)
		}
		for _, lg := range logGroups {
			if re.MatchString(lg) {
				filteredLogGroups = append(filteredLogGroups, lg)
			}
		}
	}

	p := tea.NewProgram(initialModel(*config, filteredLogGroups), tea.WithAltScreen())
	if _, err = p.Run(); err != nil {
		log.Fatalf("Alas, there's been an error: %v", err)
	}
}
