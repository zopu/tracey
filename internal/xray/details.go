package xray

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/xray"
	"github.com/aws/aws-sdk-go-v2/service/xray/types"
)

type TraceDetails struct {
	ID       TraceID
	Segments []Segment
}

func (t TraceDetails) String() string {
	return string(t.ID)
}

func parseTrace(trace types.Trace) (*TraceDetails, error) {
	segments := make([]Segment, len(trace.Segments))
	for i, seg := range trace.Segments {
		err := json.Unmarshal([]byte(*seg.Document), &segments[i])
		if err != nil {
			return nil, fmt.Errorf("failed to parse segment: %w", err)
		}
	}
	sort.Slice(segments, func(i, j int) bool {
		return segments[i].StartTime.Time().Before(segments[j].StartTime.Time())
	})
	for _, segment := range segments {
		sort.Slice(segment.SubSegments, func(i, j int) bool {
			return segment.SubSegments[i].StartTime.Time().Before(segment.SubSegments[j].StartTime.Time())
		})
	}
	return &TraceDetails{
		ID:       TraceID(*trace.Id),
		Segments: segments,
	}, nil
}

func FetchTraceDetails(ctx context.Context, id TraceID) (*TraceDetails, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration, %w", err)
	}
	client := xray.NewFromConfig(cfg)

	resp, err := client.BatchGetTraces(ctx, &xray.BatchGetTracesInput{
		TraceIds: []string{string(id)},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get trace details, %w", err)
	}
	if len(resp.Traces) == 0 {
		return nil, fmt.Errorf("trace not found: %s", id)
	}
	parsed, err := parseTrace(resp.Traces[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse trace: %w", err)
	}
	return parsed, nil
}
