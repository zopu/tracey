package ui

import (
	"fmt"
	"regexp"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/samber/mo"
	"github.com/zopu/tracey/internal/aws"
	"github.com/zopu/tracey/internal/config"
)

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

func (d *DetailsPane) Update(_ tea.Msg) tea.Cmd {
	return nil
}

func (d DetailsPane) View() string {
	if !d.Details.IsPresent() {
		return "Select a trace to view\n\n"
	}
	td := d.Details.MustGet()

	s := ""

	// Find a Client IP
	var clientIP *string
	for _, segment := range td.Segments {
		if segment.HTTP.Request.ClientIP != "" {
			ip := segment.HTTP.Request.ClientIP
			clientIP = &ip
			break
		}
	}
	if clientIP != nil {
		s += fmt.Sprintf("Client: %s\n\n", *clientIP)
	}

	for _, segment := range td.Segments {
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
		s += "Logs:\n"
		s += ViewLogs(logs, d.LogFields, d.Width-8)
	})

	style := lipgloss.NewStyle().
		Width(d.Width - 2).
		Height(d.Height - 4).
		MaxHeight(d.Height - 2).
		BorderStyle(lipgloss.RoundedBorder())
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
	for _, subsegment := range subsegment.SubSegments {
		s += viewSubsegment(subsegment)
	}
	return s
}
