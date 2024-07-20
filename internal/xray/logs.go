package xray

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

type LogData struct {
	Results *cloudwatchlogs.GetQueryResultsOutput
}

type LogQueryID string

func StartLogsQuery(ctx context.Context, logGroupName string, id TraceID) (*LogQueryID, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration, %w", err)
	}
	client := cloudwatchlogs.NewFromConfig(cfg)

	end := time.Now().Unix()
	start := time.Now().Add(-24 * time.Hour).Unix()
	query := fmt.Sprintf("fields @log, @timestamp, @message | filter @message like \"%s\" | sort @timestamp desc", id)
	params := cloudwatchlogs.StartQueryInput{
		QueryString:  &query,
		StartTime:    &start,
		EndTime:      &end,
		LogGroupName: &logGroupName,
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
