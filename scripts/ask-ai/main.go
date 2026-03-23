package main

import (
	v1 "Go2NetSpectra/api/gen/v1"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// 1. Parse command-line flags
	address := flag.String("port", "localhost:50052", "The server port")
	prompt := flag.String("prompt", "", "The prompt to send to the AI model")
	flag.Parse()

	// 2. If prompt is empty, read it from non-flag arguments
	if *prompt == "" {
		if flag.NArg() > 0 {
			*prompt = strings.Join(flag.Args(), " ")
		} else {
			log.Fatalf("error: a prompt is required; use -prompt or provide it as an argument")
		}
	}

	// 3. Connect to the gRPC server
	conn, err := grpc.NewClient(*address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to %s: %v", *address, err)
	}
	defer conn.Close()
	client := v1.NewAIServiceClient(conn)

	// 4. Call the streaming RPC
	log.Println("Sending prompt to AI and waiting for the response stream...")
	stream, err := client.AnalyzePromptStream(context.Background(), &v1.AnalyzePromptRequest{Prompt: *prompt})
	if err != nil {
		log.Fatalf("failed to call AnalyzePromptStream: %v", err)
	}

	// 5. Receive and print the stream response
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			fmt.Println()
			break
		}
		if err != nil {
			log.Fatalf("failed to receive stream response: %v", err)
		}
		fmt.Print(resp.GetChunk())
	}
}
