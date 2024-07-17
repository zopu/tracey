package xray

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/xray"
	"github.com/aws/aws-sdk-go-v2/service/xray/types"
	"github.com/samber/lo"
)

type TraceSummary struct {
	ID string
}

func (t TraceSummary) FilterValue() string {
	return t.ID
}

func FetchTraceSummaries(ctx context.Context) ([]TraceSummary, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration, %w", err)
	}
	client := xray.NewFromConfig(cfg)

	start := time.Now().Add(-6 * time.Hour)
	end := time.Now()
	resp, err := client.GetTraceSummaries(ctx, &xray.GetTraceSummariesInput{
		EndTime:   &end,
		StartTime: &start,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get trace summaries, %w", err)
	}

	return lo.Map(resp.TraceSummaries, func(ts types.TraceSummary, _ int) TraceSummary {
		return TraceSummary{ID: *ts.Id}
	}), nil
}
