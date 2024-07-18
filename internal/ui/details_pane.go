package ui

import (
	"fmt"

	"github.com/samber/mo"
	"github.com/zopu/tracey/internal/xray"
)

type DetailsPane struct {
	Details mo.Option[xray.TraceDetails]
}

func (d DetailsPane) View() string {
	if !d.Details.IsPresent() {
		return "Select a trace to view\n\n"
	}
	td := d.Details.MustGet()

	s := fmt.Sprintf("Selected trace: %s\n", string(td.ID))
	s += fmt.Sprintf("\n\nSegments: %d\n", len(td.Segments))
	for _, segment := range td.Segments {
		s += fmt.Sprintf("Name: %s, Subsegments: %d\nValue: %.40v\n\n", segment.Name, len(segment.SubSegments), segment)
	}
	return s
}
