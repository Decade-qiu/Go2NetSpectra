package api

import (
	v1 "Go2NetSpectra/api/gen/v1"
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/query"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"slices"
	"time"

	"google.golang.org/grpc"
)

// QueryServiceServer implements the repository gRPC query API.
type QueryServiceServer struct {
	v1.UnimplementedQueryServiceServer
	exactQuerier  query.Querier
	sketchQuerier query.Querier
	cfg           *config.Config
}

// RunQueryAPIServers starts the gRPC query API and the Grafana-compatible HTTP endpoint.
func RunQueryAPIServers(ctx context.Context, cfg *config.Config) error {
	service, err := newQueryServiceServer(cfg)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	v1.RegisterQueryServiceServer(grpcServer, service)

	listener, err := net.Listen("tcp", cfg.API.GRPCListenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", cfg.API.GRPCListenAddr, err)
	}
	defer listener.Close()

	httpServer := &http.Server{
		Addr:    cfg.API.HTTPListenAddr,
		Handler: newGrafanaHTTPHandler(service),
	}

	errCh := make(chan error, 2)
	go func() {
		log.Printf("gRPC API server starting on %s", cfg.API.GRPCListenAddr)
		if serveErr := grpcServer.Serve(listener); serveErr != nil {
			errCh <- fmt.Errorf("failed to serve grpc: %w", serveErr)
		}
	}()
	go func() {
		log.Printf("HTTP server (Grafana) starting on %s", cfg.API.HTTPListenAddr)
		if serveErr := httpServer.ListenAndServe(); serveErr != nil && serveErr != http.ErrServerClosed {
			errCh <- fmt.Errorf("failed to serve http: %w", serveErr)
		}
	}()

	select {
	case err := <-errCh:
		grpcServer.GracefulStop()
		if shutdownErr := shutdownHTTPServer(httpServer); shutdownErr != nil {
			log.Printf("failed to shut down http server after grpc error: %v", shutdownErr)
		}
		return err
	case <-ctx.Done():
	}

	grpcServer.GracefulStop()
	if err := shutdownHTTPServer(httpServer); err != nil {
		return err
	}

	return nil
}

// HealthCheck responds with the current service health.
func (s *QueryServiceServer) HealthCheck(ctx context.Context, req *v1.HealthCheckRequest) (*v1.HealthCheckResponse, error) {
	log.Println("Received HealthCheck request")
	return &v1.HealthCheckResponse{Status: "ok"}, nil
}

// SearchTasks returns configured task names for the enabled aggregators.
func (s *QueryServiceServer) SearchTasks(ctx context.Context, req *v1.SearchTasksRequest) (*v1.SearchTasksResponse, error) {
	log.Println("Received SearchTasks request")
	taskNames := make([]string, 0, len(s.cfg.Aggregator.Exact.Tasks)+len(s.cfg.Aggregator.Sketch.Tasks))
	for _, task := range s.cfg.Aggregator.Exact.Tasks {
		taskNames = append(taskNames, task.Name)
	}
	for _, task := range s.cfg.Aggregator.Sketch.Tasks {
		taskNames = append(taskNames, task.Name)
	}
	return &v1.SearchTasksResponse{TaskNames: taskNames}, nil
}

// AggregateFlows executes exact aggregation queries.
func (s *QueryServiceServer) AggregateFlows(ctx context.Context, req *v1.AggregationRequest) (*v1.QueryTotalCountsResponse, error) {
	if s.exactQuerier == nil {
		return nil, fmt.Errorf("exact aggregator is not configured, cannot perform aggregation query")
	}
	log.Printf("Received AggregateFlows request for task: %s, end: %v", req.TaskName, req.EndTime)
	return s.exactQuerier.AggregateFlows(ctx, req)
}

// TraceFlow executes exact flow tracing queries.
func (s *QueryServiceServer) TraceFlow(ctx context.Context, req *v1.TraceFlowRequest) (*v1.TraceFlowResponse, error) {
	if s.exactQuerier == nil {
		return nil, fmt.Errorf("exact aggregator is not configured, cannot perform trace query")
	}
	log.Printf("Received TraceFlow request for task: %s, flow: %v, end: %v", req.TaskName, req.FlowKeys, req.EndTime)
	result, err := s.exactQuerier.TraceFlow(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.TraceFlowResponse{Lifecycle: result}, nil
}

// QueryHeavyHitters executes sketch heavy hitter queries.
func (s *QueryServiceServer) QueryHeavyHitters(ctx context.Context, req *v1.HeavyHittersRequest) (*v1.HeavyHittersResponse, error) {
	if s.sketchQuerier == nil {
		return nil, fmt.Errorf("sketch aggregator is not configured, cannot perform heavy hitters query")
	}
	log.Printf("Received QueryHeavyHitters request for task: %s, type: %v, end: %v, limit: %d", req.TaskName, req.Type, req.EndTime, req.Limit)
	return s.sketchQuerier.QueryHeavyHitters(ctx, req)
}

func newQueryServiceServer(cfg *config.Config) (*QueryServiceServer, error) {
	var exactQuerier query.Querier
	if slices.Contains(cfg.Aggregator.Types, "exact") {
		querier, err := querierFromWriters(cfg.Aggregator.Exact.Writers)
		if err != nil {
			return nil, fmt.Errorf("failed to create exact querier: %w", err)
		}
		exactQuerier = querier
	}

	var sketchQuerier query.Querier
	if slices.Contains(cfg.Aggregator.Types, "sketch") {
		querier, err := querierFromWriters(cfg.Aggregator.Sketch.Writers)
		if err != nil {
			return nil, fmt.Errorf("failed to create sketch querier: %w", err)
		}
		sketchQuerier = querier
	}

	if exactQuerier == nil && sketchQuerier == nil {
		return nil, fmt.Errorf("no enabled clickhouse writer found for any active aggregator")
	}

	return &QueryServiceServer{
		exactQuerier:  exactQuerier,
		sketchQuerier: sketchQuerier,
		cfg:           cfg,
	}, nil
}

func newExactQuerier(cfg *config.Config) (query.Querier, error) {
	querier, err := querierFromWriters(cfg.Aggregator.Exact.Writers)
	if err != nil {
		return nil, err
	}
	if querier == nil {
		return nil, fmt.Errorf("no enabled clickhouse writer found in config")
	}
	return querier, nil
}

func querierFromWriters(writers []config.WriterDef) (query.Querier, error) {
	for _, writerDef := range writers {
		if !writerDef.Enabled || writerDef.Type != "clickhouse" {
			continue
		}
		return query.NewClickHouseQuerier(writerDef.ClickHouse)
	}
	return nil, nil
}

func shutdownHTTPServer(server *http.Server) error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to shut down http server: %w", err)
	}

	return nil
}
