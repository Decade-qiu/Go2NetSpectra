package flowaggregator

import (
	"Go2NetSpectra/internal/core/model"
	"Go2NetSpectra/internal/pkg/config"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFlowAggregator_EndToEndSnapshot(t *testing.T) {
	// 1. Create a temporary directory for snapshots
	tmpDir, err := os.MkdirTemp("", "aggregator_e2e_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 2. Load and modify config
	cfg, err := config.LoadConfig("../../../configs/config.yaml")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	cfg.Aggregator.SnapshotInterval = "50ms"
	cfg.Aggregator.StorageRootPath = tmpDir

	// 3. Create and start aggregator
	aggregator, err := NewFlowAggregator(cfg)
	if err != nil {
		t.Fatalf("Failed to create aggregator: %v", err)
	}
	aggregator.Start()

	// 4. Send a packet
	packet := &model.PacketInfo{
		Timestamp: time.Now(),
		FiveTuple: model.FiveTuple{
			SrcIP:    net.ParseIP("192.168.0.1"),
			DstIP:    net.ParseIP("8.8.8.8"),
			SrcPort:  12345,
			DstPort:  53,
			Protocol: 17, // UDP
		},
		Length: 100,
	}
	aggregator.InputChannel <- packet

	// 5. Wait for snapshot to be written
	time.Sleep(100 * time.Millisecond) // Wait for at least one snapshot cycle

	// 6. Stop the aggregator to trigger final snapshot
	aggregator.Stop()

	// 7. Verify that snapshot files were created
	// The exact timestamped directory name is unknown, so we find it.
	dirs, err := os.ReadDir(tmpDir)
	if err != nil || len(dirs) == 0 {
		t.Fatalf("Snapshot directory was not created in temp dir")
	}
    // There might be two directories if the clock ticked over during the test.
    // We just need to find at least one valid snapshot.
    foundSnapshots := false
    for _, dir := range dirs {
        if !dir.IsDir() {
            continue
        }
        timestampDir := filepath.Join(tmpDir, dir.Name())
        // Check for one of the aggregator outputs
        aggDir := filepath.Join(timestampDir, "by_src_ip")
        summaryPath := filepath.Join(aggDir, "summary.json")
        if _, err := os.Stat(summaryPath); err == nil {
            foundSnapshots = true
            break
        }
    }

    if !foundSnapshots {
        t.Fatalf("Could not find any valid snapshot files in output directory %s", tmpDir)
    }
}