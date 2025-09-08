package snapshot

import (
	"Go2NetSpectra/internal/core/model"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func init() {
	gob.Register(&model.Flow{})
}

// SummaryData holds the metadata for a snapshot.
type SummaryData struct {
	AggregatorName string `json:"aggregator_name"`
	TotalFlows     int    `json:"total_flows"`
	Shards         int    `json:"shards"`
	Timestamp      string `json:"timestamp"`
}

// Writer handles writing snapshot data to disk.
type Writer struct{}

// NewWriter creates a new snapshot writer.
func NewWriter() *Writer {
	return &Writer{}
}

// Write serializes and writes the data from a single aggregator snapshot to disk.
// It creates a timestamped directory for the snapshot.
func (w *Writer) Write(snapshot model.SnapshotData, rootPath string, timestamp string) error {
	// 1. Create timestamped directory
	snapshotDir := filepath.Join(rootPath, timestamp)
	// Let's make a subdirectory for the aggregator to avoid file name collisions
	aggregatorDir := filepath.Join(snapshotDir, snapshot.AggregatorName)
	if err := os.MkdirAll(aggregatorDir, 0755); err != nil {
		return fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	totalFlows := 0
	// 2. Write each shard's map to a .dat file
	for i, shard := range snapshot.Shards {
		if len(shard.Flows) == 0 {
			continue
		}
		totalFlows += len(shard.Flows)

		fileName := fmt.Sprintf("shard_%d.dat", i)
		filePath := filepath.Join(aggregatorDir, fileName)

		file, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("failed to create snapshot file '%s': %w", filePath, err)
		}
		defer file.Close()

		encoder := gob.NewEncoder(file)
		if err := encoder.Encode(shard.Flows); err != nil {
			return fmt.Errorf("failed to encode flows to gob for file '%s': %w", filePath, err)
		}
	}

	// 3. Write summary file if there were any flows
	if totalFlows > 0 {
		summary := SummaryData{
			AggregatorName: snapshot.AggregatorName,
			TotalFlows:     totalFlows,
			Shards:         len(snapshot.Shards),
			Timestamp:      time.Now().UTC().Format(time.RFC3339),
		}
		summaryFilePath := filepath.Join(aggregatorDir, "summary.json")
		summaryFile, err := os.Create(summaryFilePath)
		if err != nil {
			return fmt.Errorf("failed to create summary file: %w", err)
		}
		defer summaryFile.Close()

		jsonEncoder := json.NewEncoder(summaryFile)
		jsonEncoder.SetIndent("", "  ")
		if err := jsonEncoder.Encode(summary); err != nil {
			return fmt.Errorf("failed to encode summary to json: %w", err)
		}
	}

	return nil
}