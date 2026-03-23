package factory_test

import (
	"Go2NetSpectra/internal/config"
	_ "Go2NetSpectra/internal/engine/manager"
	"Go2NetSpectra/internal/factory"
	"testing"
)

func TestCreateUsesRegisteredAggregators(t *testing.T) {
	cfg := &config.Config{
		Aggregator: config.AggregatorConfig{
			Types: []string{"exact", "sketch"},
			Exact: config.ExactAggregatorConfig{
				Tasks: []config.ExactTaskDef{
					{Name: "per_five_tuple", KeyFields: []string{"SrcIP"}, NumShards: 8},
				},
			},
			Sketch: config.SketchAggregatorConfig{
				Tasks: []config.SketchTaskDef{
					{
						Name:            "cm_src_left",
						SktType:         0,
						FlowFields:      []string{"SrcIP"},
						ElementFields:   []string{"DstIP"},
						Width:           8,
						Depth:           2,
						SizeThereshold:  16,
						CountThereshold: 2,
					},
				},
			},
		},
	}

	taskGroups, err := factory.Create(cfg)
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}

	if got := len(taskGroups); got != 2 {
		t.Fatalf("len(taskGroups) = %d, want 2", got)
	}
}
