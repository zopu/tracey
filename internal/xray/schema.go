package xray

import (
	"encoding/json"
	"time"

	"github.com/samber/mo"
)

// Capturing the schema described here:
// https://docs.aws.amazon.com/xray/latest/devguide/xray-api-segmentdocuments.html

type Time time.Time

func (t *Time) UnmarshalJSON(b []byte) error {
	var ft float64
	err := json.Unmarshal(b, &ft)
	if err != nil {
		return err
	}
	tm := time.Unix(0, int64(ft*float64(time.Second)))
	*t = Time(tm)
	return nil
}

func (t Time) Time() time.Time {
	return time.Time(t)
}

type Segment struct {
	// Required fields
	//
	Name string `json:"name"`
	ID   string `json:"id"`

	StartTime Time `json:"start_time"`
	EndTime   Time `json:"end_time"`

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
	Metadata    map[string]any `json:"metadata,omitempty"`
	SubSegments []SubSegment   `json:"subsegments,omitempty"`

	// Not part of the schema but found in practice
	SQL mo.Option[SQL] `json:"sql,omitempty"`
}

type SubSegment struct {
	// Required fields
	//
	Name string `json:"name"`
	ID   string `json:"id"`

	StartTime Time `json:"start_time"`
	EndTime   Time `json:"end_time"`

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
	Metadata    map[string]any `json:"metadata,omitempty"`
	SubSegments []SubSegment   `json:"subsegments,omitempty"`

	// Not part of the schema but found in practice
	SQL mo.Option[SQL] `json:"sql,omitempty"`
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

type SQL struct {
	ConnectionString string `json:"connection_string"`
	URL              string `json:"url"`
	SanitizedQuery   string `json:"sanitized_query"`
	DatabaseType     string `json:"database_type"`
	User             string `json:"user"`
}
