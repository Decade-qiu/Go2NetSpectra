package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	v1 "Go2NetSpectra/api/gen/thrift/v1"

	thrift "github.com/apache/thrift/lib/go/thrift"
)

const queryClientTransportBufferSize = 32 * 1024

func main() {
	// Command-line flags
	serverAddr := flag.String("addr", "localhost:50051", "The gRPC server address")
	mode := flag.String("mode", "heavyhitters", "Query mode: 'aggregate', 'trace', 'heavyhitters', or 'superspreader'")
	taskName := flag.String("task", "", "The name of the task to query")
	flowKey := flag.String("key", "", "The flow key for trace mode (e.g., \"SrcIP=1.2.3.4,DstPort=443\")")
	hhType := flag.Int("type", 0, "Query type for heavyhitters (0 for count, 1 for size)")
	limit := flag.Int("limit", 10, "Limit for heavy hitters/super spreader query")
	defaultEnd := time.Now().UTC().Add(8 * time.Hour).Format(time.RFC3339)
	endTimeStr := flag.String("end", defaultEnd, "End time in RFC3339 format (e.g., 2025-09-12T15:10:00Z).")

	flag.Parse()

	if *taskName == "" && *mode != "aggregate" {
		log.Fatal("error: -task flag is required for this mode")
	}

	// Set up a connection to the RPC server.
	client, transport, err := newQueryClient(*serverAddr)
	if err != nil {
		log.Fatalf("failed to connect to %s: %v", *serverAddr, err)
	}
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	switch *mode {
	case "aggregate":
		doAggregateQuery(ctx, client, *taskName, *endTimeStr)
	case "trace":
		if *flowKey == "" {
			log.Fatal("error: -key flag is required for trace mode")
		}
		doTraceQuery(ctx, client, *taskName, *flowKey, *endTimeStr)
	case "heavyhitters":
		doHeavyHittersQuery(ctx, client, *taskName, *hhType, *limit, *endTimeStr)
	case "superspreader":
		doSuperSpreaderQuery(ctx, client, *taskName, *limit, *endTimeStr)
	default:
		log.Fatalf("unknown mode %q; use 'aggregate', 'trace', 'heavyhitters', or 'superspreader'", *mode)
	}
}

// doAggregateQuery performs an aggregation query.
func doAggregateQuery(ctx context.Context, client *v1.QueryServiceClient, taskName string, endTime string) {
	log.Printf("Executing aggregation query for task: %s", taskName)
	log.Printf("Query params - End time: %s", endTime)

	req := &v1.AggregationRequest{
		EndTimeUnixNano: parseAndConvert(endTime),
	}
	if taskName != "" {
		req.TaskName = &taskName
	}

	resp, err := client.AggregateFlows(ctx, req)
	if err != nil {
		log.Fatalf("could not perform aggregation query: %v", err)
	}

	log.Println("---", "Aggregation Results", "---")
	if len(resp.Summaries) == 0 {
		log.Println("No data returned.")
		return
	}
	for _, summary := range resp.Summaries {
		log.Printf("  Task: %s", summary.TaskName)
		log.Printf("    Total Flows:   %d", summary.FlowCount)
		log.Printf("    Total Packets: %d", summary.TotalPackets)
		log.Printf("    Total Bytes:   %d", summary.TotalBytes)
	}
	log.Println("---------------------------")
}

// doTraceQuery performs a trace query.
func doTraceQuery(ctx context.Context, client *v1.QueryServiceClient, taskName, flowKeyStr string, endTime string) {
	log.Printf("Executing trace query for task '%s' with key '%s'", taskName, flowKeyStr)
	log.Printf("Query params - End time: %s", endTime)

	flowKeys, err := parseFlowKeys(flowKeyStr)
	if err != nil {
		log.Fatalf("invalid flow key format: %v", err)
	}

	req := &v1.TraceFlowRequest{
		TaskName:        taskName,
		FlowKeys:        flowKeys,
		EndTimeUnixNano: parseAndConvert(endTime),
	}

	_resp, err := client.TraceFlow(ctx, req)
	if err != nil {
		log.Fatalf("could not perform trace query: %v", err)
	}

	resp := _resp.Lifecycle

	log.Println("---", "Flow Lifecycle Result", "---")
	log.Printf("  First Seen:    %s", time.Unix(0, resp.FirstSeenUnixNano).Format(time.RFC3339))
	log.Printf("  Last Seen:     %s", time.Unix(0, resp.LastSeenUnixNano).Format(time.RFC3339))
	log.Printf("  Total Packets: %d", resp.TotalPackets)
	log.Printf("  Total Bytes:   %d", resp.TotalBytes)
	log.Println("-----------------------------")
}

// parseFlowKeys converts a string like "SrcIP=1.2.3.4,DstPort=80" into a map.
func parseFlowKeys(keyStr string) (map[string]string, error) {
	if keyStr == "" {
		return nil, fmt.Errorf("key string cannot be empty")
	}
	keys := make(map[string]string)
	pairs := strings.Split(keyStr, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid key-value pair: %s", pair)
		}
		keys[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
	}
	return keys, nil
}

// doHeavyHittersQuery performs a heavy hitters query.
func doHeavyHittersQuery(ctx context.Context, client *v1.QueryServiceClient, taskName string, hhType int, limit int, endTime string) {
	log.Printf("Executing heavy hitters query for task: %s", taskName)
	log.Printf("Heavy hitter type: %d, Limit: %d", hhType, limit)
	log.Printf("Query params - End time: %s", endTime)

	req := &v1.HeavyHittersRequest{
		TaskName:        taskName,
		Type:            int32(hhType),
		EndTimeUnixNano: parseAndConvert(endTime),
		Limit:           int32(limit),
	}

	resp, err := client.QueryHeavyHitters(ctx, req)
	if err != nil {
		log.Fatalf("could not perform heavy hitters query: %v", err)
	}

	log.Printf("--- Heavy Hitters Results ---")
	if len(resp.Hitters) == 0 {
		log.Println("No data returned.")
		return
	}
	log.Printf("% -4s | % -40s | %s", "Rank", "Flow", "Value")
	log.Println(strings.Repeat("-", 60))
	for i, hitter := range resp.Hitters {
		log.Printf("% -4d | % -40s | %d", i+1, hitter.Flow, hitter.Value)
	}
	log.Println("-----------------------------")
}

// doSuperSpreaderQuery performs a super spreader query.
func doSuperSpreaderQuery(ctx context.Context, client *v1.QueryServiceClient, taskName string, limit int, endTime string) {
	log.Printf("Executing super spreader query for task: %s", taskName)
	log.Printf("Limit: %d", limit)
	log.Printf("Query params - End time: %s", endTime)

	req := &v1.HeavyHittersRequest{
		TaskName:        taskName,
		Type:            2, // Type 2 is for SuperSpreaders
		EndTimeUnixNano: parseAndConvert(endTime),
		Limit:           int32(limit),
	}

	resp, err := client.QueryHeavyHitters(ctx, req)
	if err != nil {
		log.Fatalf("could not perform super spreader query: %v", err)
	}

	log.Printf("--- Super Spreader Results ---")
	if len(resp.Hitters) == 0 {
		log.Println("No data returned.")
		return
	}
	log.Printf("% -4s | % -40s | %s", "Rank", "Flow", "Spread (Cardinality)")
	log.Println(strings.Repeat("-", 60))
	for i, hitter := range resp.Hitters {
		log.Printf("% -4d | % -40s | %d", i+1, hitter.Flow, hitter.Value)
	}
	log.Println("------------------------------")
}

func parseAndConvert(endTimeStr string) *int64 {
	t, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		log.Fatalf("failed to parse time string: %v", err)
		return nil
	}
	unixNano := t.UnixNano()
	return &unixNano
}

func newQueryClient(addr string) (*v1.QueryServiceClient, thrift.TTransport, error) {
	conf := &thrift.TConfiguration{
		ConnectTimeout: 5 * time.Second,
		SocketTimeout:  15 * time.Second,
	}
	socket := thrift.NewTSocketConf(addr, conf)
	transportFactory := thrift.NewTBufferedTransportFactory(queryClientTransportBufferSize)
	transport, err := transportFactory.GetTransport(socket)
	if err != nil {
		return nil, nil, fmt.Errorf("build thrift transport: %w", err)
	}
	if err := transport.Open(); err != nil {
		return nil, nil, fmt.Errorf("open thrift transport: %w", err)
	}

	protocolFactory := thrift.NewTBinaryProtocolFactoryConf(conf)
	return v1.NewQueryServiceClientFactory(transport, protocolFactory), transport, nil
}
