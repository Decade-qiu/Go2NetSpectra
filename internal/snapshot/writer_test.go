package snapshot

import (
	"Go2NetSpectra/internal/core/model"
	"Go2NetSpectra/internal/engine/flowaggregator"
	"encoding/gob"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriter_WriteSnapshot(t *testing.T) {
	// 1. Create sample snapshot data
	testFlows := make(map[string]*model.Flow)
	testFlows["test-key"] = &model.Flow{Key: "test-key", PacketCount: 1, ByteCount: 100}

	snapshotData := flowaggregator.SnapshotData{
		AggregatorName: "test_aggregator",
		Shards: []*flowaggregator.Shard{
			{
				Flows: testFlows,
			},
			{
				Flows: make(map[string]*model.Flow), // An empty shard
			},
		},
	}

	// 2. Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "snapshot_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 3. Write the snapshot
	writer := NewWriter()
	err = writer.WriteSnapshot(snapshotData, tmpDir)
	if err != nil {
		t.Fatalf("WriteSnapshot failed: %v", err)
	}

	// 4. Verify directory and files
	// The directory name is based on the current time, so we need to find it.
	dirs, err := os.ReadDir(tmpDir)
	if err != nil || len(dirs) != 1 || !dirs[0].IsDir() {
		t.Fatalf("Expected one timestamped directory in temp dir, found %d", len(dirs))
	}
	timestampDir := filepath.Join(tmpDir, dirs[0].Name())
    aggregatorDir := filepath.Join(timestampDir, "test_aggregator")


	// Check for summary.json
	summaryPath := filepath.Join(aggregatorDir, "summary.json")
	if _, err := os.Stat(summaryPath); os.IsNotExist(err) {
		t.Fatalf("summary.json was not created")
	}

	// Check for shard data file
	shardPath := filepath.Join(aggregatorDir, "shard_0.dat")
	if _, err := os.Stat(shardPath); os.IsNotExist(err) {
		t.Fatalf("shard_0.dat was not created")
	}

	// Check that empty shard was not written
	emptyShardPath := filepath.Join(aggregatorDir, "shard_1.dat")
	if _, err := os.Stat(emptyShardPath); !os.IsNotExist(err) {
		t.Fatalf("shard_1.dat (empty) should not have been created")
	}


	// 5. Verify summary content
	summaryBytes, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("Failed to read summary.json: %v", err)
	}
	var summary SummaryData
	if err := json.Unmarshal(summaryBytes, &summary); err != nil {
		t.Fatalf("Failed to unmarshal summary.json: %v", err)
	}
	if summary.TotalFlows != 1 {
		t.Errorf("Expected TotalFlows to be 1, got %d", summary.TotalFlows)
	}
	if summary.AggregatorName != "test_aggregator" {
		t.Errorf("Expected AggregatorName to be 'test_aggregator', got '%s'", summary.AggregatorName)
	}


	// 6. Verify gob file content
	gobFile, err := os.Open(shardPath)
	if err != nil {
		t.Fatalf("Failed to open shard_0.dat: %v", err)
	}
	defer gobFile.Close()

	var decodedFlows map[string]*model.Flow
	decoder := gob.NewDecoder(gobFile)
	if err := decoder.Decode(&decodedFlows); err != nil {
		t.Fatalf("Failed to decode gob file: %v", err)
	}

	if len(decodedFlows) != 1 {
		t.Fatalf("Expected 1 flow in decoded map, got %d", len(decodedFlows))
	}
	if flow, ok := decodedFlows["test-key"]; !ok || flow.PacketCount != 1 || flow.ByteCount != 100 {
		t.Errorf("Decoded flow content does not match expected content. Got: %+v", flow)
	}
}
