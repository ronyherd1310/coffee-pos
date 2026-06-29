package security

import (
	"context"
	"testing"
	"time"

	appauth "coffee-pos/backend/internal/app/auth"
)

func TestInMemorySessionStoreReturnsActiveSession(t *testing.T) {
	store := NewInMemorySessionStore()
	now := time.Date(2026, 6, 29, 9, 0, 0, 0, time.UTC)
	session := appauth.Session{
		ID:        "session-1",
		ExpiresAt: now.Add(time.Hour),
	}

	if err := store.Create(context.Background(), session); err != nil {
		t.Fatalf("expected create to succeed: %v", err)
	}

	stored, ok, err := store.Get(context.Background(), "session-1", now)
	if err != nil {
		t.Fatalf("expected get to succeed: %v", err)
	}
	if !ok {
		t.Fatal("expected stored session to be found")
	}
	if stored != session {
		t.Fatalf("expected stored session %+v, got %+v", session, stored)
	}
}

func TestInMemorySessionStoreRemovesExpiredSessionOnLookup(t *testing.T) {
	store := NewInMemorySessionStore()
	now := time.Date(2026, 6, 29, 9, 0, 0, 0, time.UTC)

	if err := store.Create(context.Background(), appauth.Session{
		ID:        "expired",
		ExpiresAt: now.Add(-time.Minute),
	}); err != nil {
		t.Fatalf("expected create to succeed: %v", err)
	}

	_, ok, err := store.Get(context.Background(), "expired", now)
	if err != nil {
		t.Fatalf("expected get to succeed: %v", err)
	}
	if ok {
		t.Fatal("expected expired session to be treated as missing")
	}

	_, ok, err = store.Get(context.Background(), "expired", now)
	if err != nil {
		t.Fatalf("expected repeated get to succeed: %v", err)
	}
	if ok {
		t.Fatal("expected expired session to stay removed")
	}
}

func TestInMemorySessionStoreDeleteIsIdempotent(t *testing.T) {
	store := NewInMemorySessionStore()
	session := appauth.Session{
		ID:        "session-1",
		ExpiresAt: time.Now().Add(time.Hour),
	}

	if err := store.Create(context.Background(), session); err != nil {
		t.Fatalf("expected create to succeed: %v", err)
	}
	if err := store.Delete(context.Background(), "session-1"); err != nil {
		t.Fatalf("expected delete to succeed: %v", err)
	}
	if err := store.Delete(context.Background(), "session-1"); err != nil {
		t.Fatalf("expected repeated delete to be idempotent: %v", err)
	}
}
