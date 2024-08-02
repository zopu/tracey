package ui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/itchyny/gojq"
	"github.com/samber/lo"
	"github.com/zopu/tracey/internal/aws"
	"github.com/zopu/tracey/internal/config"
)

type TraceLogsMsg struct {
	Logs *aws.LogData
}

func FetchLogs(id aws.LogQueryID, delay time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(delay)
		logs, err := aws.FetchLogs(context.Background(), id)
		if err != nil {
			return ErrorMsg{Msg: err.Error()}
		}
		return TraceLogsMsg{Logs: logs}
	}
}

//nolint:gocognit // Work in progress
func ViewLogs(logs aws.LogData, fields []config.ParsedLogField, tableWidth int) string {
	s := fmt.Sprintf("Status: %s\n\n", logs.Results.Status)

	columns := lo.Map(fields, func(f config.ParsedLogField, _ int) table.Column {
		return table.Column{
			Title: f.Title,
			Width: len(f.Title),
		}
	})

	if len(columns) == 0 {
		return s
	}

	rows := make([]table.Row, 0)
	for _, event := range logs.Results.Results {
		for _, field := range event {
			if *field.Field == "@message" { //nolint:nestif //Work in progress
				var unmarshalled map[string]any
				err := json.Unmarshal([]byte(*field.Value), &unmarshalled)
				if err != nil {
					log.Fatalf("failed to unmarshal json: %v", err)
				}
				row := make(table.Row, len(columns))
				for i, field := range fields {
					it := field.Query.Run(unmarshalled)
					for {
						v, ok := it.Next()
						if !ok {
							break
						}
						if jqErr, aok := v.(error); aok {
							if errors.Is(jqErr, &gojq.HaltError{}) {
								break
							}
							log.Fatalln(jqErr)
						}
						row[i] = fmt.Sprintf("%#s", v)
						width := len(row[i]) + 2
						if columns[i].Width < width {
							columns[i].Width = width
						}
					}
				}
				rows = append(rows, row)
			}
		}
	}

	// Extend last column to fill the width of the table
	totalWidth := 0
	for _, column := range columns {
		totalWidth += column.Width
	}
	if totalWidth < tableWidth {
		columns[len(columns)-1].Width += tableWidth - totalWidth
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
	)

	style := table.DefaultStyles()
	style.Header = style.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	style.Selected = style.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(style)
	s += t.View()

	s += "\n"
	return s
}
