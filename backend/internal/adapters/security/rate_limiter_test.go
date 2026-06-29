package security

import (
	"context"
	"testing"
	"time"
)

func TestInMemoryRateLimiterBlocksSixthAttemptWithinFiveMinutes(t *testing.T) {
	limiter := NewInMemoryRateLimiter()
	clientID := "client-1"
	now := time.Date(2026, 6, 29, 9, 0, 0, 0, time.UTC)

	for range 5 {
		blocked, err := limiter.IsBlocked(context.Background(), clientID, now)
		if err != nil {
			t.Fatalf("expected pre-failure block check to succeed: %v", err)
		}
		if blocked {
			t.Fatal("expected first five attempts not to be blocked")
		}
		if err := limiter.RegisterFailure(context.Background(), clientID, now); err != nil {
			t.Fatalf("expected failure registration to succeed: %v", err)
		}
	}

	blocked, err := limiter.IsBlocked(context.Background(), clientID, now.Add(4*time.Minute+59*time.Second))
	if err != nil {
		t.Fatalf("expected sixth-attempt block check to succeed: %v", err)
	}
	if !blocked {
		t.Fatal("expected sixth attempt at 4:59 to be blocked")
	}
}

func TestInMemoryRateLimiterAllowsNewAttemptAfterWindowRollsOver(t *testing.T) {
	limiter := NewInMemoryRateLimiter()
	clientID := "client-1"
	now := time.Date(2026, 6, 29, 9, 0, 0, 0, time.UTC)

	for range 5 {
		if err := limiter.RegisterFailure(context.Background(), clientID, now); err != nil {
			t.Fatalf("expected failure registration to succeed: %v", err)
		}
	}

	blocked, err := limiter.IsBlocked(context.Background(), clientID, now.Add(5*time.Minute+time.Second))
	if err != nil {
		t.Fatalf("expected rollover block check to succeed: %v", err)
	}
	if blocked {
		t.Fatal("expected new attempt at 5:01 not to be blocked")
	}
}

func TestInMemoryRateLimiterResetClearsFailures(t *testing.T) {
	limiter := NewInMemoryRateLimiter()
	clientID := "client-1"
	now := time.Date(2026, 6, 29, 9, 0, 0, 0, time.UTC)

	for range 3 {
		if err := limiter.RegisterFailure(context.Background(), clientID, now); err != nil {
			t.Fatalf("expected failure registration to succeed: %v", err)
		}
	}
	if err := limiter.Reset(context.Background(), clientID); err != nil {
		t.Fatalf("expected reset to succeed: %v", err)
	}

	blocked, err := limiter.IsBlocked(context.Background(), clientID, now)
	if err != nil {
		t.Fatalf("expected block check after reset to succeed: %v", err)
	}
	if blocked {
		t.Fatal("expected reset to clear failures")
	}
}

func TestInMemoryRateLimiterSupportsConcurrentAccess(t *testing.T) {
	limiter := NewInMemoryRateLimiter()
	clientID := "client-1"
	now := time.Date(2026, 6, 29, 9, 0, 0, 0, time.UTC)
	done := make(chan struct{}, 20)

	for i := range 10 {
		go func(offset int) {
			defer func() { done <- struct{}{} }()
			at := now.Add(time.Duration(offset) * time.Second)
			_, _ = limiter.IsBlocked(context.Background(), clientID, at)
			_ = limiter.RegisterFailure(context.Background(), clientID, at)
		}(i)
	}

	for range 10 {
		<-done
	}
}
