package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	authdomain "coffee-pos/backend/internal/domain/auth"
)

type LoginStatus string

const (
	LoginStatusAuthenticated   LoginStatus = "authenticated"
	LoginStatusInvalidPIN      LoginStatus = "invalid_pin"
	LoginStatusTooManyAttempts LoginStatus = "too_many_attempts"
)

type Session struct {
	ID        string
	ExpiresAt time.Time
}

type LoginInput struct {
	PIN              string
	ClientID         string
	CurrentSessionID string
}

type LoginResult struct {
	Status  LoginStatus
	Session Session
}

type SessionResult struct {
	Authenticated bool
	Session       Session
}

type Dependencies struct {
	CashierPINHash string
	Verifier       HashVerifier
	Sessions       SessionStore
	RateLimiter    RateLimiter
	SessionIDs     SessionIDGenerator
	Clock          Clock
	Location       *time.Location
}

type Service struct {
	cashierPINHash string
	verifier       HashVerifier
	sessions       SessionStore
	rateLimiter    RateLimiter
	sessionIDs     SessionIDGenerator
	clock          Clock
	location       *time.Location
}

func NewService(deps Dependencies) *Service {
	location := deps.Location
	if location == nil {
		location = time.UTC
	}

	return &Service{
		cashierPINHash: deps.CashierPINHash,
		verifier:       deps.Verifier,
		sessions:       deps.Sessions,
		rateLimiter:    deps.RateLimiter,
		sessionIDs:     deps.SessionIDs,
		clock:          deps.Clock,
		location:       location,
	}
}

func (s *Service) Login(ctx context.Context, input LoginInput) (LoginResult, error) {
	now := s.now()

	if s.rateLimiter != nil {
		blocked, err := s.rateLimiter.IsBlocked(ctx, input.ClientID, now)
		if err != nil {
			return LoginResult{}, fmt.Errorf("check login rate limit: %w", err)
		}
		if blocked {
			return LoginResult{Status: LoginStatusTooManyAttempts}, nil
		}
	}

	if err := authdomain.ValidatePIN(input.PIN); err != nil {
		if err := s.registerFailure(ctx, input.ClientID, now); err != nil {
			return LoginResult{}, err
		}
		return LoginResult{Status: LoginStatusInvalidPIN}, nil
	}

	if s.verifier == nil {
		return LoginResult{}, errors.New("verify pin: missing hash verifier")
	}
	matches, err := s.verifier.VerifyPINHash(ctx, input.PIN, s.cashierPINHash)
	if err != nil {
		return LoginResult{}, fmt.Errorf("verify pin: %w", err)
	}
	if !matches {
		if err := s.registerFailure(ctx, input.ClientID, now); err != nil {
			return LoginResult{}, err
		}
		return LoginResult{Status: LoginStatusInvalidPIN}, nil
	}

	if err := s.resetFailures(ctx, input.ClientID); err != nil {
		return LoginResult{}, err
	}
	if s.sessionIDs == nil {
		return LoginResult{}, errors.New("create session: missing session id generator")
	}
	sessionID, err := s.sessionIDs.NewSessionID()
	if err != nil {
		return LoginResult{}, fmt.Errorf("create session id: %w", err)
	}
	if sessionID == "" {
		return LoginResult{}, errors.New("create session id: generated empty session id")
	}
	if s.sessions == nil {
		return LoginResult{}, errors.New("create session: missing session store")
	}

	session := Session{
		ID:        sessionID,
		ExpiresAt: authdomain.SessionExpiry(now, s.location),
	}
	if err := s.sessions.Create(ctx, session); err != nil {
		return LoginResult{}, fmt.Errorf("create session: %w", err)
	}
	if err := s.replaceCurrentSession(ctx, input.CurrentSessionID, session.ID, now); err != nil {
		return LoginResult{}, err
	}

	return LoginResult{
		Status:  LoginStatusAuthenticated,
		Session: session,
	}, nil
}

func (s *Service) Logout(ctx context.Context, sessionID string) error {
	if sessionID == "" || s.sessions == nil {
		return nil
	}
	if err := s.sessions.Delete(ctx, sessionID); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (s *Service) Session(ctx context.Context, sessionID string) (SessionResult, error) {
	if sessionID == "" || s.sessions == nil {
		return SessionResult{Authenticated: false}, nil
	}

	session, ok, err := s.sessions.Get(ctx, sessionID, s.now())
	if err != nil {
		return SessionResult{}, fmt.Errorf("get session: %w", err)
	}
	if !ok {
		return SessionResult{Authenticated: false}, nil
	}

	return SessionResult{
		Authenticated: true,
		Session:       session,
	}, nil
}

func (s *Service) replaceCurrentSession(ctx context.Context, currentSessionID string, newSessionID string, now time.Time) error {
	if currentSessionID == "" || currentSessionID == newSessionID {
		return nil
	}

	_, ok, err := s.sessions.Get(ctx, currentSessionID, now)
	if err != nil {
		_ = s.sessions.Delete(ctx, newSessionID)
		return fmt.Errorf("lookup current session: %w", err)
	}
	if !ok {
		return nil
	}

	if err := s.sessions.Delete(ctx, currentSessionID); err != nil {
		_ = s.sessions.Delete(ctx, newSessionID)
		return fmt.Errorf("replace current session: %w", err)
	}

	return nil
}

func (s *Service) registerFailure(ctx context.Context, clientID string, now time.Time) error {
	if s.rateLimiter == nil {
		return nil
	}
	if err := s.rateLimiter.RegisterFailure(ctx, clientID, now); err != nil {
		return fmt.Errorf("record login failure: %w", err)
	}
	return nil
}

func (s *Service) resetFailures(ctx context.Context, clientID string) error {
	if s.rateLimiter == nil {
		return nil
	}
	if err := s.rateLimiter.Reset(ctx, clientID); err != nil {
		return fmt.Errorf("reset login failures: %w", err)
	}
	return nil
}

func (s *Service) now() time.Time {
	if s.clock == nil {
		return time.Now()
	}
	return s.clock.Now()
}
