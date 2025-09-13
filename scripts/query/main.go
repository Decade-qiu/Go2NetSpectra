package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
)

// --- API Query Struct ---
type AggregationRequest struct {
	EndTime  string `json:"end_time,omitempty"`
	TaskName string `json:"task_name,omitempty"`
}

// --- Main Function ---
func main() {
	// Define command-line flags
	mode := flag.String("mode", "api", "Query mode: 'api' to query via HTTP API, 'direct' to query ClickHouse directly.")
	taskName := flag.String("task", "", "The name of the task to query (optional).")

	defaultEnd := time.Now().UTC().Add(8 * time.Hour).Format(time.RFC3339)
	endTimeStr := flag.String("end", defaultEnd, "End time in RFC3339 format (e.g., 2025-09-12T15:10:00Z).")

	flag.Parse()

	log.Printf("Running in '%s' mode.", *mode)

	switch *mode {
	case "api":
		queryViaAPI(*taskName, *endTimeStr)
	case "direct":
		directQueryClickHouse(*taskName, *endTimeStr)
	default:
		log.Fatalf("Invalid mode: %s. Use 'api' or 'direct'.", *mode)
	}
}

// --- API Query Logic ---
func queryViaAPI(taskName, endTime string) {
	apiURL := "http://localhost:8080/api/v1/aggregate"

	reqBody := AggregationRequest{
		EndTime:  endTime,
		TaskName: taskName,
	}

	jsonReqBody, err := json.Marshal(reqBody)
	if err != nil {
		log.Fatalf("Error marshalling request body: %v", err)
	}

	log.Printf("Sending request to %s with body:\n%s\n", apiURL, string(jsonReqBody))

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonReqBody))
	if err != nil {
		log.Fatalf("Error sending request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("API returned non-200 status code: %d\nResponse: %s", resp.StatusCode, string(respBody))
	}

	var prettyJSON bytes.Buffer
	err = json.Indent(&prettyJSON, respBody, "", "  ")
	if err != nil {
		log.Printf("Could not prettify JSON, printing raw response:")
		fmt.Println(string(respBody))
		return
	}

	log.Println("---")
	fmt.Println(prettyJSON.String())
}

// --- Direct ClickHouse Query Logic ---
func directQueryClickHouse(taskName, endTimeStr string) {
	connOpts := clickhouse.Options{
		Addr: []string{"localhost:19000"},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
			Password: "123",
		},
	}

	var queryBuilder strings.Builder
	queryBuilder.WriteString("\n\t\tSELECT\n\t\t\tTaskName,\n\t\t\tSUM(LatestByteCount) AS TotalBytes,\n\t\t\tSUM(LatestPacketCount) AS TotalPackets,\n\t\t\tCOUNT(*) AS FlowCount\n\t\tFROM (\n\t\t\tSELECT\n\t\t\t\tTaskName,\n\t\t\t\targMax(ByteCount, Timestamp) AS LatestByteCount,\n\t\t\t\targMax(PacketCount, Timestamp) AS LatestPacketCount\n\t\t\tFROM flow_metrics\n")

	var whereClauses []string
	args := []interface{}{}

	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		log.Fatalf("Invalid end time format: %v", err)
	}
	whereClauses = append(whereClauses, "Timestamp <= ?")
	args = append(args, endTime)

	if taskName != "" {
		whereClauses = append(whereClauses, "TaskName = ?")
		args = append(args, taskName)
	}

	queryBuilder.WriteString(" WHERE " + strings.Join(whereClauses, " AND "))

	queryBuilder.WriteString("\n\t\t\tGROUP BY TaskName, SrcIP, DstIP, SrcPort, DstPort, Protocol\n\t\t)\n\t\tGROUP BY TaskName\n")

	conn, err := clickhouse.Open(&connOpts)
	if err != nil {
		log.Fatalf("Error connecting to ClickHouse: %v", err)
	}
	defer conn.Close()

	log.Println("Successfully connected to ClickHouse.")
	
	rows, err := conn.Query(context.Background(), queryBuilder.String(), args...)
	if err != nil {
		log.Fatalf("Error executing query: %v", err)
	}
	defer rows.Close()

	log.Println("--- Aggregated Query Results (Direct) ---")

	var foundResult bool
	for rows.Next() {
		foundResult = true
		var (
			queriedTaskName string
			totalBytes      uint64
			totalPackets    uint64
			flowCount       uint64
		)

		if err := rows.Scan(&queriedTaskName, &totalBytes, &totalPackets, &flowCount); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		fmt.Printf("TaskName: %s\n", queriedTaskName)
		fmt.Printf("  TotalBytes: %d\n", totalBytes)
		fmt.Printf("  TotalPackets: %d\n", totalPackets)
		fmt.Printf("  FlowCount: %d\n", flowCount)
		fmt.Println("---------------------")
	}

	if !foundResult {
		log.Println("No data found for the specified criteria.")
	}

	if err := rows.Err(); err != nil {
		log.Printf("An error occurred during row iteration: %v", err)
	}
}
