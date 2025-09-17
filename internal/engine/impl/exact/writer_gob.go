package exact

import (
	"Go2NetSpectra/internal/engine/impl/exact/statistic"
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
	gob.Register(&statistic.Flow{})
}

// SummaryData holds the metadata for a snapshot, internal to the writer.
type SummaryData struct {
	TaskName     string `json:"task_name"`
	TotalFlows   int    `json:"total_flows"`
	TotalBytes   uint64 `json:"total_bytes"`
	TotalPackets uint64 `json:"total_packets"`
	Shards       int    `json:"shards"`
	Timestamp    string `json:"timestamp"`
}

// GobWriter handles writing aggregation task snapshot data to disk in gob format.
// It implements the model.Writer interface.
type GobWriter struct {
	rootPath string
	interval time.Duration
}

// NewGobWriter creates a new writer for aggregation task data.
func NewGobWriter(rootPath string, interval time.Duration) model.Writer {
	return &GobWriter{rootPath: rootPath, interval: interval}
}

// GetInterval returns the configured snapshot interval for this writer.
func (w *GobWriter) GetInterval() time.Duration {
	return w.interval
}

// Write serializes and writes the data from a single aggregation task snapshot to disk.
// It expects the payload to be of type exact.SnapshotData.
func (w *GobWriter) Write(payload interface{}, timestamp, name string, fields []string, decodeFlowFunc func(flow []byte, fields []string) string) error {
	snapshot, ok := payload.(statistic.SnapshotData)
	if !ok {
		return fmt.Errorf("invalid payload type for GobWriter: expected statistic.SnapshotData, got %T", payload)
	}

	// 1. Create timestamped directory
	snapshotDir := filepath.Join(w.rootPath, timestamp)
	// Let's make a subdirectory for the task to avoid file name collisions
	taskDir := filepath.Join(snapshotDir, snapshot.TaskName)
	if err := os.MkdirAll(taskDir, 0755); err != nil {
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
		filePath := filepath.Join(taskDir, fileName)

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
			TaskName:     snapshot.TaskName,
			TotalFlows:   totalFlows,
			TotalBytes:   totalBytes,
			TotalPackets: totalPackets,
			Shards:       len(snapshot.Shards),
			Timestamp:    time.Now().UTC().Format(time.RFC3339),
		}
		summaryFilePath := filepath.Join(taskDir, "summary.json")
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
