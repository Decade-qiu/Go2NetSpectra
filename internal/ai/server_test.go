package ai

import (
	"context"
	"io"
	"testing"
	"time"
)

func TestPromptSessionStoreDeliversChunksAndDone(t *testing.T) {
	store := newPromptSessionStore(
		func(ctx context.Context, prompt string, sendChunk func(string) error) error {
			if prompt != "hello" {
				t.Fatalf("prompt = %q, want %q", prompt, "hello")
			}
			if err := sendChunk("part-1"); err != nil {
				return err
			}
			if err := sendChunk("part-2"); err != nil {
				return err
			}
			return nil
		},
		time.Minute,
	)
	defer store.Stop()

	session, err := store.Start(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Start() unexpected error: %v", err)
	}

	first, err := store.Read(context.Background(), session.SessionID, 1)
	if err != nil {
		t.Fatalf("Read(first) unexpected error: %v", err)
	}
	if len(first.Chunks) != 1 || first.Chunks[0] != "part-1" {
		t.Fatalf("Read(first) chunks = %#v, want %#v", first.Chunks, []string{"part-1"})
	}
	if first.Done {
		t.Fatal("Read(first) done = true, want false")
	}

	second, err := store.Read(context.Background(), session.SessionID, 8)
	if err != nil {
		t.Fatalf("Read(second) unexpected error: %v", err)
	}
	if len(second.Chunks) != 1 || second.Chunks[0] != "part-2" {
		t.Fatalf("Read(second) chunks = %#v, want %#v", second.Chunks, []string{"part-2"})
	}
	if !second.Done {
		t.Fatal("Read(second) done = false, want true")
	}
	if second.ErrorText != nil {
		t.Fatalf("Read(second) error_text = %v, want nil", *second.ErrorText)
	}
}

func TestPromptSessionStoreCancelIsIdempotent(t *testing.T) {
	store := newPromptSessionStore(
		func(ctx context.Context, prompt string, sendChunk func(string) error) error {
			<-ctx.Done()
			return ctx.Err()
		},
		time.Minute,
	)
	defer store.Stop()

	session, err := store.Start(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Start() unexpected error: %v", err)
	}

	if canceled := store.Cancel(session.SessionID); !canceled {
		t.Fatal("Cancel(first) = false, want true")
	}
	if canceled := store.Cancel(session.SessionID); !canceled {
		t.Fatal("Cancel(second) = false, want true")
	}

	resp, err := store.Read(context.Background(), session.SessionID, 4)
	if err != nil {
		t.Fatalf("Read() unexpected error: %v", err)
	}
	if !resp.Done {
		t.Fatal("Read() done = false, want true")
	}
	if resp.ErrorText == nil {
		t.Fatal("Read() error_text = nil, want non-nil")
	}
	if got := *resp.ErrorText; got == "" {
		t.Fatal("Read() error_text = empty string, want non-empty")
	}
}

func TestPromptSessionStorePropagatesTerminalError(t *testing.T) {
	store := newPromptSessionStore(
		func(ctx context.Context, prompt string, sendChunk func(string) error) error {
			return io.ErrUnexpectedEOF
		},
		time.Minute,
	)
	defer store.Stop()

	session, err := store.Start(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Start() unexpected error: %v", err)
	}

	resp, err := store.Read(context.Background(), session.SessionID, 4)
	if err != nil {
		t.Fatalf("Read() unexpected error: %v", err)
	}
	if !resp.Done {
		t.Fatal("Read() done = false, want true")
	}
	if resp.ErrorText == nil {
		t.Fatal("Read() error_text = nil, want non-nil")
	}
	if got := *resp.ErrorText; got == "" {
		t.Fatalf("Read() error_text = %q, want non-empty terminal error text", got)
	}
}
