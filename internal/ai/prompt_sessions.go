package ai

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	v1 "Go2NetSpectra/api/gen/thrift/v1"
)

const (
	defaultPromptSessionTTL      = 10 * time.Minute
	defaultPromptCleanupInterval = time.Minute
	defaultPromptReadChunkLimit  = 16
)

type promptAnalysisRunner func(ctx context.Context, prompt string, sendChunk func(string) error) error

type promptSessionStore struct {
	mu       sync.Mutex
	sessions map[string]*promptSession
	run      promptAnalysisRunner
	ttl      time.Duration
	done     chan struct{}
	wg       sync.WaitGroup
	stopped  bool
}

type promptSession struct {
	mu          sync.Mutex
	sessionID   string
	cancel      context.CancelFunc
	chunks      []string
	done        bool
	errorText   *string
	lastTouched time.Time
	notify      chan struct{}
}

func newPromptSessionStore(run promptAnalysisRunner, ttl time.Duration) *promptSessionStore {
	if ttl <= 0 {
		ttl = defaultPromptSessionTTL
	}

	store := &promptSessionStore{
		sessions: make(map[string]*promptSession),
		run:      run,
		ttl:      ttl,
		done:     make(chan struct{}),
	}

	store.wg.Add(1)
	go store.cleanupLoop()

	return store
}

func (s *promptSessionStore) Start(ctx context.Context, prompt string) (*v1.PromptAnalysisSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return nil, fmt.Errorf("prompt session store stopped")
	}

	sessionID, err := newPromptSessionID()
	if err != nil {
		return nil, err
	}

	sessionCtx, cancel := context.WithCancel(context.Background())
	session := &promptSession{
		sessionID:   sessionID,
		cancel:      cancel,
		lastTouched: time.Now(),
		notify:      make(chan struct{}),
	}
	s.sessions[sessionID] = session

	s.wg.Add(1)
	go s.runSession(sessionCtx, session, prompt)

	return &v1.PromptAnalysisSession{
		SessionID: sessionID,
		Done:      false,
	}, nil
}

func (s *promptSessionStore) Read(ctx context.Context, sessionID string, maxChunks int32) (*v1.PromptChunkResponse, error) {
	session := s.getSession(sessionID)
	if session == nil {
		return nil, fmt.Errorf("prompt session %q not found", sessionID)
	}

	limit := int(maxChunks)
	if limit <= 0 {
		limit = defaultPromptReadChunkLimit
	}

	for {
		chunks, done, errorText, notify := session.read(limit)
		if len(chunks) > 0 || done {
			resp := &v1.PromptChunkResponse{
				Chunks: chunks,
				Done:   done,
			}
			if errorText != nil && done {
				resp.ErrorText = errorText
			}
			return resp, nil
		}

		select {
		case <-notify:
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-s.done:
			return nil, fmt.Errorf("prompt session store stopped")
		}
	}
}

func (s *promptSessionStore) Cancel(sessionID string) bool {
	session := s.getSession(sessionID)
	if session == nil {
		return false
	}

	session.cancel()
	return true
}

func (s *promptSessionStore) Stop() {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	close(s.done)

	sessions := make([]*promptSession, 0, len(s.sessions))
	for _, session := range s.sessions {
		sessions = append(sessions, session)
	}
	s.mu.Unlock()

	for _, session := range sessions {
		session.cancel()
	}

	s.wg.Wait()
}

func (s *promptSessionStore) cleanupLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(defaultPromptCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.expireSessions(time.Now())
		case <-s.done:
			return
		}
	}
}

func (s *promptSessionStore) expireSessions(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for sessionID, session := range s.sessions {
		if session.expired(now, s.ttl) {
			delete(s.sessions, sessionID)
		}
	}
}

func (s *promptSessionStore) runSession(ctx context.Context, session *promptSession, prompt string) {
	defer s.wg.Done()

	err := s.run(ctx, prompt, func(chunk string) error {
		if chunk == "" {
			return nil
		}
		session.appendChunk(chunk)
		return nil
	})

	session.finish(err)
}

func (s *promptSessionStore) getSession(sessionID string) *promptSession {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessions[sessionID]
}

func (s *promptSessionStore) ReadPromptChunks(ctx context.Context, req *v1.PromptChunkRequest) (*v1.PromptChunkResponse, error) {
	return s.Read(ctx, req.SessionID, req.GetMaxChunks())
}

func (s *promptSessionStore) CancelPromptAnalysis(req *v1.PromptCancelRequest) *v1.PromptCancelResponse {
	return &v1.PromptCancelResponse{Canceled: s.Cancel(req.SessionID)}
}

func (p *promptSession) appendChunk(chunk string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.chunks = append(p.chunks, chunk)
	p.lastTouched = time.Now()
	p.broadcastLocked()
}

func (p *promptSession) finish(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.done = true
	if err != nil {
		errorText := err.Error()
		p.errorText = &errorText
	}
	p.lastTouched = time.Now()
	p.broadcastLocked()
}

func (p *promptSession) read(limit int) ([]string, bool, *string, chan struct{}) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.lastTouched = time.Now()
	if len(p.chunks) == 0 && !p.done {
		return nil, false, nil, p.notify
	}

	if limit > len(p.chunks) {
		limit = len(p.chunks)
	}

	chunks := append([]string(nil), p.chunks[:limit]...)
	p.chunks = append([]string(nil), p.chunks[limit:]...)
	done := p.done && len(p.chunks) == 0

	var errorText *string
	if done && p.errorText != nil {
		errorValue := *p.errorText
		errorText = &errorValue
	}

	return chunks, done, errorText, p.notify
}

func (p *promptSession) expired(now time.Time, ttl time.Duration) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	return now.Sub(p.lastTouched) >= ttl
}

func (p *promptSession) broadcastLocked() {
	close(p.notify)
	p.notify = make(chan struct{})
}

func newPromptSessionID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate prompt session id: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
