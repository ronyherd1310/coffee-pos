package security

import (
	"context"
	"sync"
	"time"

	appauth "coffee-pos/backend/internal/app/auth"
)

type InMemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string]appauth.Session
}

func NewInMemorySessionStore() *InMemorySessionStore {
	return &InMemorySessionStore{
		sessions: map[string]appauth.Session{},
	}
}

func (s *InMemorySessionStore) Create(_ context.Context, session appauth.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[session.ID] = session
	return nil
}

func (s *InMemorySessionStore) Get(_ context.Context, sessionID string, now time.Time) (appauth.Session, bool, error) {
	s.mu.RLock()
	session, ok := s.sessions[sessionID]
	s.mu.RUnlock()
	if !ok {
		return appauth.Session{}, false, nil
	}

	if !session.ExpiresAt.After(now) {
		s.mu.Lock()
		delete(s.sessions, sessionID)
		s.mu.Unlock()
		return appauth.Session{}, false, nil
	}

	return session, true, nil
}

func (s *InMemorySessionStore) Delete(_ context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, sessionID)
	return nil
}
