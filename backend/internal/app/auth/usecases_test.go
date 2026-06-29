package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestLoginRejectsInvalidPINFormatBeforeHashVerification(t *testing.T) {
	jakarta := mustJakartaLocation(t)
	verifier := &fakeHashVerifier{}
	service := NewService(Dependencies{
		CashierPINHash: "configured-hash",
		Verifier:       verifier,
		Sessions:       newFakeSessionStore(),
		RateLimiter:    newFakeRateLimiter(),
		SessionIDs:     &fakeSessionIDGenerator{id: "session-1"},
		Clock:          fakeClock{now: time.Date(2026, 6, 29, 9, 0, 0, 0, jakarta)},
		Location:       jakarta,
	})

	result, err := service.Login(context.Background(), LoginInput{
		PIN:      "12345",
		ClientID: "client-1",
	})
	if err != nil {
		t.Fatalf("expected invalid pin result, got error: %v", err)
	}
	if result.Status != LoginStatusInvalidPIN {
		t.Fatalf("expected invalid pin status, got %q", result.Status)
	}
	if verifier.calls != 0 {
		t.Fatalf("expected verifier not to be called, got %d calls", verifier.calls)
	}
}

func TestLoginReturnsInvalidPINForIncorrectPINAndRegistersFailure(t *testing.T) {
	jakarta := mustJakartaLocation(t)
	rateLimiter := newFakeRateLimiter()
	service := NewService(Dependencies{
		CashierPINHash: "configured-hash",
		Verifier: &fakeHashVerifier{
			matches: false,
		},
		Sessions:    newFakeSessionStore(),
		RateLimiter: rateLimiter,
		SessionIDs:  &fakeSessionIDGenerator{id: "session-1"},
		Clock:       fakeClock{now: time.Date(2026, 6, 29, 9, 0, 0, 0, jakarta)},
		Location:    jakarta,
	})

	result, err := service.Login(context.Background(), LoginInput{
		PIN:      "123456",
		ClientID: "client-1",
	})
	if err != nil {
		t.Fatalf("expected invalid pin result, got error: %v", err)
	}
	if result.Status != LoginStatusInvalidPIN {
		t.Fatalf("expected invalid pin status, got %q", result.Status)
	}
	if failures := rateLimiter.failureCount("client-1"); failures != 1 {
		t.Fatalf("expected one recorded failure, got %d", failures)
	}
}

func TestLoginReturnsTooManyAttemptsWithoutVerifyingBlockedClient(t *testing.T) {
	jakarta := mustJakartaLocation(t)
	verifier := &fakeHashVerifier{matches: true}
	rateLimiter := newFakeRateLimiter()
	rateLimiter.blocked["client-1"] = true
	service := NewService(Dependencies{
		CashierPINHash: "configured-hash",
		Verifier:       verifier,
		Sessions:       newFakeSessionStore(),
		RateLimiter:    rateLimiter,
		SessionIDs:     &fakeSessionIDGenerator{id: "session-1"},
		Clock:          fakeClock{now: time.Date(2026, 6, 29, 9, 0, 0, 0, jakarta)},
		Location:       jakarta,
	})

	result, err := service.Login(context.Background(), LoginInput{
		PIN:      "123456",
		ClientID: "client-1",
	})
	if err != nil {
		t.Fatalf("expected rate-limit result, got error: %v", err)
	}
	if result.Status != LoginStatusTooManyAttempts {
		t.Fatalf("expected too many attempts status, got %q", result.Status)
	}
	if verifier.calls != 0 {
		t.Fatalf("expected verifier not to be called, got %d calls", verifier.calls)
	}
}

func TestLoginCreatesFreshSessionAndResetsFailuresOnSuccess(t *testing.T) {
	jakarta := mustJakartaLocation(t)
	now := time.Date(2026, 6, 29, 10, 0, 0, 0, jakarta)
	rateLimiter := newFakeRateLimiter()
	rateLimiter.failures["client-1"] = 3
	sessions := newFakeSessionStore()
	service := NewService(Dependencies{
		CashierPINHash: "configured-hash",
		Verifier: &fakeHashVerifier{
			matches: true,
		},
		Sessions:    sessions,
		RateLimiter: rateLimiter,
		SessionIDs:  &fakeSessionIDGenerator{id: "session-1"},
		Clock:       fakeClock{now: now},
		Location:    jakarta,
	})

	result, err := service.Login(context.Background(), LoginInput{
		PIN:      "123456",
		ClientID: "client-1",
	})
	if err != nil {
		t.Fatalf("expected successful login, got error: %v", err)
	}
	if result.Status != LoginStatusAuthenticated {
		t.Fatalf("expected authenticated status, got %q", result.Status)
	}
	if result.Session.ID != "session-1" {
		t.Fatalf("expected session-1, got %q", result.Session.ID)
	}
	expectedExpiry := time.Date(2026, 6, 29, 22, 0, 0, 0, jakarta)
	if !result.Session.ExpiresAt.Equal(expectedExpiry) {
		t.Fatalf("expected expiry %s, got %s", expectedExpiry, result.Session.ExpiresAt)
	}
	if failures := rateLimiter.failureCount("client-1"); failures != 0 {
		t.Fatalf("expected failures reset, got %d", failures)
	}
	if _, ok := sessions.sessions["session-1"]; !ok {
		t.Fatal("expected session to be stored")
	}
}

func TestLoginReplacesOnlyCurrentSessionFromRequest(t *testing.T) {
	jakarta := mustJakartaLocation(t)
	now := time.Date(2026, 6, 29, 9, 0, 0, 0, jakarta)
	sessions := newFakeSessionStore()
	sessions.sessions["current-session"] = Session{
		ID:        "current-session",
		ExpiresAt: now.Add(2 * time.Hour),
	}
	sessions.sessions["other-session"] = Session{
		ID:        "other-session",
		ExpiresAt: now.Add(2 * time.Hour),
	}
	service := NewService(Dependencies{
		CashierPINHash: "configured-hash",
		Verifier: &fakeHashVerifier{
			matches: true,
		},
		Sessions:    sessions,
		RateLimiter: newFakeRateLimiter(),
		SessionIDs:  &fakeSessionIDGenerator{id: "fresh-session"},
		Clock:       fakeClock{now: now},
		Location:    jakarta,
	})

	result, err := service.Login(context.Background(), LoginInput{
		PIN:              "123456",
		ClientID:         "client-1",
		CurrentSessionID: "current-session",
	})
	if err != nil {
		t.Fatalf("expected successful login, got error: %v", err)
	}
	if result.Session.ID != "fresh-session" {
		t.Fatalf("expected fresh-session, got %q", result.Session.ID)
	}
	if _, ok := sessions.sessions["current-session"]; ok {
		t.Fatal("expected current session to be removed")
	}
	if _, ok := sessions.sessions["other-session"]; !ok {
		t.Fatal("expected unrelated session to remain valid")
	}
}

func TestLogoutInvalidatesSessionAndIsIdempotent(t *testing.T) {
	sessions := newFakeSessionStore()
	sessions.sessions["session-1"] = Session{
		ID:        "session-1",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	service := NewService(Dependencies{
		Sessions: sessions,
	})

	if err := service.Logout(context.Background(), "session-1"); err != nil {
		t.Fatalf("expected logout to succeed: %v", err)
	}
	if _, ok := sessions.sessions["session-1"]; ok {
		t.Fatal("expected session to be removed")
	}
	if err := service.Logout(context.Background(), "session-1"); err != nil {
		t.Fatalf("expected repeated logout to be idempotent: %v", err)
	}
}

func TestSessionAuthenticatedReturnsFalseForMissingOrExpiredSessions(t *testing.T) {
	jakarta := mustJakartaLocation(t)
	now := time.Date(2026, 6, 29, 9, 0, 0, 0, jakarta)
	sessions := newFakeSessionStore()
	sessions.sessions["expired"] = Session{
		ID:        "expired",
		ExpiresAt: now.Add(-time.Minute),
	}
	service := NewService(Dependencies{
		Sessions: sessions,
		Clock:    fakeClock{now: now},
	})

	missing, err := service.Session(context.Background(), "missing")
	if err != nil {
		t.Fatalf("expected missing session lookup to succeed: %v", err)
	}
	if missing.Authenticated {
		t.Fatal("expected missing session to be unauthenticated")
	}

	expired, err := service.Session(context.Background(), "expired")
	if err != nil {
		t.Fatalf("expected expired session lookup to succeed: %v", err)
	}
	if expired.Authenticated {
		t.Fatal("expected expired session to be unauthenticated")
	}
}

func TestSessionAuthenticatedReturnsTrueForActiveSession(t *testing.T) {
	jakarta := mustJakartaLocation(t)
	now := time.Date(2026, 6, 29, 9, 0, 0, 0, jakarta)
	sessions := newFakeSessionStore()
	sessions.sessions["active"] = Session{
		ID:        "active",
		ExpiresAt: now.Add(time.Hour),
	}
	service := NewService(Dependencies{
		Sessions: sessions,
		Clock:    fakeClock{now: now},
	})

	result, err := service.Session(context.Background(), "active")
	if err != nil {
		t.Fatalf("expected active session lookup to succeed: %v", err)
	}
	if !result.Authenticated {
		t.Fatal("expected active session to be authenticated")
	}
}

func TestLoginReturnsInfrastructureErrorsExplicitly(t *testing.T) {
	jakarta := mustJakartaLocation(t)
	now := time.Date(2026, 6, 29, 9, 0, 0, 0, jakarta)
	expectedErr := errors.New("boom")

	testCases := []struct {
		name string
		deps Dependencies
	}{
		{
			name: "verifier error",
			deps: Dependencies{
				CashierPINHash: "configured-hash",
				Verifier: &fakeHashVerifier{
					err: expectedErr,
				},
				Sessions:    newFakeSessionStore(),
				RateLimiter: newFakeRateLimiter(),
				SessionIDs:  &fakeSessionIDGenerator{id: "session-1"},
				Clock:       fakeClock{now: now},
				Location:    jakarta,
			},
		},
		{
			name: "session creation error",
			deps: Dependencies{
				CashierPINHash: "configured-hash",
				Verifier: &fakeHashVerifier{
					matches: true,
				},
				Sessions:    &fakeSessionStore{createErr: expectedErr, sessions: map[string]Session{}},
				RateLimiter: newFakeRateLimiter(),
				SessionIDs:  &fakeSessionIDGenerator{id: "session-1"},
				Clock:       fakeClock{now: now},
				Location:    jakarta,
			},
		},
		{
			name: "session id generation error",
			deps: Dependencies{
				CashierPINHash: "configured-hash",
				Verifier: &fakeHashVerifier{
					matches: true,
				},
				Sessions:    newFakeSessionStore(),
				RateLimiter: newFakeRateLimiter(),
				SessionIDs:  &fakeSessionIDGenerator{err: expectedErr},
				Clock:       fakeClock{now: now},
				Location:    jakarta,
			},
		},
		{
			name: "rate limiter reset error",
			deps: Dependencies{
				CashierPINHash: "configured-hash",
				Verifier: &fakeHashVerifier{
					matches: true,
				},
				Sessions:    newFakeSessionStore(),
				RateLimiter: &fakeRateLimiter{resetErr: expectedErr, failures: map[string]int{}, blocked: map[string]bool{}},
				SessionIDs:  &fakeSessionIDGenerator{id: "session-1"},
				Clock:       fakeClock{now: now},
				Location:    jakarta,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			service := NewService(testCase.deps)

			_, err := service.Login(context.Background(), LoginInput{
				PIN:      "123456",
				ClientID: "client-1",
			})
			if !errors.Is(err, expectedErr) {
				t.Fatalf("expected %v, got %v", expectedErr, err)
			}
		})
	}
}

func TestLoginAllowsAttemptAgainAfterRateLimitWindowRollsOver(t *testing.T) {
	jakarta := mustJakartaLocation(t)
	now := time.Date(2026, 6, 29, 9, 0, 0, 0, jakarta)
	rateLimiter := newFakeRateLimiter()
	for range 5 {
		_ = rateLimiter.RegisterFailure(context.Background(), "client-1", now)
	}

	service := NewService(Dependencies{
		CashierPINHash: "configured-hash",
		Verifier: &fakeHashVerifier{
			matches: false,
		},
		Sessions:    newFakeSessionStore(),
		RateLimiter: rateLimiter,
		SessionIDs:  &fakeSessionIDGenerator{id: "session-1"},
		Clock:       fakeClock{now: now.Add(5*time.Minute + time.Second)},
		Location:    jakarta,
	})

	result, err := service.Login(context.Background(), LoginInput{
		PIN:      "123456",
		ClientID: "client-1",
	})
	if err != nil {
		t.Fatalf("expected invalid pin after rate-limit rollover, got error: %v", err)
	}
	if result.Status != LoginStatusInvalidPIN {
		t.Fatalf("expected invalid pin after rollover, got %q", result.Status)
	}
}

func mustJakartaLocation(t *testing.T) *time.Location {
	t.Helper()

	location, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatalf("expected Asia/Jakarta location to load: %v", err)
	}

	return location
}

type fakeHashVerifier struct {
	matches bool
	err     error
	calls   int
}

func (f *fakeHashVerifier) VerifyPINHash(_ context.Context, _ string, _ string) (bool, error) {
	f.calls++
	if f.err != nil {
		return false, f.err
	}

	return f.matches, nil
}

type fakeSessionIDGenerator struct {
	id  string
	err error
}

func (f *fakeSessionIDGenerator) NewSessionID() (string, error) {
	if f.err != nil {
		return "", f.err
	}

	return f.id, nil
}

type fakeSessionStore struct {
	sessions   map[string]Session
	createErr  error
	getErr     error
	deleteErr  error
	deleteHits int
}

func newFakeSessionStore() *fakeSessionStore {
	return &fakeSessionStore{sessions: map[string]Session{}}
}

func (f *fakeSessionStore) Create(_ context.Context, session Session) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.sessions[session.ID] = session
	return nil
}

func (f *fakeSessionStore) Get(_ context.Context, sessionID string, now time.Time) (Session, bool, error) {
	if f.getErr != nil {
		return Session{}, false, f.getErr
	}
	session, ok := f.sessions[sessionID]
	if !ok {
		return Session{}, false, nil
	}
	if !session.ExpiresAt.After(now) {
		delete(f.sessions, sessionID)
		return Session{}, false, nil
	}
	return session, true, nil
}

func (f *fakeSessionStore) Delete(_ context.Context, sessionID string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deleteHits++
	delete(f.sessions, sessionID)
	return nil
}

type fakeRateLimiter struct {
	records   map[string][]time.Time
	resetErr  error
	blockErr  error
	recordErr error
	failures  map[string]int
	blocked   map[string]bool
}

func newFakeRateLimiter() *fakeRateLimiter {
	return &fakeRateLimiter{
		records:  map[string][]time.Time{},
		failures: map[string]int{},
		blocked:  map[string]bool{},
	}
}

func (f *fakeRateLimiter) IsBlocked(_ context.Context, clientID string, now time.Time) (bool, error) {
	if f.blockErr != nil {
		return false, f.blockErr
	}
	if f.records == nil {
		f.records = map[string][]time.Time{}
	}
	if f.failures == nil {
		f.failures = map[string]int{}
	}
	if f.blocked == nil {
		f.blocked = map[string]bool{}
	}
	if f.blocked[clientID] {
		return true, nil
	}
	events := pruneWindow(f.records[clientID], now)
	f.records[clientID] = events
	return len(events) >= 5, nil
}

func (f *fakeRateLimiter) RegisterFailure(_ context.Context, clientID string, now time.Time) error {
	if f.recordErr != nil {
		return f.recordErr
	}
	if f.records == nil {
		f.records = map[string][]time.Time{}
	}
	if f.failures == nil {
		f.failures = map[string]int{}
	}
	events := pruneWindow(f.records[clientID], now)
	events = append(events, now)
	f.records[clientID] = events
	f.failures[clientID] = len(events)
	return nil
}

func (f *fakeRateLimiter) Reset(_ context.Context, clientID string) error {
	if f.resetErr != nil {
		return f.resetErr
	}
	if f.records == nil {
		f.records = map[string][]time.Time{}
	}
	if f.failures == nil {
		f.failures = map[string]int{}
	}
	if f.blocked == nil {
		f.blocked = map[string]bool{}
	}
	delete(f.records, clientID)
	delete(f.failures, clientID)
	delete(f.blocked, clientID)
	return nil
}

func (f *fakeRateLimiter) failureCount(clientID string) int {
	return f.failures[clientID]
}

func pruneWindow(events []time.Time, now time.Time) []time.Time {
	filtered := events[:0]
	windowStart := now.Add(-5 * time.Minute)
	for _, event := range events {
		if event.After(windowStart) {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

type fakeClock struct {
	now time.Time
}

func (f fakeClock) Now() time.Time {
	return f.now
}
