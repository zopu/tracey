package ui

import (
	"fmt"
	"regexp"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/samber/mo"
	"github.com/zopu/tracey/internal/xray"
)

type DetailsPane struct {
	Details mo.Option[xray.TraceDetails]
	Logs    mo.Option[xray.LogData]
	Width   int
	Height  int
}

func (d DetailsPane) View() string {
	if !d.Details.IsPresent() {
		return "Select a trace to view\n\n"
	}
	td := d.Details.MustGet()

	s := string(td.ID) + "\n\n"

	for _, segment := range td.Segments {
		duration := segment.EndTime.Time().Sub(segment.StartTime.Time())
		s += fmt.Sprintf("%s\t%s (%s)\n", segment.Origin, segment.Name, duration.String())

		for _, subsegment := range segment.SubSegments {
			s += viewSubsegment(subsegment)
		}

		segment.SQL.ForEach(func(sql xray.SQL) {
			re := regexp.MustCompile(`\s+`)
			q := re.ReplaceAllString(sql.SanitizedQuery, " ")
			s += fmt.Sprintf("Query: %.150s\n", q)
		})

		s += "\n"
	}

	d.Logs.ForEach(func(logs xray.LogData) {
		s += "Logs:\n"
		s += fmt.Sprintf("Status: %s\n", logs.Results.Status)
		for _, event := range logs.Results.Results {
			for _, field := range event {
				s += fmt.Sprintf("%s: %s\n", *field.Field, *field.Value)
			}
		}
		s += "\n"
	})

	style := lipgloss.NewStyle().
		Width(d.Width - 2).
		Height(d.Height - 4).
		MaxHeight(d.Height - 2).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63"))
	return style.Render(s)
}

func viewSubsegment(subsegment xray.SubSegment) string {
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
