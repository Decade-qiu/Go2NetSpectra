package alerter

import (
	"net"
	"testing"
	"time"
)

func TestNewAIClientConnectsToReachableAddress(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() unexpected error: %v", err)
	}
	defer listener.Close()

	accepted := make(chan struct{})
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			close(accepted)
			time.Sleep(100 * time.Millisecond)
			_ = conn.Close()
		}
	}()

	client, transport, err := newAIClient(listener.Addr().String())
	if err != nil {
		t.Fatalf("newAIClient() unexpected error: %v", err)
	}
	defer transport.Close()

	if client == nil {
		t.Fatal("newAIClient() client = nil, want non-nil")
	}
	if !transport.IsOpen() {
		t.Fatal("newAIClient() transport is not open")
	}

	select {
	case <-accepted:
	case <-time.After(time.Second):
		t.Fatal("listener did not accept the thrift client connection")
	}
}

func TestNewAIClientRejectsUnreachableAddress(t *testing.T) {
	client, transport, err := newAIClient("127.0.0.1:1")
	if err == nil {
		if transport != nil {
			_ = transport.Close()
		}
		t.Fatalf("newAIClient() client = %#v, transport = %#v, want connection error", client, transport)
	}
}
