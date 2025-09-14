package main

import (
	v1 "Go2NetSpectra/api/gen/v1"
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/query"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Grafana-specific structs
type QueryRequest struct {
	Targets []struct {
		Target string `json:"target"`
	} `json:"targets"`
	Range struct {
		From time.Time `json:"from"`
		To   time.Time `json:"to"`
	} `json:"range"`
}

type TimeSeriesResponse struct {
	Target     string      `json:"target"`
	Datapoints [][]float64 `json:"datapoints"` // [ [value, timestamp_ms], ... ]
}

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	var chCfg *config.ClickHouseConfig
	for _, writerDef := range cfg.Aggregator.Exact.Writers {
		if writerDef.Enabled && writerDef.Type == "clickhouse" {
			chCfg = &writerDef.ClickHouse
			break
		}
	}

	if chCfg == nil {
		log.Fatalf("No enabled ClickHouse writer found in config. API server cannot start.")
	}

	querier, err := query.NewClickHouseQuerier(*chCfg)
	if err != nil {
		log.Fatalf("Failed to create querier: %v", err)
	}

	r := mux.NewRouter()
	apiHandler := &APIHandler{querier: querier, cfg: cfg}

	r.HandleFunc("/", apiHandler.healthCheckHandler).Methods("GET")
	r.HandleFunc("/search", apiHandler.searchHandler).Methods("POST")
	r.HandleFunc("/query", apiHandler.queryHandler).Methods("POST")
	r.HandleFunc("/api/v1/aggregate", apiHandler.aggregateFlowsHandler).Methods("POST")
	r.HandleFunc("/api/v1/flows/trace", apiHandler.traceFlowHandler).Methods("POST")

	server := &http.Server{
		Addr:    cfg.API.ListenAddr,
		Handler: r,
	}

	go func() {
		log.Printf("API server starting on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on %s: %v", server.Addr, err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("API server shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("API server exited.")
}

// APIHandler holds the dependencies for API handlers.
type APIHandler struct {
	querier query.Querier
	cfg     *config.Config
}

func (h *APIHandler) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *APIHandler) searchHandler(w http.ResponseWriter, r *http.Request) {
	var taskNames []string
	for _, task := range h.cfg.Aggregator.Exact.Tasks {
		taskNames = append(taskNames, task.Name)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(taskNames)
}

func (h *APIHandler) queryHandler(w http.ResponseWriter, r *http.Request) {
	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	startTime := req.Range.From
	if startTime.IsZero() {
		startTime = time.Unix(0, 0) // Default to epoch start time if not provided
	}
	endTime := req.Range.To
	if endTime.IsZero() {
		endTime = time.Now().Add(24 * time.Hour) // Default to 24 hours from now if not provided
	}

	var response []TimeSeriesResponse

	for _, target := range req.Targets {
		aggReq := &v1.AggregationRequest{
			EndTime:   timestamppb.New(endTime),
			TaskName:  target.Target,
		}

		aggResp, err := h.querier.AggregateFlows(r.Context(), aggReq)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var totalPackets float64
		if len(aggResp.Summaries) > 0 {
			totalPackets = float64(aggResp.Summaries[0].TotalPackets)
		}

		ts := TimeSeriesResponse{
			Target: target.Target,
			Datapoints: [][]float64{
				{totalPackets, float64(endTime.Unix() * 1000)},
			},
		}
		response = append(response, ts)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *APIHandler) aggregateFlowsHandler(w http.ResponseWriter, r *http.Request) {
	var req v1.AggregationRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read request body: %v", err), http.StatusBadRequest)
		return
	}
	if err := protojson.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("failed to decode request: %v", err), http.StatusBadRequest)
		return
	}

	resp, err := h.querier.AggregateFlows(r.Context(), &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to query flows: %v", err), http.StatusInternalServerError)
		return
	}

	jsonBytes, err := protojson.Marshal(resp)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to marshal response: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func (h *APIHandler) traceFlowHandler(w http.ResponseWriter, r *http.Request) {
	var req v1.TraceFlowRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read request body: %v", err), http.StatusBadRequest)
		return
	}
	if err := protojson.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("failed to decode request: %v", err), http.StatusBadRequest)
		return
	}

	resp, err := h.querier.TraceFlow(r.Context(), &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to trace flow: %v", err), http.StatusInternalServerError)
		return
	}

	jsonBytes, err := protojson.Marshal(resp)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to marshal response: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}