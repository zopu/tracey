package ui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	"github.com/samber/lo"
	"github.com/zopu/tracey/internal/aws"
)

type timeline struct {
	tableModel table.Model
}

func (t timeline) View() string {
	return t.tableModel.View()
}

func newTimeline(td aws.TraceDetails, width int) timeline {
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
		table.NewColumn("Details", "Details", width-34),
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
				Bold(true))
	return timeline{tableModel: t}
}

func (t timeline) Update(msg tea.Msg) (timeline, tea.Cmd) {
	switch msg := msg.(type) { //nolint:gocritic // standard pattern
	case tea.KeyMsg:
		tb, cmd := t.tableModel.Update(msg)
		return timeline{tableModel: tb}, cmd
	}
	return t, nil
}

func (t timeline) SetFocus(focus bool) timeline {
	color := "240"
	if focus {
		color = "63"
	}
	t.tableModel = t.tableModel.WithBaseStyle(
		lipgloss.NewStyle().BorderForeground(lipgloss.Color(color)).
			Foreground(lipgloss.Color("#c6d0f5")).
			Bold(false)).
		Focused(focus)
	return t
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
