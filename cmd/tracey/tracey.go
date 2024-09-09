package main

import (
	"context"
	"log"
	"regexp"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

func (m model) Init() tea.Cmd {
	return func() tea.Msg {
		return ui.FetchTraceSummaries(m.store, m.config.ParsedExcludePaths, mo.None[string]())
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

	case ui.ErrorMsg:
		m.error = mo.Some(msg.Msg)

	case ui.TraceSummaryMsg:
		m.list.Traces = msg.Traces
		m.list.NextToken = msg.NextToken
		if msg.ShouldFetchMore {
			return m, func() tea.Msg {
				return ui.FetchTraceSummaries(m.store, m.config.ParsedExcludePaths, msg.NextToken)
			}
		}

	case ui.TraceDetailsMsg:
		return m, m.detailsPane.Update(msg)

	case ui.ClearTraceDetailsMsg:
		return m, m.detailsPane.Update(msg)

	case ui.TraceLogsMsg:
		m.detailsPane.Logs = mo.Some(*msg.Logs)

	case ui.ListSelectionMsg:
		clearCmd := func() tea.Msg {
			return ui.ClearTraceDetailsMsg{}
		}
		fetchCmd := ui.FetchTraceDetails(msg.ID, m.logGroups)
		return m, tea.Sequence(clearCmd, fetchCmd)

	case ui.ListAtEndMsg:
		return m, func() tea.Msg {
			return ui.FetchTraceSummaries(m.store, m.config.ParsedExcludePaths, m.list.NextToken)
		}
	case ui.SelectNextPaneMsg:
		m.selectNextPane()
		m.updatePaneDimensions()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

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
}

func (m model) View() string {
	if m.error.IsPresent() {
		return "Error: " + m.error.MustGet() + "\n\n"
	}

	list := m.list.View()
	helpBar := m.helpBar.Render()
	main := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height - lipgloss.Height(list) - lipgloss.Height(helpBar)).
		Render(m.detailsPane.View())

	return lipgloss.JoinVertical(lipgloss.Top, list, main, helpBar)
}

func main() {
	config, err := config.Parse()
	if err != nil {
		log.Fatalf("Error loading config: %s", err)
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
