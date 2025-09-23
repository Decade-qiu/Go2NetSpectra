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
			log.Fatalf("Error: A prompt is required. Use -prompt or provide it as an argument.")
		}
	}

	// 3. Connect to the gRPC server
	conn, err := grpc.NewClient(*address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Did not connect: %v", err)
	}
	defer conn.Close()
	client := v1.NewAIServiceClient(conn)

	// 4. Call the streaming RPC
	log.Println("Sending prompt to AI... (waiting for stream)")
	stream, err := client.AnalyzePromptStream(context.Background(), &v1.AnalyzePromptRequest{Prompt: *prompt})
	if err != nil {
		log.Fatalf("Error calling AnalyzePromptStream: %v", err)
	}

	// 5. Receive and print the stream response
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			// Stream ended
			fmt.Println() // Add a newline for clean terminal output
			break
		}
		if err != nil {
			log.Fatalf("Error receiving stream: %v", err)
		}
		// Print the received text chunk directly to standard output
		fmt.Print(resp.GetChunk())
	}
}
