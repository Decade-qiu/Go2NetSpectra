package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"slices"
	"time"

	v1 "Go2NetSpectra/api/gen/thrift/v1"
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/query"

	thrift "github.com/apache/thrift/lib/go/thrift"
)

const queryRPCBufferSize = 32 * 1024

// QueryServiceServer implements the repository Thrift query API.
type QueryServiceServer struct {
	exactQuerier  query.Querier
	sketchQuerier query.Querier
	cfg           *config.Config
}

// RunQueryAPIServers starts the Thrift query API and the Grafana-compatible HTTP endpoint.
func RunQueryAPIServers(ctx context.Context, cfg *config.Config) error {
	service, err := newQueryServiceServer(cfg)
	if err != nil {
		return err
	}

	serverTransport, err := thrift.NewTServerSocket(cfg.API.RPCListenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", cfg.API.RPCListenAddr, err)
	}

	transportFactory := thrift.NewTBufferedTransportFactory(queryRPCBufferSize)
	protocolFactory := thrift.NewTBinaryProtocolFactoryConf(&thrift.TConfiguration{})
	processor := v1.NewQueryServiceProcessor(service)
	rpcServer := thrift.NewTSimpleServer4(processor, serverTransport, transportFactory, protocolFactory)

	httpServer := &http.Server{
		Addr:    cfg.API.HTTPListenAddr,
		Handler: newGrafanaHTTPHandler(service),
	}

	errCh := make(chan error, 2)
	go func() {
		log.Printf("Query RPC server starting on %s", cfg.API.RPCListenAddr)
		if serveErr := rpcServer.Serve(); serveErr != nil {
			errCh <- fmt.Errorf("failed to serve thrift rpc: %w", serveErr)
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
		if stopErr := rpcServer.Stop(); stopErr != nil {
			log.Printf("failed to stop query rpc server after serve error: %v", stopErr)
		}
		if shutdownErr := shutdownHTTPServer(httpServer); shutdownErr != nil {
			log.Printf("failed to shut down http server after rpc error: %v", shutdownErr)
		}
		return err
	case <-ctx.Done():
	}

	if err := rpcServer.Stop(); err != nil {
		return fmt.Errorf("failed to stop query rpc server: %w", err)
	}
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
	result, err := s.aggregateFlows(ctx, aggregationRequestFromThrift(req))
	if err != nil {
		return nil, err
	}
	return queryTotalCountsResponseToThrift(result), nil
}

// TraceFlow executes exact flow tracing queries.
func (s *QueryServiceServer) TraceFlow(ctx context.Context, req *v1.TraceFlowRequest) (*v1.TraceFlowResponse, error) {
	result, err := s.traceFlow(ctx, traceFlowRequestFromThrift(req))
	if err != nil {
		return nil, err
	}
	return &v1.TraceFlowResponse{Lifecycle: flowLifecycleToThrift(result)}, nil
}

// QueryHeavyHitters executes sketch heavy hitter queries.
func (s *QueryServiceServer) QueryHeavyHitters(ctx context.Context, req *v1.HeavyHittersRequest) (*v1.HeavyHittersResponse, error) {
	result, err := s.queryHeavyHitters(ctx, heavyHittersRequestFromThrift(req))
	if err != nil {
		return nil, err
	}
	return heavyHittersResponseToThrift(result), nil
}

func (s *QueryServiceServer) aggregateFlows(ctx context.Context, req *query.AggregationRequest) (*query.QueryTotalCountsResponse, error) {
	if s.exactQuerier == nil {
		return nil, fmt.Errorf("exact aggregator is not configured, cannot perform aggregation query")
	}
	log.Printf("Received AggregateFlows request for task: %s, end: %v", req.TaskName, req.EndTime)
	return s.exactQuerier.AggregateFlows(ctx, req)
}

func (s *QueryServiceServer) traceFlow(ctx context.Context, req *query.TraceFlowRequest) (*query.FlowLifecycle, error) {
	if s.exactQuerier == nil {
		return nil, fmt.Errorf("exact aggregator is not configured, cannot perform trace query")
	}
	log.Printf("Received TraceFlow request for task: %s, flow: %v, end: %v", req.TaskName, req.FlowKeys, req.EndTime)
	return s.exactQuerier.TraceFlow(ctx, req)
}

func (s *QueryServiceServer) queryHeavyHitters(ctx context.Context, req *query.HeavyHittersRequest) (*query.HeavyHittersResponse, error) {
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

func aggregationRequestFromThrift(req *v1.AggregationRequest) *query.AggregationRequest {
	if req == nil {
		return &query.AggregationRequest{}
	}

	return &query.AggregationRequest{
		EndTime:  timePtrFromOptionalUnixNano(req.IsSetEndTimeUnixNano(), req.GetEndTimeUnixNano()),
		TaskName: req.GetTaskName(),
		SrcIP:    optionalString(req.IsSetSrcIP(), req.GetSrcIP()),
		DstIP:    optionalString(req.IsSetDstIP(), req.GetDstIP()),
		SrcPort:  int32PtrFromOptional(req.IsSetSrcPort(), req.GetSrcPort()),
		DstPort:  int32PtrFromOptional(req.IsSetDstPort(), req.GetDstPort()),
		Protocol: int32PtrFromOptional(req.IsSetProtocol(), req.GetProtocol()),
	}
}

func traceFlowRequestFromThrift(req *v1.TraceFlowRequest) *query.TraceFlowRequest {
	if req == nil {
		return &query.TraceFlowRequest{}
	}

	flowKeys := make(map[string]string, len(req.GetFlowKeys()))
	for key, value := range req.GetFlowKeys() {
		flowKeys[key] = value
	}

	return &query.TraceFlowRequest{
		TaskName: req.GetTaskName(),
		FlowKeys: flowKeys,
		EndTime:  timePtrFromOptionalUnixNano(req.IsSetEndTimeUnixNano(), req.GetEndTimeUnixNano()),
	}
}

func heavyHittersRequestFromThrift(req *v1.HeavyHittersRequest) *query.HeavyHittersRequest {
	if req == nil {
		return &query.HeavyHittersRequest{}
	}

	return &query.HeavyHittersRequest{
		TaskName: req.GetTaskName(),
		Type:     req.GetType(),
		EndTime:  timePtrFromOptionalUnixNano(req.IsSetEndTimeUnixNano(), req.GetEndTimeUnixNano()),
		Limit:    req.GetLimit(),
	}
}

func queryTotalCountsResponseToThrift(resp *query.QueryTotalCountsResponse) *v1.QueryTotalCountsResponse {
	if resp == nil {
		return &v1.QueryTotalCountsResponse{Summaries: []*v1.TaskSummary{}}
	}

	summaries := make([]*v1.TaskSummary, 0, len(resp.Summaries))
	for _, summary := range resp.Summaries {
		summaries = append(summaries, &v1.TaskSummary{
			TaskName:     summary.TaskName,
			TotalBytes:   summary.TotalBytes,
			TotalPackets: summary.TotalPackets,
			FlowCount:    summary.FlowCount,
		})
	}

	return &v1.QueryTotalCountsResponse{Summaries: summaries}
}

func flowLifecycleToThrift(lifecycle *query.FlowLifecycle) *v1.FlowLifecycle {
	if lifecycle == nil {
		return &v1.FlowLifecycle{}
	}

	return &v1.FlowLifecycle{
		FirstSeenUnixNano: lifecycle.FirstSeen.UnixNano(),
		LastSeenUnixNano:  lifecycle.LastSeen.UnixNano(),
		TotalPackets:      lifecycle.TotalPackets,
		TotalBytes:        lifecycle.TotalBytes,
	}
}

func heavyHittersResponseToThrift(resp *query.HeavyHittersResponse) *v1.HeavyHittersResponse {
	if resp == nil {
		return &v1.HeavyHittersResponse{Hitters: []*v1.HeavyHitter{}}
	}

	hitters := make([]*v1.HeavyHitter, 0, len(resp.Hitters))
	for _, hitter := range resp.Hitters {
		hitters = append(hitters, &v1.HeavyHitter{
			Flow:  hitter.Flow,
			Value: hitter.Value,
		})
	}

	return &v1.HeavyHittersResponse{Hitters: hitters}
}

func timePtrFromOptionalUnixNano(isSet bool, unixNano int64) *time.Time {
	if !isSet {
		return nil
	}
	value := time.Unix(0, unixNano)
	return &value
}

func int32PtrFromOptional(isSet bool, value int32) *int32 {
	if !isSet {
		return nil
	}
	copyValue := value
	return &copyValue
}

func optionalString(isSet bool, value string) string {
	if !isSet {
		return ""
	}
	return value
}
