package xray

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/xray"
	"github.com/aws/aws-sdk-go-v2/service/xray/types"
)

type Segment struct {
	// Required fields
	//
	Name string `json:"name"`
	ID   string `json:"id"`

	// TODO: Parse these with a wrapper around time.Time
	StartTime float64 `json:"start_time"`
	EndTime   float64 `json:"end_time"`

	TraceID string `json:"trace_id"`

	// Optional fields
	//
	Service     map[string]any `json:"service,omitempty"`
	User        map[string]any `json:"user,omitempty"`
	Origin      string         `json:"origin,omitempty"`
	ParentID    string         `json:"parent_id,omitempty"`
	HTTP        SegmentHTTP    `json:"http,omitempty"`
	Aws         SegmentAWS     `json:"aws,omitempty"`
	Error       bool           `json:"error,omitempty"`
	Throttle    bool           `json:"throttle,omitempty"`
	Fault       bool           `json:"fault,omitempty"`
	Cause       any            `json:"cause,omitempty"`
	Annotations map[string]any `json:"annotations,omitempty"`
	Metatada    map[string]any `json:"metadata,omitempty"`
	SubSegments []SubSegment   `json:"subsegments,omitempty"`
}

type SubSegment struct {
	// Required fields
	//
	Name string `json:"name"`
	ID   string `json:"id"`

	// TODO: Parse these with a wrapper around time.Time
	StartTime float64 `json:"start_time"`
	EndTime   float64 `json:"end_time"`

	InProgress bool   `json:"in_progress"`
	TraceID    string `json:"trace_id"`
	ParentID   string `json:"parent_id"`

	// Optional fields
	//
	Namespace   string         `json:"namespace,omitempty"`
	HTTP        SegmentHTTP    `json:"http,omitempty"`
	Aws         SegmentAWS     `json:"aws,omitempty"`
	Error       bool           `json:"error,omitempty"`
	Throttle    bool           `json:"throttle,omitempty"`
	Fault       bool           `json:"fault,omitempty"`
	Cause       any            `json:"cause,omitempty"`
	Annotations map[string]any `json:"annotations,omitempty"`
	Metatada    map[string]any `json:"metadata,omitempty"`
	SubSegments []SubSegment   `json:"subsegments,omitempty"`
}

type SegmentHTTP struct {
	Request  SegmentHTTPRequest  `json:"request,omitempty"`
	Response SegmentHTTPResponse `json:"response,omitempty"`
}

type SegmentHTTPRequest struct {
	Method        string `json:"method,omitempty"`
	URL           string `json:"url,omitempty"`
	UserAgent     string `json:"user_agent,omitempty"`
	ClientIP      string `json:"client_ip,omitempty"`
	XForwardedFor bool   `json:"x_forwarded_for,omitempty"`
	Traced        bool   `json:"traced,omitempty"`
}

type SegmentHTTPResponse struct {
	Status        int `json:"status,omitempty"`
	ContentLength int `json:"content_length,omitempty"`
}

type SegmentAWS struct {
	Operation string `json:"operation"`
	AccountID string `json:"account_id"`
	Region    string `json:"region"`
	RequestID string `json:"request_id"`
	QueueURL  string `json:"queue_url"`
	TableName string `json:"table_name"`
}

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
