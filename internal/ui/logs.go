package ui

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/itchyny/gojq"
	"github.com/zopu/tracey/internal/xray"
)

//nolint:gocognit // Work in progress
func ViewLogs(logs xray.LogData, queries []gojq.Query) string {
	s := fmt.Sprintf("Status: %s\n", logs.Results.Status)
	for _, event := range logs.Results.Results {
		for _, field := range event {
			if *field.Field == "@message" { //nolint:nestif //Work in progress
				var unmarshalled map[string]any
				err := json.Unmarshal([]byte(*field.Value), &unmarshalled)
				if err != nil {
					log.Fatalf("failed to unmarshal json: %v", err)
				}
				for _, query := range queries {
					it := query.Run(unmarshalled)
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
						s += fmt.Sprintf("%#v\n", v)
					}
				}
			}
		}
	}
	s += "\n"
	return s
}
