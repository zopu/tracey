package ui

import (
	"context"
	"fmt"
	"regexp"
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
	Details   mo.Option[aws.TraceDetails]
	Logs      mo.Option[aws.LogData]
	focused   bool
	Width     int
	Height    int
}

func (d *DetailsPane) SetFocus(focus bool) {
	d.focused = focus
}

func (d *DetailsPane) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) { //nolint:gocritic // Standard pattern for messages
	case TraceDetailsMsg:
		d.Details = mo.Some(*msg.Trace)
		d.Logs = mo.None[aws.LogData]()
		if msg.LogsQueryID != nil {
			return FetchLogs(*msg.LogsQueryID, time.Second)
		}
	}
	return nil
}

func (d DetailsPane) View() string {
	if !d.Details.IsPresent() {
		s := "Select a trace to view"
		for range d.Height - 3 {
			s += "\n"
		}
		return s
	}
	td := d.Details.MustGet()

	s := ""

	for _, segment := range td.Segments {
		if segment.ParentID != "" {
			continue
		}

		duration := segment.EndTime.Time().Sub(segment.StartTime.Time())
		s += fmt.Sprintf("%s\t%s (%s)\n", segment.Origin, segment.Name, duration.String())

		for _, subsegment := range segment.SubSegments {
			s += viewSubsegment(subsegment)
		}

		segment.SQL.ForEach(func(sql aws.SQL) {
			re := regexp.MustCompile(`\s+`)
			q := re.ReplaceAllString(sql.SanitizedQuery, " ")
			s += fmt.Sprintf("Query: %.150s\n", q)
		})

		s += "\n"
	}

	d.Logs.ForEach(func(logs aws.LogData) {
		if !logs.IsEmpty() {
			s += "Logs:\n\n"
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

func viewSubsegment(subsegment aws.SubSegment) string {
	duration := subsegment.EndTime.Time().Sub(subsegment.StartTime.Time())
	s := fmt.Sprintf("%s\t%s\t%s\n",
		subsegment.StartTime.Time().Format(time.StampMilli),
		duration.String(),
		subsegment.Name,
	)
	subsegment.SQL.ForEach(func(sql aws.SQL) {
		re := regexp.MustCompile(`\s+`)
		q := re.ReplaceAllString(sql.SanitizedQuery, " ")
		if len(q) > 0 {
			s += fmt.Sprintf("\t\t\t\tQuery: %.150s\n", q)
		}
	})
	for _, subsegment := range subsegment.SubSegments {
		s += viewSubsegment(subsegment)
	}
	return s
}
