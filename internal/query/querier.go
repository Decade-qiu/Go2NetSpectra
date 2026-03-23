package query

import (
	"context"
	"crypto/tls"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"Go2NetSpectra/internal/config"

	"github.com/ClickHouse/clickhouse-go/v2"
)

// AggregationRequest defines the supported aggregate query filters.
type AggregationRequest struct {
	EndTime  *time.Time
	TaskName string
	SrcIP    string
	DstIP    string
	SrcPort  *int32
	DstPort  *int32
	Protocol *int32
}

// TaskSummary represents an aggregate summary row for a task.
type TaskSummary struct {
	TaskName     string
	TotalBytes   int64
	TotalPackets int64
	FlowCount    int64
}

// QueryTotalCountsResponse contains aggregate summaries.
type QueryTotalCountsResponse struct {
	Summaries []TaskSummary
}

// TraceFlowRequest defines the supported flow-trace filters.
type TraceFlowRequest struct {
	TaskName string
	FlowKeys map[string]string
	EndTime  *time.Time
}

// FlowLifecycle describes the observed lifecycle of a flow.
type FlowLifecycle struct {
	FirstSeen    time.Time
	LastSeen     time.Time
	TotalPackets int64
	TotalBytes   int64
}

// HeavyHittersRequest defines the supported heavy-hitter query filters.
type HeavyHittersRequest struct {
	TaskName string
	Type     int32
	EndTime  *time.Time
	Limit    int32
}

// HeavyHitter represents a single heavy-hitter result row.
type HeavyHitter struct {
	Flow  string
	Value int64
}

// HeavyHittersResponse contains heavy-hitter query results.
type HeavyHittersResponse struct {
	Hitters []HeavyHitter
}

// Querier defines the interface for querying flow data.
type Querier interface {
	AggregateFlows(ctx context.Context, req *AggregationRequest) (*QueryTotalCountsResponse, error)
	TraceFlow(ctx context.Context, req *TraceFlowRequest) (*FlowLifecycle, error)
	QueryHeavyHitters(ctx context.Context, req *HeavyHittersRequest) (*HeavyHittersResponse, error)
}

// clickhouseQuerier implements the Querier interface for ClickHouse.
type clickhouseQuerier struct {
	conn clickhouse.Conn
}

func uint64ToInt64(value uint64, field string) (int64, error) {
	if value > math.MaxInt64 {
		return 0, fmt.Errorf("%s exceeds int64 range: %d", field, value)
	}
	return int64(value), nil
}

var traceFlowKeys = map[string]struct{}{
	"DstIP":    {},
	"DstPort":  {},
	"Protocol": {},
	"SrcIP":    {},
	"SrcPort":  {},
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

	opts := &clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: cfg.Database,
			Username: cfg.Username,
			Password: cfg.Password,
		},
	}
	if cfg.Cloud {
		opts.Protocol = clickhouse.HTTP
		opts.TLS = &tls.Config{
			InsecureSkipVerify: true,
		}
	}
	conn, err := clickhouse.Open(opts)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping clickhouse: %w", err)
	}

	return conn, nil
}

func appendAggregationFilters(whereClauses []string, args []any, req *AggregationRequest) ([]string, []any) {
	if req.TaskName != "" {
		whereClauses = append(whereClauses, "TaskName = ?")
		args = append(args, req.TaskName)
	}
	if req.SrcIP != "" {
		whereClauses = append(whereClauses, "SrcIP = ?")
		args = append(args, req.SrcIP)
	}
	if req.DstIP != "" {
		whereClauses = append(whereClauses, "DstIP = ?")
		args = append(args, req.DstIP)
	}
	if req.SrcPort != nil {
		whereClauses = append(whereClauses, "SrcPort = ?")
		args = append(args, *req.SrcPort)
	}
	if req.DstPort != nil {
		whereClauses = append(whereClauses, "DstPort = ?")
		args = append(args, *req.DstPort)
	}
	if req.Protocol != nil {
		whereClauses = append(whereClauses, "Protocol = ?")
		args = append(args, *req.Protocol)
	}

	return whereClauses, args
}

func appendTraceFlowFilters(whereClauses []string, args []any, flowKeys map[string]string) ([]string, []any, error) {
	sortedKeys := make([]string, 0, len(flowKeys))
	for key := range flowKeys {
		if _, ok := traceFlowKeys[key]; !ok {
			return nil, nil, fmt.Errorf("unsupported flow key: %s", key)
		}
		sortedKeys = append(sortedKeys, key)
	}
	slices.Sort(sortedKeys)

	for _, key := range sortedKeys {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", key))
		args = append(args, flowKeys[key])
	}

	return whereClauses, args, nil
}

// QueryHeavyHitters builds and executes a dynamic heavy hitters query.
func (q *clickhouseQuerier) QueryHeavyHitters(ctx context.Context, req *HeavyHittersRequest) (*HeavyHittersResponse, error) {
	var queryBuilder strings.Builder
	queryBuilder.WriteString(`
		SELECT Flow, LatestValue
		FROM (
			SELECT
				Flow,
				argMax(Value, Timestamp) AS LatestValue
			FROM heavy_hitters
	`)

	whereClauses := make([]string, 0, 3)
	args := make([]any, 0, 4)

	whereClauses = append(whereClauses, "TaskName = ?")
	args = append(args, req.TaskName)

	whereClauses = append(whereClauses, "Type = ?")
	args = append(args, req.Type)

	if req.EndTime != nil {
		whereClauses = append(whereClauses, "Timestamp <= ?")
		args = append(args, *req.EndTime)
	}

	queryBuilder.WriteString(" WHERE " + strings.Join(whereClauses, " AND "))
	queryBuilder.WriteString(`
			GROUP BY Flow
		)
		ORDER BY LatestValue DESC
		LIMIT ?
	`)
	args = append(args, req.Limit)

	rows, err := q.conn.Query(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute heavy hitters query: %w", err)
	}
	defer rows.Close()

	var hitters []HeavyHitter
	for rows.Next() {
		var (
			hitter HeavyHitter
			value  uint64
		)
		if err := rows.Scan(&hitter.Flow, &value); err != nil {
			return nil, fmt.Errorf("failed to scan heavy hitter row: %w", err)
		}
		hitter.Value, err = uint64ToInt64(value, "heavy_hitter.value")
		if err != nil {
			return nil, err
		}
		hitters = append(hitters, hitter)
	}

	return &HeavyHittersResponse{Hitters: hitters}, nil
}

// AggregateFlows builds and executes a dynamic aggregation query.
func (q *clickhouseQuerier) AggregateFlows(ctx context.Context, req *AggregationRequest) (*QueryTotalCountsResponse, error) {
	var queryBuilder strings.Builder
	queryBuilder.WriteString(`
		SELECT
			TaskName,
			SUM(LatestByteCount) AS TotalBytes,
			SUM(LatestPacketCount) AS TotalPackets,
			COUNT(*) AS FlowCount
		FROM (
			SELECT
				TaskName,
				argMax(ByteCount, Timestamp) AS LatestByteCount,
				argMax(PacketCount, Timestamp) AS LatestPacketCount
			FROM flow_metrics
	`)

	whereClauses := make([]string, 0, 7)
	args := make([]any, 0, 7)

	whereClauses, args = appendAggregationFilters(whereClauses, args, req)
	if req.EndTime != nil {
		whereClauses = append(whereClauses, "Timestamp <= ?")
		args = append(args, *req.EndTime)
	}

	if len(whereClauses) > 0 {
		queryBuilder.WriteString(" WHERE " + strings.Join(whereClauses, " AND "))
	}

	queryBuilder.WriteString(`
			GROUP BY TaskName, SrcIP, DstIP, SrcPort, DstPort, Protocol
		)
		GROUP BY TaskName
	`)

	rows, err := q.conn.Query(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var summaries []TaskSummary
	for rows.Next() {
		var (
			summary      TaskSummary
			totalBytes   uint64
			totalPackets uint64
			flowCount    uint64
		)
		if err := rows.Scan(&summary.TaskName, &totalBytes, &totalPackets, &flowCount); err != nil {
			return nil, fmt.Errorf("failed to scan aggregation result: %w", err)
		}
		summary.TotalBytes, err = uint64ToInt64(totalBytes, "aggregation.total_bytes")
		if err != nil {
			return nil, err
		}
		summary.TotalPackets, err = uint64ToInt64(totalPackets, "aggregation.total_packets")
		if err != nil {
			return nil, err
		}
		summary.FlowCount, err = uint64ToInt64(flowCount, "aggregation.flow_count")
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}

	return &QueryTotalCountsResponse{Summaries: summaries}, nil
}

// TraceFlow executes a query to trace the lifecycle of a single flow.
func (q *clickhouseQuerier) TraceFlow(ctx context.Context, req *TraceFlowRequest) (*FlowLifecycle, error) {
	var queryBuilder strings.Builder
	queryBuilder.WriteString(`
		SELECT
			min(StartTime) AS FirstSeen,
			max(EndTime) AS LastSeen,
			max(PacketCount) AS TotalPackets,
			max(ByteCount) AS TotalBytes
		FROM flow_metrics
	`)

	whereClauses := make([]string, 0, len(req.FlowKeys)+2)
	args := make([]any, 0, len(req.FlowKeys)+2)

	whereClauses = append(whereClauses, "TaskName = ?")
	args = append(args, req.TaskName)

	whereClauses, args, err := appendTraceFlowFilters(whereClauses, args, req.FlowKeys)
	if err != nil {
		return nil, err
	}

	if req.EndTime != nil {
		whereClauses = append(whereClauses, "Timestamp <= ?")
		args = append(args, *req.EndTime)
	}

	if len(whereClauses) > 0 {
		queryBuilder.WriteString(" WHERE " + strings.Join(whereClauses, " AND "))
	}

	var (
		result       FlowLifecycle
		totalPackets uint64
		totalBytes   uint64
	)
	row := q.conn.QueryRow(ctx, queryBuilder.String(), args...)
	if err := row.Scan(&result.FirstSeen, &result.LastSeen, &totalPackets, &totalBytes); err != nil {
		return nil, fmt.Errorf("failed to scan flow lifecycle result: %w", err)
	}
	result.TotalPackets, err = uint64ToInt64(totalPackets, "trace.total_packets")
	if err != nil {
		return nil, err
	}
	result.TotalBytes, err = uint64ToInt64(totalBytes, "trace.total_bytes")
	if err != nil {
		return nil, err
	}

	return &result, nil
}
