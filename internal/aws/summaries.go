package aws

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
	"github.com/samber/mo"
)

type SummaryData struct {
	NextToken mo.Option[string]
	Summaries []TraceSummary
}

type TraceSummary struct {
	Data types.TraceSummary
}

func (t TraceSummary) Title() string {
	startTime := t.Data.StartTime.Format("01-02 15:04:05")
	if t.Data.Http == nil {
		return "(no trace http data)"
	}
	ip := "(no client ip)"
	if t.Data.Http.ClientIp != nil {
		ip = *t.Data.Http.ClientIp
	}
	title := fmt.Sprintf(
		"%s %v %s (%d) %s %vms %s",
		*t.Data.Id,
		startTime,
		ip,
		*t.Data.Http.HttpStatus,
		*t.Data.Http.HttpMethod,
		*t.Data.ResponseTime*1000,
		t.Path(),
	)
	return title
}

func (t TraceSummary) ID() string {
	return *t.Data.Id
}

func (t TraceSummary) Path() string {
	u, err := url.Parse(*t.Data.Http.HttpURL)
	if err != nil {
		return ""
	}
	return u.Path
}

func (t TraceSummary) FilterValue() string {
	return fmt.Sprintf(
		"%s %d %s %s",
		*t.Data.Id,
		*t.Data.Http.HttpStatus,
		*t.Data.Http.HttpMethod,
		t.Path(),
	)
}

func (t TraceSummary) HasError() bool {
	status := *t.Data.Http.HttpStatus
	return status >= 400 && status < 500
}

func (t TraceSummary) HasFault() bool {
	status := *t.Data.Http.HttpStatus
	return status >= 500 && status < 600
}

func FetchTraceSummaries(
	ctx context.Context,
	nextToken mo.Option[string],
) (*SummaryData, error) {
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
		NextToken: nextToken.ToPointer(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get trace summaries, %w", err)
	}

	summaries := lo.Map(resp.TraceSummaries, func(ts types.TraceSummary, _ int) TraceSummary {
		return TraceSummary{Data: ts}
	})
	sort.Slice(summaries, func(i, j int) bool {
		return !summaries[i].Data.StartTime.Before(*summaries[j].Data.StartTime)
	})
	result := SummaryData{Summaries: summaries}
	if resp.NextToken != nil {
		result.NextToken = mo.Some(*resp.NextToken)
	}
	return &result, nil
}
