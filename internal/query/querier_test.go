package query

import (
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
