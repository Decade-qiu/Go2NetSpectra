package query

import (
	"math"
	"reflect"
	"testing"
)

func TestAppendTraceFlowFiltersSortsFlowKeys(t *testing.T) {
	whereClauses, args, err := appendTraceFlowFilters(
		[]string{"TaskName = ?"},
		[]any{"demo-task"},
		map[string]string{
			"SrcIP":    "10.0.0.1",
			"Protocol": "6",
			"DstPort":  "443",
		},
	)
	if err != nil {
		t.Fatalf("appendTraceFlowFilters() unexpected error: %v", err)
	}

	wantClauses := []string{
		"TaskName = ?",
		"DstPort = ?",
		"Protocol = ?",
		"SrcIP = ?",
	}
	if !reflect.DeepEqual(whereClauses, wantClauses) {
		t.Fatalf("appendTraceFlowFilters() clauses = %#v, want %#v", whereClauses, wantClauses)
	}

	wantArgs := []any{"demo-task", "443", "6", "10.0.0.1"}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("appendTraceFlowFilters() args = %#v, want %#v", args, wantArgs)
	}
}

func TestAppendTraceFlowFiltersRejectsUnsupportedKeys(t *testing.T) {
	_, _, err := appendTraceFlowFilters(nil, nil, map[string]string{
		"DropTable": "true",
	})
	if err == nil {
		t.Fatal("appendTraceFlowFilters() error = nil, want non-nil")
	}

	want := "unsupported flow key: DropTable"
	if got := err.Error(); got != want {
		t.Fatalf("appendTraceFlowFilters() error = %q, want %q", got, want)
	}
}

func TestAppendAggregationFiltersIncludesSupportedFields(t *testing.T) {
	srcPort := int32(443)
	protocol := int32(6)
	req := &AggregationRequest{
		TaskName: "demo-task",
		SrcIP:    "10.0.0.1",
		SrcPort:  &srcPort,
		Protocol: &protocol,
	}

	whereClauses, args := appendAggregationFilters(nil, nil, req)

	wantClauses := []string{
		"TaskName = ?",
		"SrcIP = ?",
		"SrcPort = ?",
		"Protocol = ?",
	}
	if !reflect.DeepEqual(whereClauses, wantClauses) {
		t.Fatalf("appendAggregationFilters() clauses = %#v, want %#v", whereClauses, wantClauses)
	}

	wantArgs := []any{"demo-task", "10.0.0.1", int32(443), int32(6)}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("appendAggregationFilters() args = %#v, want %#v", args, wantArgs)
	}
}

func TestUint64ToInt64(t *testing.T) {
	got, err := uint64ToInt64(42, "demo")
	if err != nil {
		t.Fatalf("uint64ToInt64() unexpected error: %v", err)
	}
	if got != 42 {
		t.Fatalf("uint64ToInt64() = %d, want 42", got)
	}

	_, err = uint64ToInt64(math.MaxUint64, "demo")
	if err == nil {
		t.Fatal("uint64ToInt64() error = nil, want non-nil")
	}
}
