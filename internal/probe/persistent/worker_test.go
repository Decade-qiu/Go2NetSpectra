package persistent

import (
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/model"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWorkerStopFlushesQueuedPackets(t *testing.T) {
	dir := t.TempDir()
	worker, err := NewWorker(config.PersistenceConfig{
		Path:              dir,
		Encoding:          "text",
		NumWorkers:        1,
		ChannelBufferSize: 1,
	})
	if err != nil {
		t.Fatalf("NewWorker() unexpected error: %v", err)
	}

	worker.Enqueue(&PacketContainer{
		PacketInfo: &model.PacketInfo{
			Timestamp: time.Unix(1700000002, 0),
			Length:    512,
			FiveTuple: model.FiveTuple{
				SrcPort:  12345,
				DstPort:  443,
				Protocol: 6,
			},
		},
	})

	worker.Stop()

	var (
		content string
		found   bool
	)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		entries, readErr := os.ReadDir(dir)
		if readErr != nil {
			t.Fatalf("ReadDir(%s) error: %v", dir, readErr)
		}
		if len(entries) == 0 {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(dir, entries[0].Name()))
		if readErr != nil {
			t.Fatalf("ReadFile(%s) error: %v", entries[0].Name(), readErr)
		}
		content = string(data)
		if strings.Contains(content, "12345") && strings.Contains(content, "443") {
			found = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if !found {
		t.Fatalf("persisted content missing expected ports, got %q", content)
	}
}
