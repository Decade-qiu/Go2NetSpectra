package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// --- API Query Structs ---
type AggregationRequest struct {
	EndTime  string `json:"end_time,omitempty"`
	TaskName string `json:"task_name,omitempty"`
}

type TraceFlowRequest struct {
	TaskName string            `json:"task_name"`
	FlowKeys map[string]string `json:"flow_keys"`
	EndTime  string            `json:"end_time,omitempty"`
}

// --- Main Function ---
func main() {
	// Define command-line flags
	mode := flag.String("mode", "aggregate", "Query mode: 'aggregate' for totals, 'trace' for a single flow lifecycle.")
	taskName := flag.String("task", "", "The name of the task to query.")
	flowKey := flag.String("key", "SrcIP=127.0.0.1", "Flow key for trace mode (e.g., SrcIP=1.2.3.4,DstPort=443).")

	defaultEnd := time.Now().UTC().Add(8 * time.Hour).Format(time.RFC3339)
	endTimeStr := flag.String("end", defaultEnd, "End time in RFC3339 format (e.g., 2025-09-12T15:10:00Z).")

	flag.Parse()

	log.Printf("Running in '%s' mode.", *mode)

	switch *mode {
	case "aggregate":
		queryAggregation(*taskName, *endTimeStr)
	case "trace":
		traceFlow(*taskName, *flowKey, *endTimeStr)
	default:
		log.Fatalf("Invalid mode: %s. Use 'aggregate' or 'trace'.", *mode)
	}
}

// --- API Query Logic ---
func queryAggregation(taskName, endTime string) {
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

func traceFlow(taskName, flowKeyStr, endTime string) {
	apiURL := "http://localhost:8080/api/v1/flows/trace"

	flowKeys := make(map[string]string)
	pairs := strings.Split(flowKeyStr, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			flowKeys[kv[0]] = kv[1]
		}
	}

	reqBody := TraceFlowRequest{
		TaskName: taskName,
		FlowKeys: flowKeys,
		EndTime:  endTime,
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
