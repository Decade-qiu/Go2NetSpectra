package main

import (
	v1 "Go2NetSpectra/api/gen/v1"
	"Go2NetSpectra/internal/ai"
	"Go2NetSpectra/internal/config"
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
)

// server struct now holds different types of analyzers
type server struct {
	v1.UnimplementedAIServiceServer
	alerterAnalyzer *ai.AlerterAnalyzer
	commonAnalyzer  *ai.CommonAnalyzer
}

// AnalyzeTraffic routes to the AlerterAnalyzer.
func (s *server) AnalyzeTraffic(ctx context.Context, req *v1.AnalyzeTrafficRequest) (*v1.AnalyzeTrafficResponse, error) {
	log.Printf("Received AnalyzeTraffic request, routing to AlerterAnalyzer...")
	output, err := s.alerterAnalyzer.AnalyzeTraffic(ctx, req.GetTextInput())
	if err != nil {
		return nil, fmt.Errorf("failed to analyze traffic: %w", err)
	}
	return &v1.AnalyzeTrafficResponse{TextOutput: output}, nil
}

// AnalyzePromptStream implements the streaming RPC.
func (s *server) AnalyzePromptStream(req *v1.AnalyzePromptRequest, stream v1.AIService_AnalyzePromptStreamServer) error {
	log.Printf("Received streaming request for prompt: %s", req.GetPrompt())

	// Define a callback function to send AI-generated chunks to the gRPC stream
	sendChunk := func(chunk string) error {
		return stream.Send(&v1.AnalyzePromptResponse{Chunk: chunk})
	}

	// Call the streaming method of the common analyzer
	return s.commonAnalyzer.AnalyzeStream(stream.Context(), req.GetPrompt(), sendChunk)
}

func main() {
	configFile := flag.String("config", "configs/config.yaml", "Path to the configuration file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create instances for both analyzers
	alerterAnalyzer, err := ai.NewAlerterAnalyzer(&cfg.AI)
	if err != nil {
		log.Fatalf("Failed to create AlerterAnalyzer: %v", err)
	}
	commonAnalyzer, err := ai.NewCommonAnalyzer(&cfg.AI)
	if err != nil {
		log.Fatalf("Failed to create CommonAnalyzer: %v", err)
	}

	lis, err := net.Listen("tcp", cfg.AI.GRPCLisenAddr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	// Register the service with both analyzers
	v1.RegisterAIServiceServer(s, &server{
		alerterAnalyzer: alerterAnalyzer,
		commonAnalyzer:  commonAnalyzer,
	})

	go func() {
		log.Printf("AI-gRPC API server starting on %s", cfg.AI.GRPCLisenAddr)
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Servers shutting down...")

	s.GracefulStop()
}
