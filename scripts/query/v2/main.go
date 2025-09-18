package main

import (
	v1 "Go2NetSpectra/api/gen/v1"
	"context"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Command-line flags
	serverAddr := flag.String("addr", "localhost:50051", "The gRPC server address")
	mode := flag.String("mode", "aggregate", "Query mode: 'aggregate' or 'trace'")
	taskName := flag.String("task", "", "The name of the task to query (required for both modes)")
	flowKey := flag.String("key", "", "The flow key for trace mode (e.g., \"SrcIP=1.2.3.4,DstPort=443\")")
	flag.Parse()

	// Set up a connection to the server.
	conn, err := grpc.NewClient(*serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := v1.NewQueryServiceClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	switch *mode {
	case "aggregate":
		doAggregateQuery(ctx, client, *taskName)
	case "trace":
		if *flowKey == "" {
			log.Fatal("Error: -key flag is required for trace mode")
		}
		doTraceQuery(ctx, client, *taskName, *flowKey)
	default:
		log.Fatalf("Unknown mode: %s. Use 'aggregate' or 'trace'", *mode)
	}
}

func doAggregateQuery(ctx context.Context, client v1.QueryServiceClient, taskName string) {
	log.Printf("Executing aggregation query for task: %s", taskName)

	req := &v1.AggregationRequest{
		TaskName: taskName,
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

func doTraceQuery(ctx context.Context, client v1.QueryServiceClient, taskName, flowKeyStr string) {
	log.Printf("Executing trace query for task '%s' with key '%s'", taskName, flowKeyStr)

	flowKeys, err := parseFlowKeys(flowKeyStr)
	if err != nil {
		log.Fatalf("Invalid flow key format: %v", err)
	}

	req := &v1.TraceFlowRequest{
		TaskName: taskName,
		FlowKeys: flowKeys,
	}

	_resp, err := client.TraceFlow(ctx, req)
	if err != nil {
		log.Fatalf("could not perform trace query: %v", err)
	}

	resp := _resp.Lifecycle

	log.Println("---", "Flow Lifecycle Result", "---")
	log.Printf("  First Seen:    %s", resp.FirstSeen.AsTime().Format(time.RFC3339))
	log.Printf("  Last Seen:     %s", resp.LastSeen.AsTime().Format(time.RFC3339))
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