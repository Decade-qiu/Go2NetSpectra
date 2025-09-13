package main

import (
	v1 "Go2NetSpectra/api/gen/v1"
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/query"
	"context"
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
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Find the first enabled ClickHouse writer config
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

	// Initialize querier with the found config
	querier, err := query.NewClickHouseQuerier(*chCfg)
	if err != nil {
		log.Fatalf("Failed to create querier: %v", err)
	}

	// Initialize router
	r := mux.NewRouter()

	// Create API handler with querier dependency
	apiHandler := &APIHandler{querier: querier}

	// Define API routes
	r.HandleFunc("/api/v1/aggregate", apiHandler.aggregateFlowsHandler).Methods("POST")
	r.HandleFunc("/api/v1/flows/trace", apiHandler.traceFlowHandler).Methods("POST")

	// Start HTTP server
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
}

// aggregateFlowsHandler handles aggregation queries.
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

// traceFlowHandler handles tracing a single flow's lifecycle.
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
