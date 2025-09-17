package sketch

import (
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/engine/impl/sketch/statistic"
	"Go2NetSpectra/internal/model"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

const createHeavyHittersTableStatement = `
CREATE TABLE IF NOT EXISTS heavy_hitters (
    Timestamp   DateTime,
    TaskName    String,
    Flow        String,
    Value       UInt64,
	Type		UInt8
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(Timestamp)
ORDER BY (TaskName, Timestamp);
`

// ClickHouseWriter implements the model.Writer interface for ClickHouse.
type ClickHouseWriter struct {
	conn     driver.Conn
	interval time.Duration
}

// NewClickHouseWriter creates a new ClickHouse writer for heavy hitters.
func NewClickHouseWriter(cfg config.ClickHouseConfig, interval time.Duration) (model.Writer, error) {
	conn, err := connect(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to clickhouse: %w", err)
	}

	if err := conn.Exec(context.Background(), createHeavyHittersTableStatement); err != nil {
		return nil, fmt.Errorf("failed to create heavy_hitters table: %w", err)
	}
	log.Println("Successfully connected to ClickHouse and ensured heavy_hitters table exists.")

	return &ClickHouseWriter{conn: conn, interval: interval}, nil
}

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
	})

	if err != nil {
		return nil, err
	}

	if err := conn.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping clickhouse: %w", err)
	}

	return conn, nil
}

func (w *ClickHouseWriter) Write(payload interface{}, timestamp, name string, fields []string, decodeFlowFunc func(flow []byte, fields []string) string) error {
	heavyHitters, ok := payload.(statistic.HeavyRecord)
	if !ok {
		return fmt.Errorf("invalid payload type for ClickHouse Writer: expected statistic.HeavyRecord, got %T", payload)
	}

	batch, err := w.conn.PrepareBatch(context.Background(), "INSERT INTO heavy_hitters")
	if err != nil {
		return fmt.Errorf("failed to prepare batch: %w", err)
	}

	snapshotTime, _ := time.Parse("2006-01-02_15-04-05", timestamp)

	total := 0

	// size
	for _, hitter := range heavyHitters.Size {
		flow := decodeFlowFunc(hitter.Flow, fields)
		err = batch.Append(snapshotTime, name, flow, hitter.Size, 1)
		if err != nil {
			return fmt.Errorf("failed to append heavy hitter to batch: %w", err)
		} else {
			total++
		}
	}

	// count
	for _, hitter := range heavyHitters.Count {
		flow := decodeFlowFunc(hitter.Flow, fields)
		err = batch.Append(snapshotTime, name, flow, hitter.Count, 0)
		if err != nil {
			return fmt.Errorf("failed to append heavy hitter to batch: %w", err)
		} else {
			total++
		}
	}

	if err := batch.Send(); err != nil {
		return fmt.Errorf("failed to send batch: %w", err)
	}

	log.Printf("Wrote %d heavy hitters to ClickHouse", total)
	return nil
}