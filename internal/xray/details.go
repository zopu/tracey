package xray

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/xray"
	"github.com/aws/aws-sdk-go-v2/service/xray/types"
)

type TraceDetails struct {
	trace types.Trace
}

func (t TraceDetails) Segments() []types.Segment {
	return t.trace.Segments
}

func (t TraceDetails) String() string {
	return *t.trace.Id
}

func FetchTraceDetails(ctx context.Context, traceID string) (*TraceDetails, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration, %w", err)
	}
	client := xray.NewFromConfig(cfg)

	resp, err := client.BatchGetTraces(ctx, &xray.BatchGetTracesInput{
		TraceIds: []string{traceID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get trace details, %w", err)
	}
	if len(resp.Traces) == 0 {
		return nil, fmt.Errorf("trace not found")
	}
	return &TraceDetails{trace: resp.Traces[0]}, nil
}
