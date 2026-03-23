package ai

import (
	"context"
	"fmt"
	"log"
	"net"

	v1 "Go2NetSpectra/api/gen/v1"
	"Go2NetSpectra/internal/config"

	"google.golang.org/grpc"
)

// Server exposes the AI gRPC API.
type Server struct {
	v1.UnimplementedAIServiceServer
	alerterAnalyzer *AlerterAnalyzer
	commonAnalyzer  *CommonAnalyzer
}

// RunServer starts the AI gRPC server and blocks until shutdown.
func RunServer(ctx context.Context, cfg *config.Config) error {
	alerterAnalyzer, err := NewAlerterAnalyzer(&cfg.AI)
	if err != nil {
		return fmt.Errorf("failed to create alerter analyzer: %w", err)
	}
	commonAnalyzer, err := NewCommonAnalyzer(&cfg.AI)
	if err != nil {
		return fmt.Errorf("failed to create common analyzer: %w", err)
	}

	listener, err := net.Listen("tcp", cfg.AI.GRPCListenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", cfg.AI.GRPCListenAddr, err)
	}
	defer listener.Close()

	server := grpc.NewServer()
	v1.RegisterAIServiceServer(server, &Server{
		alerterAnalyzer: alerterAnalyzer,
		commonAnalyzer:  commonAnalyzer,
	})

	errCh := make(chan error, 1)
	go func() {
		log.Printf("AI-gRPC API server starting on %s", cfg.AI.GRPCListenAddr)
		if serveErr := server.Serve(listener); serveErr != nil {
			errCh <- fmt.Errorf("failed to serve grpc: %w", serveErr)
		}
	}()

	select {
	case err := <-errCh:
		server.GracefulStop()
		return err
	case <-ctx.Done():
		server.GracefulStop()
		return nil
	}
}

// AnalyzeTraffic routes alert summaries to the alert-focused analyzer.
func (s *Server) AnalyzeTraffic(ctx context.Context, req *v1.AnalyzeTrafficRequest) (*v1.AnalyzeTrafficResponse, error) {
	log.Printf("Received AnalyzeTraffic request, routing to AlerterAnalyzer...")
	output, err := s.alerterAnalyzer.AnalyzeTraffic(ctx, req.GetTextInput())
	if err != nil {
		return nil, fmt.Errorf("failed to analyze traffic: %w", err)
	}
	return &v1.AnalyzeTrafficResponse{TextOutput: output}, nil
}

// AnalyzePromptStream streams chunked AI responses back to the client.
func (s *Server) AnalyzePromptStream(req *v1.AnalyzePromptRequest, stream v1.AIService_AnalyzePromptStreamServer) error {
	log.Printf("Received streaming request for prompt: %s", req.GetPrompt())
	sendChunk := func(chunk string) error {
		return stream.Send(&v1.AnalyzePromptResponse{Chunk: chunk})
	}
	return s.commonAnalyzer.AnalyzeStream(stream.Context(), req.GetPrompt(), sendChunk)
}
