package xray

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/xray"
	"github.com/aws/aws-sdk-go-v2/service/xray/types"
	"github.com/samber/lo"
)

type TraceSummary struct {
	summary types.TraceSummary
}

func (t TraceSummary) Title() string {
	title := fmt.Sprintf(
		"%s %v (%d) %s %vms %s",
		*t.summary.Id,
		*t.summary.StartTime,
		*t.summary.Http.HttpStatus,
		*t.summary.Http.HttpMethod,
		*t.summary.ResponseTime*1000,
		t.Path(),
	)
	return title
}

func (t TraceSummary) ID() string {
	return *t.summary.Id
}

func (t TraceSummary) Path() string {
	u, err := url.Parse(*t.summary.Http.HttpURL)
	if err != nil {
		return ""
	}
	return u.Path
}

func (t TraceSummary) FilterValue() string {
	return fmt.Sprintf(
		"%s %d %s %s",
		*t.summary.Id,
		*t.summary.Http.HttpStatus,
		*t.summary.Http.HttpMethod,
		t.Path(),
	)
}

func (t TraceSummary) HasError() bool {
	status := *t.summary.Http.HttpStatus
	return status >= 400 && status < 500
}

func (t TraceSummary) HasFault() bool {
	status := *t.summary.Http.HttpStatus
	return status >= 500 && status < 600
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

	summaries := lo.Map(resp.TraceSummaries, func(ts types.TraceSummary, _ int) TraceSummary {
		return TraceSummary{summary: ts}
	})
	sort.Slice(summaries, func(i, j int) bool {
		return !summaries[i].summary.StartTime.Before(*summaries[j].summary.StartTime)
	})
	return summaries, nil
}
