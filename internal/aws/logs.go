package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/samber/lo"
)

type LogData struct {
	Results *cloudwatchlogs.GetQueryResultsOutput
}

type LogQueryID string

func StartLogsQuery(ctx context.Context, logGroupNames []string, id TraceID) (*LogQueryID, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration, %w", err)
	}
	client := cloudwatchlogs.NewFromConfig(cfg)

	end := time.Now().Unix()
	start := time.Now().Add(-24 * time.Hour).Unix()
	query := fmt.Sprintf("fields @log, @timestamp, @message | filter @message like \"%s\" | sort @timestamp desc", id)
	params := cloudwatchlogs.StartQueryInput{
		QueryString:   &query,
		StartTime:     &start,
		EndTime:       &end,
		LogGroupNames: logGroupNames,
	}

	output, err := client.StartQuery(ctx, &params)
	if err != nil {
		return nil, fmt.Errorf("failed to start query, %w", err)
	}
	result := LogQueryID(*output.QueryId)
	return &result, nil
}

func FetchLogs(ctx context.Context, queryID LogQueryID) (*LogData, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration, %w", err)
	}
	client := cloudwatchlogs.NewFromConfig(cfg)
	q := string(queryID)
	resultsParams := cloudwatchlogs.GetQueryResultsInput{
		QueryId: &q,
	}
	results, err := client.GetQueryResults(ctx, &resultsParams)
	if err != nil {
		return nil, fmt.Errorf("failed to get query results, %w", err)
	}

	return &LogData{Results: results}, nil
}

func GetLogGroups(ctx context.Context) ([]string, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration, %w", err)
	}
	client := cloudwatchlogs.NewFromConfig(cfg)

	// TODO: Handle pagination
	params := cloudwatchlogs.DescribeLogGroupsInput{}

	resp, err := client.DescribeLogGroups(ctx, &params)
	if err != nil {
		return nil, fmt.Errorf("failed to get log groups: %w", err)
	}

	groups := lo.Map(resp.LogGroups, func(g types.LogGroup, _ int) string {
		return *g.LogGroupName
	})
	return groups, nil
}
