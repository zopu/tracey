package ui

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	"github.com/samber/lo"
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

type DetailsPane struct {
	LogFields []config.ParsedLogField
	Logs      mo.Option[aws.LogData]
	focused   bool
	Width     int
	Height    int
	timeline  mo.Option[table.Model]
}

func (d *DetailsPane) SetFocus(focus bool) {
	d.focused = focus
}

func (d *DetailsPane) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case TraceDetailsMsg:
		d.updateTimeline(*msg.Trace)
		d.Logs = mo.None[aws.LogData]()
		if msg.LogsQueryID != nil {
			return FetchLogs(*msg.LogsQueryID, time.Second)
		}
	case ClearTraceDetailsMsg:
		d.timeline = mo.None[table.Model]()
		d.Logs = mo.None[aws.LogData]()
	}
	return nil
}

type timeLineRow struct {
	startTime time.Duration
	duration  time.Duration
	details   []string
}

func (d *DetailsPane) updateTimeline(td aws.TraceDetails) {
	rows := make([]timeLineRow, 0)
	for _, segment := range td.Segments {
		if segment.ParentID != "" {
			continue
		}

		details := []string{
			fmt.Sprintf("%s %s", segment.Name, segment.Origin),
		}

		segment.SQL.ForEach(func(sql aws.SQL) {
			re := regexp.MustCompile(`\s+`)
			q := re.ReplaceAllString(sql.SanitizedQuery, " ")
			truncated := fmt.Sprintf("SQL Query: %.150s", q)
			details = append(details, truncated)
		})

		duration := segment.EndTime.Time().Sub(segment.StartTime.Time())
		rows = append(rows, timeLineRow{
			startTime: time.Duration(0),
			duration:  duration,
			details:   details,
		})

		for _, subsegment := range segment.SubSegments {
			rows = append(rows, getSubsegmentRows(subsegment, segment.StartTime.Time())...)
		}
	}

	columns := []table.Column{
		table.NewColumn("Start Time", "Start Time", 15),
		table.NewColumn("Duration", "Duration", 15),
		table.NewColumn("Details", "Details", d.Width-34),
	}
	tableRows := lo.Map(rows, func(row timeLineRow, _ int) table.Row {
		return table.NewRow(table.RowData{
			"Start Time": row.startTime.String(),
			"Duration":   row.duration.String(),
			"Details":    strings.Join(row.details, "\n"),
		})
	})
	t := table.New(columns).
		WithRows(tableRows).
		WithMultiline(true).
		WithBaseStyle(
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("#c6d0f5")).
				BorderForeground(lipgloss.Color("240")).
				Bold(false)).
		HeaderStyle(
			lipgloss.NewStyle().
				Bold(true)).
		HighlightStyle(
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("#c6d0f5")).
				Background(lipgloss.Color("#414559")))
	d.timeline = mo.Some(t)
}

func getSubsegmentRows(subsegment aws.SubSegment, startTime time.Time) []timeLineRow {
	rows := make([]timeLineRow, 0)
	timeOffset := subsegment.StartTime.Time().Sub(startTime)
	duration := subsegment.EndTime.Time().Sub(subsegment.StartTime.Time())
	details := []string{
		subsegment.Name,
	}
	subsegment.SQL.ForEach(func(sql aws.SQL) {
		re := regexp.MustCompile(`\s+`)
		q := re.ReplaceAllString(sql.SanitizedQuery, " ")
		if len(q) > 0 {
			truncated := fmt.Sprintf("SQL Query: %.150s", q)
			details = append(details, truncated)
		}
	})
	rows = append(rows, timeLineRow{
		startTime: timeOffset,
		duration:  duration,
		details:   details,
	})
	for _, subsegment := range subsegment.SubSegments {
		rows = append(rows, getSubsegmentRows(subsegment, startTime)...)
	}
	return rows
}

func (d DetailsPane) View() string {
	if !d.timeline.IsPresent() {
		s := "Select a trace to view"
		for range d.Height - 3 {
			s += "\n"
		}
		return s
	}

	s := "Timeline:\n"
	s += d.timeline.MustGet().View()
	s += "\n"

	d.Logs.ForEach(func(logs aws.LogData) {
		if !logs.IsEmpty() {
			s += "Logs:\n"
			s += ViewLogs(logs, d.LogFields, d.Width)
		}
	})

	style := lipgloss.NewStyle().
		Width(d.Width).
		Height(d.Height - 2).
		MaxHeight(d.Height)
	if d.focused {
		style = style.BorderForeground(lipgloss.Color("63"))
	}
	return style.Render(s)
}
