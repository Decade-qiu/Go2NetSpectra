package exactaggregator

import (
	"Go2NetSpectra/internal/model"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func init() {
	// Register the concrete type of Flow for gob encoding/decoding.
	gob.Register(&Flow{})
}

// SummaryData holds the metadata for a snapshot, internal to the writer.
type SummaryData struct {
	AggregatorName string `json:"aggregator_name"`
	TotalFlows     int    `json:"total_flows"`
	TotalBytes     uint64 `json:"total_bytes"`
	TotalPackets   uint64 `json:"total_packets"`
	Shards         int    `json:"shards"`
	Timestamp      string `json:"timestamp"`
}

// ExactWriter handles writing exact aggregator snapshot data to disk.
// It implements the model.Writer interface.
type ExactWriter struct{}

// NewExactWriter creates a new writer for exact aggregation data.
func NewExactWriter() model.Writer {
	return &ExactWriter{}
}

// Write serializes and writes the data from a single aggregator snapshot to disk.
// It expects the payload to be of type exactaggregator.SnapshotData.
func (w *ExactWriter) Write(payload interface{}, rootPath string, timestamp string) error {
	snapshot, ok := payload.(SnapshotData)
	if !ok {
		return fmt.Errorf("invalid payload type for ExactWriter: expected SnapshotData, got %T", payload)
	}

	// 1. Create timestamped directory
	snapshotDir := filepath.Join(rootPath, timestamp)
	// Let's make a subdirectory for the aggregator to avoid file name collisions
	aggregatorDir := filepath.Join(snapshotDir, snapshot.AggregatorName)
	if err := os.MkdirAll(aggregatorDir, 0755); err != nil {
		return fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	totalFlows := 0
	totalPackets, totalBytes := uint64(0), uint64(0)
	// 2. Write each shard's map to a .dat file
	for i, shard := range snapshot.Shards {
		if len(shard.Flows) == 0 {
			continue
		}
		totalFlows += len(shard.Flows)
		for _, flow := range shard.Flows {
			totalPackets += flow.PacketCount
			totalBytes += flow.ByteCount
		}

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
			TotalBytes:     totalBytes,
			TotalPackets:   totalPackets,
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
