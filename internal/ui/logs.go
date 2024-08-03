package ui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
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
	if len(fields) == 0 {
		return ""
	}

	widths := lo.Map(fields, func(f config.ParsedLogField, _ int) int {
		return len(f.Title)
	})

	rows := make([]table.Row, 0)
	for _, event := range logs.Results.Results {
		for _, field := range event {
			if *field.Field == "@message" { //nolint:nestif //Work in progress
				var unmarshalled map[string]any
				err := json.Unmarshal([]byte(*field.Value), &unmarshalled)
				if err != nil {
					log.Fatalf("failed to unmarshal json: %v", err)
				}
				row := make(table.RowData, len(fields))
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
						rowKey := strconv.Itoa(i)
						val := strings.TrimSpace(fmt.Sprintf("%#s", v))
						row[rowKey] = val
						width := len(val)
						if widths[i] < width {
							widths[i] = width
						}
					}
				}
				rows = append(rows, table.NewRow(row))
			}
		}
	}

	// Extend last column to fill the width of the table
	totalWidth := 0
	for _, w := range widths {
		totalWidth += w
	}
	if totalWidth < tableWidth-4 {
		widths[len(widths)-1] += tableWidth - totalWidth - 4
	}

	columns := lo.Map(fields, func(f config.ParsedLogField, i int) table.Column {
		return table.NewColumn(strconv.Itoa(i), f.Title, widths[i])
	})

	t := table.New(columns).
		WithTargetWidth(tableWidth).
		WithRows(rows).
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
	s := t.View() + "\n"
	return s
}
