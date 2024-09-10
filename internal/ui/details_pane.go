package ui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/samber/mo"
	"github.com/zopu/tracey/internal/aws"
	"github.com/zopu/tracey/internal/config"
)

type TraceDetailsMsg struct {
	Trace       *aws.TraceDetails
	LogsQueryID *aws.LogQueryID
}

type ClearTraceDetailsMsg struct{}

func FetchTraceDetails(id aws.TraceID, logGroupNames []string) tea.Cmd {
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
		return TraceDetailsMsg{Trace: details, LogsQueryID: logsQueryID}
	}
}

const (
	detailSelectedNone = iota
	detailSelectedTimeline
	detailSelectedLogs
)

type DetailsPane struct {
	LogFields     []config.ParsedLogField
	Logs          mo.Option[aws.LogData]
	focused       bool
	Width         int
	timeline      mo.Option[timeline]
	selectedTable int
}

func (d *DetailsPane) SetFocus(focus bool) {
	d.focused = focus
	if focus {
		d.selectedTable = detailSelectedTimeline
		d.SetTimelineFocus(true)
		return
	}
	d.selectedTable = detailSelectedNone
}

func (d *DetailsPane) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case TraceDetailsMsg:
		d.timeline = mo.Some(newTimeline(*msg.Trace, d.Width))
		d.Logs = mo.None[aws.LogData]()
		if msg.LogsQueryID != nil {
			return FetchLogs(*msg.LogsQueryID, time.Second)
		}
	case ClearTraceDetailsMsg:
		d.timeline = mo.None[timeline]()
		d.Logs = mo.None[aws.LogData]()
	case tea.KeyMsg:
		switch msg.String() { //nolint:gocritic // standard pattern
		case "tab":
			switch d.selectedTable {
			case detailSelectedNone:
				d.selectedTable = detailSelectedTimeline
				d.SetTimelineFocus(true)
			case detailSelectedTimeline:
				d.SetTimelineFocus(false)
				if d.Logs.IsPresent() &&
					d.Logs.MustGet().Results != nil &&
					len(d.Logs.MustGet().Results.Results) > 0 {
					d.selectedTable = detailSelectedLogs
					return nil
				}
				return func() tea.Msg {
					return SelectNextPaneMsg{}
				}
			case detailSelectedLogs:
				d.selectedTable = detailSelectedNone
				d.SetTimelineFocus(false)
				return func() tea.Msg {
					return SelectNextPaneMsg{}
				}
			}
		}
	}
	if d.selectedTable == detailSelectedTimeline && d.timeline.IsPresent() {
		t, cmd := d.timeline.MustGet().Update(msg)
		d.timeline = mo.Some(t)
		return cmd
	}
	return nil
}

func (d *DetailsPane) SetTimelineFocus(focus bool) {
	d.timeline = d.timeline.Map(func(t timeline) (timeline, bool) {
		return t.SetFocus(focus), true
	})
}

type timeLineRow struct {
	startTime time.Duration
	duration  time.Duration
	details   []string
}

func (d DetailsPane) View() string {
	if !d.timeline.IsPresent() {
		s := "Select a trace to view"
		return s
	}

	s := "Timeline:\n"
	s += d.timeline.MustGet().View()
	s += "\n"

	d.Logs.ForEach(func(logs aws.LogData) {
		if !logs.IsEmpty() {
			s += "Logs:\n"
			logsFocused := d.selectedTable == detailSelectedLogs
			s += ViewLogs(logs, d.LogFields, d.Width, logsFocused)
		}
	})

	style := lipgloss.NewStyle()
	if d.focused {
		style = style.BorderForeground(lipgloss.Color("63"))
	}
	return style.Render(s)
}
