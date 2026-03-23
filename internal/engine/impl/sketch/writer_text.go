package sketch

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"Go2NetSpectra/internal/engine/impl/sketch/statistic"
	"Go2NetSpectra/internal/model"
)

// TextWriter handles writing heavy hitters to a text file.
type TextWriter struct {
	rootPath string
	interval time.Duration
}

// NewTextWriter creates a new text writer for heavy hitters.
func NewTextWriter(rootPath string, interval time.Duration) model.Writer {
	return &TextWriter{rootPath: rootPath, interval: interval}
}

// Interval returns the configured snapshot interval for this writer.
func (w *TextWriter) Interval() time.Duration {
	return w.interval
}

// Write writes heavy hitter results to text files under the configured snapshot directory.
func (w *TextWriter) Write(payload interface{}, timestamp, name string, fields []string, decodeFlowFunc func(flow []byte, fields []string) string) error {
	heavyHitters, ok := payload.(statistic.HeavyRecord)
	if !ok {
		return fmt.Errorf("invalid payload type for text writer: expected statistic.HeavyRecord, got %T", payload)
	}

	snapshotDir := filepath.Join(w.rootPath, timestamp)
	taskDir := filepath.Join(snapshotDir, name)
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	total := 0

	if heavyHitters.Size != nil {
		// size
		filePath := filepath.Join(taskDir, "size_hh.txt")
		file, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("failed to create snapshot file '%s': %w", filePath, err)
		}
		defer file.Close()

		for _, hitter := range heavyHitters.Size {
			line := fmt.Sprintf("%s %d\n", decodeFlowFunc(hitter.Flow, fields), hitter.Size)
			if _, err := file.WriteString(line); err != nil {
				return fmt.Errorf("failed to write heavy hitter to file: %w", err)
			}
			total++
		}

		// count
		filePath = filepath.Join(taskDir, "count_hh.txt")
		file, err = os.Create(filePath)
		if err != nil {
			return fmt.Errorf("failed to create snapshot file '%s': %w", filePath, err)
		}
		defer file.Close()

		for _, hitter := range heavyHitters.Count {
			line := fmt.Sprintf("%s %d\n", decodeFlowFunc(hitter.Flow, fields), hitter.Count)
			if _, err := file.WriteString(line); err != nil {
				return fmt.Errorf("failed to write heavy hitter to file: %w", err)
			}
			total++
		}
	} else {
		// count
		filePath := filepath.Join(taskDir, "spread_ss.txt")
		file, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("failed to create snapshot file '%s': %w", filePath, err)
		}
		defer file.Close()

		for _, hitter := range heavyHitters.Count {
			line := fmt.Sprintf("%s %d\n", decodeFlowFunc(hitter.Flow, fields), hitter.Count)
			if _, err := file.WriteString(line); err != nil {
				return fmt.Errorf("failed to write heavy hitter to file: %w", err)
			}
			total++
		}
	}

	log.Printf("Successfully wrote %d heavy hitters to %s\n", total, taskDir)

	return nil
}
