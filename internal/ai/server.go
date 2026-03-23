package ai

import (
	"context"
	"fmt"
	"log"

	v1 "Go2NetSpectra/api/gen/thrift/v1"
	"Go2NetSpectra/internal/config"

	thrift "github.com/apache/thrift/lib/go/thrift"
)

const thriftBufferSize = 32 * 1024

// Server exposes the AI Thrift RPC API.
type Server struct {
	alerterAnalyzer *AlerterAnalyzer
	commonAnalyzer  *CommonAnalyzer
	promptSessions  *promptSessionStore
}

func newServer(cfg *config.Config) (*Server, error) {
	alerterAnalyzer, err := NewAlerterAnalyzer(&cfg.AI)
	if err != nil {
		return nil, fmt.Errorf("failed to create alerter analyzer: %w", err)
	}
	commonAnalyzer, err := NewCommonAnalyzer(&cfg.AI)
	if err != nil {
		return nil, fmt.Errorf("failed to create common analyzer: %w", err)
	}

	return &Server{
		alerterAnalyzer: alerterAnalyzer,
		commonAnalyzer:  commonAnalyzer,
		promptSessions:  newPromptSessionStore(commonAnalyzer.AnalyzeStream, defaultPromptSessionTTL),
	}, nil
}

// RunServer starts the AI Thrift RPC server and blocks until shutdown.
func RunServer(ctx context.Context, cfg *config.Config) error {
	service, err := newServer(cfg)
	if err != nil {
		return err
	}
	defer service.promptSessions.Stop()

	serverTransport, err := thrift.NewTServerSocket(cfg.AI.RPCListenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", cfg.AI.RPCListenAddr, err)
	}

	transportFactory := thrift.NewTBufferedTransportFactory(thriftBufferSize)
	protocolFactory := thrift.NewTBinaryProtocolFactoryConf(&thrift.TConfiguration{})
	processor := v1.NewAIServiceProcessor(service)
	server := thrift.NewTSimpleServer4(processor, serverTransport, transportFactory, protocolFactory)

	errCh := make(chan error, 1)
	go func() {
		log.Printf("AI RPC server starting on %s", cfg.AI.RPCListenAddr)
		if serveErr := server.Serve(); serveErr != nil {
			errCh <- fmt.Errorf("failed to serve thrift rpc: %w", serveErr)
		}
	}()

	select {
	case err := <-errCh:
		if stopErr := server.Stop(); stopErr != nil {
			log.Printf("AI RPC server stop error after serve failure: %v", stopErr)
		}
		return err
	case <-ctx.Done():
	}

	if err := server.Stop(); err != nil {
		return fmt.Errorf("failed to stop ai rpc server: %w", err)
	}

	return nil
}

// AnalyzeTraffic routes alert summaries to the alert-focused analyzer.
func (s *Server) AnalyzeTraffic(ctx context.Context, req *v1.AnalyzeTrafficRequest) (*v1.AnalyzeTrafficResponse, error) {
	log.Printf("Received AnalyzeTraffic request, routing to AlerterAnalyzer")
	output, err := s.alerterAnalyzer.AnalyzeTraffic(ctx, req.GetTextInput())
	if err != nil {
		return nil, fmt.Errorf("failed to analyze traffic: %w", err)
	}
	return &v1.AnalyzeTrafficResponse{TextOutput: output}, nil
}

// StartPromptAnalysis creates a session for incremental prompt analysis.
func (s *Server) StartPromptAnalysis(ctx context.Context, req *v1.PromptAnalysisRequest) (*v1.PromptAnalysisSession, error) {
	log.Printf("Received StartPromptAnalysis request")
	return s.promptSessions.Start(ctx, req.GetPrompt())
}

// ReadPromptChunks reads queued chunks for an active prompt session.
func (s *Server) ReadPromptChunks(ctx context.Context, req *v1.PromptChunkRequest) (*v1.PromptChunkResponse, error) {
	return s.promptSessions.Read(ctx, req.GetSessionID(), req.GetMaxChunks())
}

// CancelPromptAnalysis cancels a prompt-analysis session.
func (s *Server) CancelPromptAnalysis(ctx context.Context, req *v1.PromptCancelRequest) (*v1.PromptCancelResponse, error) {
	return &v1.PromptCancelResponse{Canceled: s.promptSessions.Cancel(req.GetSessionID())}, nil
}
