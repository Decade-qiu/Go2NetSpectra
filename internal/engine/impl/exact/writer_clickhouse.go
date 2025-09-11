package exact

import (
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/engine/impl/exact/statistic"
	"Go2NetSpectra/internal/model"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

const createTableStatement = `
CREATE TABLE IF NOT EXISTS flow_metrics (
    Timestamp   DateTime,
    TaskName    String,
    SrcIP       Nullable(String),
    DstIP       Nullable(String),
    SrcPort     Nullable(UInt16),
    DstPort     Nullable(UInt16),
    Protocol    Nullable(UInt8),
    StartTime   DateTime,
    EndTime     DateTime,
    ByteCount   UInt64,
    PacketCount UInt64
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(Timestamp)
ORDER BY (TaskName, Timestamp);
`

// ClickHouseWriter implements the model.Writer interface for ClickHouse.
type ClickHouseWriter struct {
	conn     driver.Conn
	interval time.Duration
}

// NewClickHouseWriter creates a new ClickHouse writer.
func NewClickHouseWriter(cfg config.ClickHouseConfig, interval time.Duration) (model.Writer, error) {
	conn, err := connect(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to clickhouse: %w", err)
	}

	if err := conn.Exec(context.Background(), createTableStatement); err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}
	log.Println("Successfully connected to ClickHouse and ensured table exists.")

	return &ClickHouseWriter{conn: conn, interval: interval}, nil
}

// GetInterval returns the configured snapshot interval for this writer.
func (w *ClickHouseWriter) GetInterval() time.Duration {
	return w.interval
}

func connect(cfg config.ClickHouseConfig) (driver.Conn, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: cfg.Database,
			Username: cfg.Username,
			Password: cfg.Password,
		},
		Debug:       false,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
	})

	if err != nil {
		return nil, err
	}

	if err := conn.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping clickhouse: %w", err)
	}

	return conn, nil
}

// Write inserts flow data into the ClickHouse flow_metrics table.
func (w *ClickHouseWriter) Write(payload interface{}, timestamp string) error {
	snapshot, ok := payload.(statistic.SnapshotData)
	if !ok {
		return fmt.Errorf("invalid payload type for ClickHouse Writer: expected statistic.SnapshotData, got %T", payload)
	}

	batch, err := w.conn.PrepareBatch(context.Background(), "INSERT INTO flow_metrics")
	if err != nil {
		return fmt.Errorf("failed to prepare batch: %w", err)
	}

	snapshotTime, _ := time.Parse("2006-01-02_15-04-05", timestamp)
	flowCount := 0

	for _, shard := range snapshot.Shards {
		for _, flow := range shard.Flows {
			flowCount++
			err = batch.Append(
				snapshotTime,
				snapshot.TaskName,
				getNullableField(flow.Fields, "SrcIP"),
				getNullableField(flow.Fields, "DstIP"),
				getNullableField(flow.Fields, "SrcPort"),
				getNullableField(flow.Fields, "DstPort"),
				getNullableField(flow.Fields, "Protocol"),
				flow.StartTime,
				flow.EndTime,
				flow.ByteCount,
				flow.PacketCount,
			)
			if err != nil {
				return fmt.Errorf("failed to append flow to batch: %w", err)
			}
		}
	}

	if flowCount == 0 {
		return nil // Nothing to write
	}

	if err := batch.Send(); err != nil {
		return fmt.Errorf("failed to send batch: %w", err)
	}

	log.Printf("Wrote %d flows to ClickHouse for task '%s'", flowCount, snapshot.TaskName)
	return nil
}

// getNullableField safely gets a value from the map for insertion.
func getNullableField(fields map[string]interface{}, key string) interface{} {
	if val, ok := fields[key]; ok {
		return val
	}
	return nil
}
