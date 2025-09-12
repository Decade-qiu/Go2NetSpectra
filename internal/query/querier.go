package query

import (
	v1 "Go2NetSpectra/api/gen/v1"
	"Go2NetSpectra/internal/config"
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
)

// Querier defines the interface for querying flow data.
type Querier interface {
	AggregateFlows(ctx context.Context, req *v1.AggregationRequest) (*v1.QueryTotalCountsResponse, error)
}

// clickhouseQuerier implements the Querier interface for ClickHouse.
type clickhouseQuerier struct {
	conn clickhouse.Conn
}

// NewClickHouseQuerier creates a new querier for ClickHouse.
func NewClickHouseQuerier(cfg config.ClickHouseConfig) (Querier, error) {
	conn, err := connect(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to clickhouse: %w", err)
	}
	return &clickhouseQuerier{conn: conn}, nil
}

func connect(cfg config.ClickHouseConfig) (clickhouse.Conn, error) {
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

// AggregateFlows executes a fixed query to get the total counts for each task.
func (q *clickhouseQuerier) AggregateFlows(ctx context.Context, req *v1.AggregationRequest) (*v1.QueryTotalCountsResponse, error) {
	query := `
		SELECT
			TaskName,
			SUM(LatestByteCount) AS TotalBytes,
			SUM(LatestPacketCount) AS TotalPackets
		FROM (
			SELECT
				TaskName,
				argMax(ByteCount, Timestamp) AS LatestByteCount,
				argMax(PacketCount, Timestamp) AS LatestPacketCount
			FROM flow_metrics
			GROUP BY TaskName, SrcIP, DstIP, SrcPort, DstPort, Protocol
		)
		GROUP BY TaskName
	`

	rows, err := q.conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var summaries []*v1.TaskSummary
	for rows.Next() {
		var summary v1.TaskSummary
		if err := rows.Scan(&summary.TaskName, &summary.TotalBytes, &summary.TotalPackets); err != nil {
			return nil, fmt.Errorf("failed to scan aggregation result: %w", err)
		}
		summaries = append(summaries, &summary)
	}

	return &v1.QueryTotalCountsResponse{Summaries: summaries}, nil
}
