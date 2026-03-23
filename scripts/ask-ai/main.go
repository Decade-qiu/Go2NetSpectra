package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	v1 "Go2NetSpectra/api/gen/thrift/v1"

	thrift "github.com/apache/thrift/lib/go/thrift"
)

const askAITransportBufferSize = 32 * 1024

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

	// 3. Connect to the RPC server
	client, transport, err := newAIClient(*address)
	if err != nil {
		log.Fatalf("failed to connect to %s: %v", *address, err)
	}
	defer transport.Close()

	// 4. Start prompt-analysis session
	log.Println("Sending prompt to AI and waiting for chunked responses...")
	ctx := context.Background()
	session, err := client.StartPromptAnalysis(ctx, &v1.PromptAnalysisRequest{Prompt: *prompt})
	if err != nil {
		log.Fatalf("failed to call StartPromptAnalysis: %v", err)
	}

	// 5. Poll and print chunked responses
	maxChunks := int32(8)
	for {
		resp, err := client.ReadPromptChunks(ctx, &v1.PromptChunkRequest{
			SessionID: session.SessionID,
			MaxChunks: &maxChunks,
		})
		if err != nil {
			log.Fatalf("failed to read prompt chunks: %v", err)
		}
		for _, chunk := range resp.GetChunks() {
			fmt.Print(chunk)
		}
		if resp.Done {
			fmt.Println()
			if resp.ErrorText != nil {
				log.Fatalf("prompt analysis failed: %s", *resp.ErrorText)
			}
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func newAIClient(addr string) (*v1.AIServiceClient, thrift.TTransport, error) {
	conf := &thrift.TConfiguration{
		ConnectTimeout: 5 * time.Second,
		SocketTimeout:  60 * time.Second,
	}
	socket := thrift.NewTSocketConf(addr, conf)
	transportFactory := thrift.NewTBufferedTransportFactory(askAITransportBufferSize)
	transport, err := transportFactory.GetTransport(socket)
	if err != nil {
		return nil, nil, fmt.Errorf("build thrift transport: %w", err)
	}
	if err := transport.Open(); err != nil {
		return nil, nil, fmt.Errorf("open thrift transport: %w", err)
	}

	protocolFactory := thrift.NewTBinaryProtocolFactoryConf(conf)
	return v1.NewAIServiceClientFactory(transport, protocolFactory), transport, nil
}
