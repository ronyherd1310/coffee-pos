package security

import (
	"context"
	"sync"
	"time"
)

const (
	loginWindowDuration = 5 * time.Minute
	loginAttemptLimit   = 5
)

type InMemoryRateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
}

func NewInMemoryRateLimiter() *InMemoryRateLimiter {
	return &InMemoryRateLimiter{
		attempts: map[string][]time.Time{},
	}
}

func (l *InMemoryRateLimiter) IsBlocked(_ context.Context, clientID string, now time.Time) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.attempts[clientID] = pruneAttempts(l.attempts[clientID], now)
	return len(l.attempts[clientID]) >= loginAttemptLimit, nil
}

func (l *InMemoryRateLimiter) RegisterFailure(_ context.Context, clientID string, now time.Time) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	events := pruneAttempts(l.attempts[clientID], now)
	l.attempts[clientID] = append(events, now)
	return nil
}

func (l *InMemoryRateLimiter) Reset(_ context.Context, clientID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.attempts, clientID)
	return nil
}

func pruneAttempts(events []time.Time, now time.Time) []time.Time {
	if len(events) == 0 {
		return nil
	}

	windowStart := now.Add(-loginWindowDuration)
	filtered := events[:0]
	for _, event := range events {
		if event.After(windowStart) {
			filtered = append(filtered, event)
		}
	}
	return filtered
}
