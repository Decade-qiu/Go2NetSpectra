package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/query"
)

type stubQuerier struct {
	aggregateResp *query.QueryTotalCountsResponse
}

func (s *stubQuerier) AggregateFlows(ctx context.Context, req *query.AggregationRequest) (*query.QueryTotalCountsResponse, error) {
	return s.aggregateResp, nil
}

func (s *stubQuerier) TraceFlow(ctx context.Context, req *query.TraceFlowRequest) (*query.FlowLifecycle, error) {
	return nil, nil
}

func (s *stubQuerier) QueryHeavyHitters(ctx context.Context, req *query.HeavyHittersRequest) (*query.HeavyHittersResponse, error) {
	return nil, nil
}

func TestRunLegacyHTTPServerReturnsUnsupportedError(t *testing.T) {
	err := RunLegacyHTTPServer(context.Background(), &config.Config{})
	if err == nil {
		t.Fatal("RunLegacyHTTPServer() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "no longer supported") {
		t.Fatalf("RunLegacyHTTPServer() error = %q, want substring %q", err.Error(), "no longer supported")
	}
}

func TestGrafanaHTTPHandlerAggregatesTaskPackets(t *testing.T) {
	handler := newGrafanaHTTPHandler(&QueryServiceServer{
		exactQuerier: &stubQuerier{
			aggregateResp: &query.QueryTotalCountsResponse{
				Summaries: []query.TaskSummary{
					{
						TaskName:     "demo-task",
						TotalPackets: 123,
					},
				},
			},
		},
	})

	requestBody := map[string]any{
		"targets": []map[string]string{
			{"target": "demo-task"},
		},
		"range": map[string]string{
			"to": time.Unix(1700000000, 0).UTC().Format(time.RFC3339),
		},
	}

	data, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("json.Marshal() unexpected error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/query", bytes.NewReader(data))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("ServeHTTP() status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var resp []timeSeriesResponse
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("Decode() unexpected error: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("response length = %d, want 1", len(resp))
	}
	if got := resp[0].Target; got != "demo-task" {
		t.Fatalf("response target = %q, want %q", got, "demo-task")
	}
	if len(resp[0].Datapoints) != 1 {
		t.Fatalf("datapoint length = %d, want 1", len(resp[0].Datapoints))
	}
	if got := resp[0].Datapoints[0][0]; got != 123 {
		t.Fatalf("packet datapoint = %v, want %v", got, float64(123))
	}
}
