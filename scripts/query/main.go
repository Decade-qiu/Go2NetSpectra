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

	"github.com/ClickHouse/clickhouse-go/v2"
)

// --- Main Function ---
func main() {
	// Define command-line flags
	mode := flag.String("mode", "api", "Query mode: 'api' to query via HTTP API, 'direct' to query ClickHouse directly.")
	flag.Parse()

	log.Printf("Running in '%s' mode.", *mode)

	switch *mode {
	case "api":
		queryViaAPI()
	case "direct":
		directQueryClickHouse()
	default:
		log.Fatalf("Invalid mode: %s. Use 'api' or 'direct'.", *mode)
	}
}

// --- API Query Logic ---
func queryViaAPI() {
	apiURL := "http://localhost:8080/api/v1/aggregate"

	log.Printf("Sending request to %s to get total counts for all tasks.", apiURL)

	// Send an empty JSON object as the request body
	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer([]byte("{}")))
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

	log.Println("--- Aggregated Query Results (via API) ---")
	fmt.Println(prettyJSON.String())
}

// --- Direct ClickHouse Query Logic ---
func directQueryClickHouse() {
	connOpts := clickhouse.Options{
		Addr: []string{"localhost:19000"},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
			Password: "123",
		},
	}

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

	conn, err := clickhouse.Open(&connOpts)
	if err != nil {
		log.Fatalf("Error connecting to ClickHouse: %v", err)
	}
	defer conn.Close()

	log.Println("Successfully connected to ClickHouse.")

	rows, err := conn.Query(context.Background(), query)
	if err != nil {
		log.Fatalf("Error executing query: %v", err)
	}
	defer rows.Close()

	log.Println("Successfully executed query for all tasks.")
	log.Println("--- Aggregated Query Results (Direct) ---")

	var foundResult bool
	for rows.Next() {
		foundResult = true
		var (
			queriedTaskName string
			totalBytes      uint64
			totalPackets    uint64
		)

		if err := rows.Scan(&queriedTaskName, &totalBytes, &totalPackets); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		fmt.Printf("TaskName: %s\n", queriedTaskName)
		fmt.Printf("  TotalBytes: %d\n", totalBytes)
		fmt.Printf("  TotalPackets: %d\n", totalPackets)
		fmt.Println("---------------------")
	}

	if !foundResult {
		log.Println("No data found in the database.")
	}

	if err := rows.Err(); err != nil {
		log.Printf("An error occurred during row iteration: %v", err)
	}
}