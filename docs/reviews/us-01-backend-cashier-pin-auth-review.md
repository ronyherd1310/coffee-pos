# Code Review: US-01 Backend Cashier PIN Authentication Implementation

**Date:** 2026-06-29
**Reviewer:** opencode
**Document:** `docs/plan/us-01-backend-cashier-pin-auth-plan.md`
**Status:** No Critical Findings

---

## Verification Commands

```bash
go -C backend clean -testcache && go -C backend test ./...
go -C backend vet ./...
go -C backend test -race ./...
go -C backend test -tags=integration ./...
```

All commands pass with zero failures.

---

## Findings

### Severity: Low

#### 1. Rate limiter uses full mutex for read and write operations

**File:** `backend/internal/adapters/security/rate_limiter.go:14-17`

`InMemoryRateLimiter` uses a single `sync.Mutex` for all operations including `IsBlocked` (a read-only check). The session store (`session_store.go:12`) correctly uses `sync.RWMutex` for read/write separation. Under high-concurrent login attempts from the same client, this serializes all rate-limit checks unnecessarily.

**Recommendation:** Switch to `sync.RWMutex` — use `RLock` for `IsBlocked`, full `Lock` for `RegisterFailure` and `Reset`.

#### 2. `clientIdentifier` relies solely on `RemoteAddr`

**File:** `backend/internal/adapters/http/auth_handlers.go:156-165`

The plan acknowledges this risk (line 322): "Client identifier from IP can be inaccurate behind proxies." The implementation uses `net.SplitHostPort(r.RemoteAddr)` with no `X-Forwarded-For` or `X-Real-IP` trust path. Behind a reverse proxy or load balancer, all clients share the same rate-limit bucket, making rate limiting ineffective.

**Recommendation:** Acceptable for MVP single-container deployment. Document that multi-instance or proxied deployments need a trusted-header configuration path. This matches the plan's stated mitigation.

#### 3. No test coverage for empty-string PIN value

**File:** `backend/internal/adapters/http/auth_handlers_test.go:18-47`

The malformed-request test cases include `""` (empty body), `null pin`, `numeric pin`, and `short pin`, but do not include `{"pin":""}`. The implementation handles this correctly (`ValidatePIN` rejects empty strings), but the HTTP layer has no explicit test for this case.

**Recommendation:** Add a test case: `{name: "empty pin string", body: '{"pin":""}'}` to `TestLoginReturnsInvalidPINForMalformedRequests`.

#### 4. `loginBodyLimit` is a hardcoded constant

**File:** `backend/internal/adapters/http/auth_handlers.go:14`

The 1024-byte body limit is hardcoded. For the MVP this is fine, but production deployments may need to tune this without a code change.

**Recommendation:** Acceptable for MVP. Consider moving to `RouterOptions` or config if needed later.

#### 5. `handleLogout` always clears cookie even if service is nil

**File:** `backend/internal/adapters/http/auth_handlers.go:76-86`

When `h.service == nil`, the handler skips the service call but still clears the cookie and returns `200 OK`. This is consistent with the plan's idempotent-logout requirement, but the asymmetry with `handleLogin` (which returns `500` when service is nil) may confuse future maintainers.

**Recommendation:** Acceptable as-is. The nil-service check is a defensive guard for incomplete wiring; both handlers fail safely. Add a code comment if desired.

---

## Positive Observations

### Architecture and Design

- **Clean hexagonal architecture:** Domain rules (`internal/domain/auth`) have zero dependencies. Application ports (`internal/app/auth/ports.go`) define interfaces. Adapters implement those interfaces. HTTP handlers depend only on the application service. This matches the plan's dependency graph precisely.
- **Session lifecycle is well-specified:** Concurrent sessions are allowed, only the current request's session is replaced on re-login, and other browser/device sessions remain valid. This matches the plan's architecture decision.
- **Rate-limit reset on success:** A successful login resets the failed-attempt window for that client identifier. The plan's ambiguity here is resolved correctly.
- **Infrastructure errors are never masqueraded as auth failures:** Hash verifier errors, session-store write failures, session ID generation failures, and rate-limiter storage failures all propagate as explicit errors. The HTTP layer maps these to `500` rather than `401`.

### Domain Rules

- **PIN validation** (`domain/auth/pin.go`): Correctly rejects exactly the set of invalid formats specified in the plan: empty, short, long, non-numeric, whitespace-padded, and full-width digits.
- **Session expiry** (`domain/auth/session.go`): Correctly computes the earlier of 12 hours from login or end-of-day in Asia/Jakarta. Boundary tests at `23:59`, `00:00`, and `12:00` prove the expiry never returns an already-expired time.

### Security Adapters

- **Bcrypt PIN hash** (`adapters/security/pin_hash.go`): Uses `golang.org/x/crypto/bcrypt` as planned. Validates PIN format before hashing. The `HashPIN` method is only used by the CLI command and test helpers — not exposed in the application service.
- **Cryptographic session IDs** (`adapters/security/session_id.go`): 32 bytes of `crypto/rand` encoded as hex (64 characters). No claims or user data encoded in the cookie value.
- **In-memory session store** (`adapters/security/session_store.go`): Uses `sync.RWMutex` for concurrent access. Expired sessions are cleaned up on lookup (lazy expiry).
- **In-memory rate limiter** (`adapters/security/rate_limiter.go`): Window-based pruning with 5-attempt limit over 5 minutes. Concurrent access test passes under `-race`.

### HTTP Layer

- **Cookie attributes:** `HttpOnly`, `SameSite=Lax`, `Secure` when configured, `Path=/`, `Expires` aligned to server-side session expiry. The `clearSessionCookie` function sets both `Expires` to epoch and `MaxAge: -1` to ensure browser removal.
- **Body size limit:** `MaxBytesReader` at 1024 bytes prevents oversized payload attacks. Oversized bodies produce `401` not `500`.
- **Extra JSON fields are ignored:** The handler decodes into `map[string]any` and extracts only `"pin"`, so extra fields have no effect.
- **Session endpoint returns `200` with `{"authenticated": false}` for all non-authenticated states:** Unknown, tampered, expired, and invalidated cookies all return the same response, avoiding information leakage.

### CLI Command

- **`hash-pin` subcommand** (`cmd/coffee-pos/main.go:42-55`): Validates PIN format via `ValidatePIN` before hashing. Prints only the hash to stdout. Test verifies the generated hash can be used for login.

### Test Coverage

- **Unit tests** cover: invalid format before hash verification, incorrect PIN registers failure, rate-limited client skips verification, successful login creates session and resets failures, repeated login replaces only current session, idempotent logout, expired/missing sessions, and all infrastructure error paths.
- **Integration tests** cover: full auth flow (login → session check → protected route → logout), rate limiting over HTTP, and health endpoint.
- **Race check passes** under `-race` flag for all packages.

---

## Critical Findings Resolution

No findings in this review are marked Critical. The review contains Low-severity findings only, so no code or test changes were required under the requested scope.

**Verification:** Not run; no Critical fixes or behavior changes were made.

---

## Summary

The implementation is well-structured, follows the plan precisely, and all acceptance criteria are met. The findings are low-severity observations about test coverage gaps and minor optimization opportunities. No blocking issues were found. All verification commands pass.

**Verdict:** Approved with minor findings. No implementation changes required for the current scope.
