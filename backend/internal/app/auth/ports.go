package auth

import (
	"context"
	"time"
)

type HashVerifier interface {
	VerifyPINHash(ctx context.Context, pin string, hash string) (bool, error)
}

type SessionStore interface {
	Create(ctx context.Context, session Session) error
	Get(ctx context.Context, sessionID string, now time.Time) (Session, bool, error)
	Delete(ctx context.Context, sessionID string) error
}

type RateLimiter interface {
	IsBlocked(ctx context.Context, clientID string, now time.Time) (bool, error)
	RegisterFailure(ctx context.Context, clientID string, now time.Time) error
	Reset(ctx context.Context, clientID string) error
}

type SessionIDGenerator interface {
	NewSessionID() (string, error)
}

type Clock interface {
	Now() time.Time
}
