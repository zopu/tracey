package store_test

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/xray/types"
	"github.com/zopu/tracey/internal/aws"
	"github.com/zopu/tracey/internal/store"
)

func TestSummaryDeduplication(t *testing.T) {
	ids := []string{"1234", "5678", "9012"}
	ts := []aws.TraceSummary{
		{
			Data: types.TraceSummary{
				Id: &ids[0],
			},
		},
		{
			Data: types.TraceSummary{
				Id: &ids[1],
			},
		},
	}
	st := store.New()
	st.AddTraceSummaries(ts)
	got := st.GetTraceSummaries()
	if len(got) != len(ts) {
		t.Errorf("Expected %d traces, got %d", len(ts), len(got))
	}

	ts = []aws.TraceSummary{
		{
			Data: types.TraceSummary{
				Id: &ids[1],
			},
		},
		{
			Data: types.TraceSummary{
				Id: &ids[2],
			},
		},
	}
	st.AddTraceSummaries(ts)
	got = st.GetTraceSummaries()
	if len(got) != 3 {
		t.Errorf("Expected 3 traces, got %d", len(got))
	}
}
