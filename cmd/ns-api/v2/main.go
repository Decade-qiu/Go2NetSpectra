package main

import (
	v1 "Go2NetSpectra/api/gen/v1"
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/query"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ---- Grafana-specific structs ----
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

// ---- gRPC service implementation ----
type QueryServiceServer struct {
	v1.UnimplementedQueryServiceServer
	exactQuerier  query.Querier
	sketchQuerier query.Querier
	cfg           *config.Config
}

func (s *QueryServiceServer) HealthCheck(ctx context.Context, req *v1.HealthCheckRequest) (*v1.HealthCheckResponse, error) {
	log.Println("Received HealthCheck request")
	return &v1.HealthCheckResponse{Status: "ok"}, nil
}

func (s *QueryServiceServer) SearchTasks(ctx context.Context, req *v1.SearchTasksRequest) (*v1.SearchTasksResponse, error) {
	log.Println("Received SearchTasks request")
	var taskNames []string
	// This can be expanded to include sketch tasks if needed
	for _, task := range s.cfg.Aggregator.Exact.Tasks {
		taskNames = append(taskNames, task.Name)
	}
	for _, task := range s.cfg.Aggregator.Sketch.Tasks {
		taskNames = append(taskNames, task.Name)
	}
	return &v1.SearchTasksResponse{TaskNames: taskNames}, nil
}

func (s *QueryServiceServer) AggregateFlows(ctx context.Context, req *v1.AggregationRequest) (*v1.QueryTotalCountsResponse, error) {
	if s.exactQuerier == nil {
		return nil, fmt.Errorf("exact aggregator is not configured, cannot perform aggregation query")
	}
	log.Printf("Received AggregateFlows request for task: %s, end: %v", req.TaskName, req.EndTime)
	return s.exactQuerier.AggregateFlows(ctx, req)
}

func (s *QueryServiceServer) TraceFlow(ctx context.Context, req *v1.TraceFlowRequest) (*v1.TraceFlowResponse, error) {
	if s.exactQuerier == nil {
		return nil, fmt.Errorf("exact aggregator is not configured, cannot perform trace query")
	}
	log.Printf("Received TraceFlow request for task: %s, flow: %v, end: %v", req.TaskName, req.FlowKeys, req.EndTime)
	result, err := s.exactQuerier.TraceFlow(ctx, req)
	var resultProto v1.TraceFlowResponse
	resultProto.Lifecycle = result
	return &resultProto, err
}

func (s *QueryServiceServer) QueryHeavyHitters(ctx context.Context, req *v1.HeavyHittersRequest) (*v1.HeavyHittersResponse, error) {
	if s.sketchQuerier == nil {
		return nil, fmt.Errorf("sketch aggregator is not configured, cannot perform heavy hitters query")
	}
	log.Printf("Received QueryHeavyHitters request for task: %s, type: %v, end: %v, limit: %d", req.TaskName, req.Type, req.EndTime, req.Limit)
	return s.sketchQuerier.QueryHeavyHitters(ctx, req)
}

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	var exactQuerier, sketchQuerier query.Querier

	// Create querier for the exact aggregator if it's active
	if slices.Contains(cfg.Aggregator.Types, "exact") {
		for _, writerDef := range cfg.Aggregator.Exact.Writers {
			if writerDef.Enabled && writerDef.Type == "clickhouse" {
				log.Println("Found enabled ClickHouse writer for exact aggregator.")
				exactQuerier, err = query.NewClickHouseQuerier(writerDef.ClickHouse)
				if err != nil {
					log.Fatalf("Failed to create exact querier: %v", err)
				}
				break
			}
		}
	}

	// Create querier for the sketch aggregator if it's active
	if slices.Contains(cfg.Aggregator.Types, "sketch") {
		for _, writerDef := range cfg.Aggregator.Sketch.Writers {
			if writerDef.Enabled && writerDef.Type == "clickhouse" {
				log.Println("Found enabled ClickHouse writer for sketch aggregator.")
				sketchQuerier, err = query.NewClickHouseQuerier(writerDef.ClickHouse)
				if err != nil {
					log.Fatalf("Failed to create sketch querier: %v", err)
			}
				break
			}
		}
	}

	if exactQuerier == nil && sketchQuerier == nil {
		log.Fatalf("No enabled ClickHouse writer found for any active aggregator. API server cannot start.")
	}

	service := &QueryServiceServer{
		exactQuerier:  exactQuerier,
		sketchQuerier: sketchQuerier,
		cfg:           cfg,
	}

	// Run gRPC server
	grpcServer := grpc.NewServer()
	v1.RegisterQueryServiceServer(grpcServer, service)

	lis, err := net.Listen("tcp", cfg.API.GrpcListenAddr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", cfg.API.GrpcListenAddr, err)
	}
	go func() {
		log.Printf("gRPC API server starting on %s", cfg.API.GrpcListenAddr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// Run HTTP server for Grafana
	httpServer := &http.Server{
		Addr:    cfg.API.HttpListenAddr,
		Handler: newHTTPHandler(service),
	}

	go func() {
		log.Printf("HTTP server (Grafana) starting on %s", cfg.API.HttpListenAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Servers shutting down...")

	grpcServer.GracefulStop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	httpServer.Shutdown(ctx)

	log.Println("All servers exited.")
}

// ---- HTTP handler for Grafana ----
func newHTTPHandler(s *QueryServiceServer) http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/query", func(w http.ResponseWriter, r *http.Request) {
		var req QueryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		startTime := req.Range.From
		if startTime.IsZero() {
			startTime = time.Unix(0, 0)
		}
		endTime := req.Range.To
		if endTime.IsZero() {
			endTime = time.Now().Add(24 * time.Hour)
		}

		var response []TimeSeriesResponse
		for _, target := range req.Targets {
			aggReq := &v1.AggregationRequest{
				EndTime:  timestamppb.New(endTime),
				TaskName: target.Target,
			}

			aggResp, err := s.AggregateFlows(r.Context(), aggReq)
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
	}).Methods("POST")

	return r
}