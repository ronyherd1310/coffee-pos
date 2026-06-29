# Implementation Plan: US-01 Backend Cashier PIN Authentication

**Status:** Ready for code review

## Overview

Implement the backend portion of `US-01: Sign In With Cashier PIN` from `docs/specs/small-coffee-shop-pos-mvp-spec.md`. This plan covers PIN validation and verification, server-side sessions, secure session cookies, logout, session status, login rate limiting, and backend protection for future POS API routes. Frontend PIN entry, browser redirects, and POS screen access UI are intentionally out of scope; backend protected endpoints should return authentication failures that the frontend can use to redirect to the PIN login screen.

## Scope

In scope:

- Backend-only PIN login API.
- Exact 6-digit PIN format validation in backend domain/application code.
- Slow-hash PIN verification using configured hash material, not plaintext PIN configuration.
- Server-side session creation, lookup, expiration, and invalidation.
- `HttpOnly`, `SameSite=Lax` or stricter, and production `Secure` session cookie settings.
- Session expiration at the end of the Asia/Jakarta business day or after 12 hours, whichever comes first.
- Logout endpoint that invalidates the current server-side session.
- Session status endpoint for frontend bootstrapping.
- Login rate limiting starting at 5 failed attempts per 5 minutes per client identifier.
- Authentication middleware for protected backend POS API routes.
- Backend unit and HTTP integration tests for US-01 behavior.

Out of scope:

- Frontend PIN screen, redirect behavior, or API client changes.
- Individual cashier identity, roles, password reset, or PIN management UI.
- Database-backed user accounts.
- Menu, order, payment, queue-number, ticket, or report implementation.
- Production secret values committed to source control.

## Architecture Decisions

- Keep auth business rules out of HTTP handlers by adding `internal/domain/auth` for PIN/session value rules and `internal/app/auth` for login, logout, and session-check use cases.
- Store only a slow hash of the predefined cashier PIN in backend configuration, exposed as `CASHIER_PIN_HASH`.
- Standardize auth-related environment variables as `CASHIER_PIN_HASH`, `APP_ENV`, `SESSION_COOKIE_NAME`, and `SESSION_COOKIE_SECURE`.
- Add a hash-verification adapter under `internal/adapters/security`; bcrypt is the likely first implementation because it is established for slow secret verification and available through `golang.org/x/crypto/bcrypt`.
- Use an in-memory server-side session store and in-memory rate limiter for the MVP backend slice. This matches the single small API process assumption and can be replaced later behind application ports if persistence or multi-instance deployment is needed.
- Use opaque random session IDs in cookies. Do not encode auth claims in client-visible tokens and do not rely on `localStorage`.
- Allow concurrent sessions for the shared cashier PIN across multiple browsers or devices. A successful login always issues a fresh session ID; if that request already has a valid session cookie, only that previous session is invalidated and replaced. Other active sessions remain valid until logout or expiry.
- Reset the failed-attempt window after a successful login that is allowed through the rate limiter. Once a client identifier is already blocked, login attempts return the rate-limit response until the 5-minute window rolls over.
- Return JSON `401 Unauthorized` for unauthenticated protected API requests. Frontend/browser redirection to the PIN login screen belongs to the frontend.
- Return JSON `401 Unauthorized` with `{"error":"invalid_pin"}` for invalid-format, missing, wrongly typed, or incorrect PIN login attempts. Return JSON `429 Too Many Requests` with `{"error":"too_many_attempts"}` for blocked login attempts.
- Keep `GET /api/health`, `POST /api/auth/login`, and `GET /api/auth/session` public. Login remains rate-limited, logout is idempotent without a valid session, and future POS API groups require a valid session.
- Return `200 OK` from `GET /api/auth/session` with `{ "authenticated": false }` when no valid session exists, so frontend bootstrapping can treat signed-out state as expected state instead of an error.
- Include a small developer-only command to generate a cashier PIN hash, such as `go -C backend run ./cmd/coffee-pos auth hash-pin 123456`. The command should print only the generated hash and should not store the PIN or hash.
- Load Asia/Jakarta location in backend configuration or a clock/session service so expiration logic is deterministic in tests.

## Dependency Graph

```text
Config for PIN hash, session secret/cookie settings, and app environment
  -> auth domain PIN/session rules
  -> auth application ports and use cases
      -> hash verifier adapter
      -> server-side session store adapter
      -> rate limiter adapter
      -> clock/timezone adapter
          -> HTTP auth handlers
              -> auth middleware for protected POS APIs
                  -> integration tests for login/session/logout/protection
```

## Task List

### Phase 1: Auth Foundation

## Task 1: Add Auth Configuration

**Description:** Extend backend configuration so the API can be started with the cashier PIN hash and session/cookie settings required for US-01. Configuration should fail clearly when required production auth material is missing, while tests can construct config values directly.

**Acceptance criteria:**

- [ ] Backend config supports `CASHIER_PIN_HASH` for the slow-hash value.
- [ ] Backend config supports `APP_ENV`, `SESSION_COOKIE_NAME`, `SESSION_COOKIE_SECURE`, and Asia/Jakarta business timezone loading.
- [ ] Production-like configuration does not accept or expose a plaintext cashier PIN.
- [ ] Startup validation rejects malformed, empty, or unsupported `CASHIER_PIN_HASH` values before the first login request.
- [ ] Production `APP_ENV` forces session cookies to `Secure` even if `SESSION_COOKIE_SECURE` is missing or mis-set.
- [ ] Startup validation fails clearly when session cookie configuration is internally inconsistent, such as an empty cookie name or invalid boolean value.

**Verification:**

- [ ] Tests pass: `go -C backend test ./...`
- [ ] Static checks pass: `go -C backend vet ./...`
- [ ] Manual check: starting without required auth config or with a malformed hash produces a clear error once auth wiring is enabled.

**Dependencies:** None

**Files likely touched:**

- `backend/internal/config/config.go`
- `backend/internal/config/config_test.go`
- `backend/cmd/coffee-pos/main.go`

**Estimated scope:** Medium: 3 files

## Task 2: Implement Auth Domain Rules

**Description:** Add pure domain rules for cashier PIN format and session expiration. These rules should validate exactly 6 numeric digits and compute the session expiry as the earlier of 12 hours from login or the end of the Asia/Jakarta business day.

**Acceptance criteria:**

- [ ] PIN format validation accepts only strings matching exactly 6 ASCII digits.
- [ ] PIN format validation rejects short, long, empty, non-numeric, and whitespace-padded values.
- [ ] Session expiration calculation uses Asia/Jakarta day boundaries and chooses the earlier of end-of-day or 12 hours.
- [ ] Session expiration calculation never returns an already-expired time for logins at `23:59`, exactly midnight, or exactly 12 hours before midnight in Asia/Jakarta.

**Verification:**

- [ ] Tests pass: `go -C backend test ./...`
- [ ] Targeted tests cover PIN validation and expiry edge cases at `23:59`, exactly midnight, and exactly 12 hours before midnight in Asia/Jakarta.

**Dependencies:** Task 1

**Files likely touched:**

- `backend/internal/domain/auth/pin.go`
- `backend/internal/domain/auth/session.go`
- `backend/internal/domain/auth/pin_test.go`
- `backend/internal/domain/auth/session_test.go`

**Estimated scope:** Medium: 4 files

### Checkpoint: Foundation

- [ ] Auth config loads without plaintext PIN support.
- [ ] Domain tests prove PIN format and session-expiry rules.
- [ ] `go -C backend test ./...` and `go -C backend vet ./...` pass.

### Phase 2: Application Use Cases and Security Adapters

## Task 3: Add Auth Application Use Cases and Ports

**Description:** Define the application layer for login, logout, and session checking. The use cases should depend on interfaces for PIN hash verification, session storage, rate limiting, random session ID generation, and time instead of importing HTTP or concrete adapters.

**Acceptance criteria:**

- [ ] Login rejects invalid PIN format before hash verification.
- [ ] Login returns the same invalid-PIN application result for invalid format and wrong PIN.
- [ ] Successful login that is not already blocked resets the failed-attempt window for that client identifier.
- [ ] Login blocked by the rate limiter returns a distinct application result that the HTTP layer maps to `429 Too Many Requests` with `{"error":"too_many_attempts"}`.
- [ ] Successful login creates a server-side session with the computed expiry and always returns a fresh session ID.
- [ ] Concurrent sessions are allowed for the shared cashier PIN, but a successful login with an existing valid session ID invalidates only that previous session and replaces it with the fresh session ID.
- [ ] Logout invalidates the current session ID and is idempotent when the session is missing or already invalidated.
- [ ] Session check returns authenticated only for existing, unexpired sessions.
- [ ] Hash verifier errors, session-store write failures, session ID generation failures, and rate-limiter storage failures return explicit internal-error application results rather than being converted into invalid-PIN failures.

**Verification:**

- [ ] Tests pass: `go -C backend test ./...`
- [ ] Unit tests use fake ports for success, wrong PIN, invalid format, expired session, repeated login, repeated logout, rate-limit rollover, and infrastructure failure paths.

**Dependencies:** Task 2

**Files likely touched:**

- `backend/internal/app/auth/usecases.go`
- `backend/internal/app/auth/ports.go`
- `backend/internal/app/auth/usecases_test.go`

**Estimated scope:** Medium: 3 files

## Task 4: Implement Security Adapters

**Description:** Add concrete backend adapters for slow-hash PIN verification, opaque session ID generation, in-memory server-side sessions, and in-memory rate limiting. Keep the implementations small and behind the application ports created in Task 3.

**Acceptance criteria:**

- [ ] PIN verification compares the supplied PIN with the configured slow hash without logging the PIN or hash.
- [ ] Session IDs are cryptographically random and stored only server-side.
- [ ] Session store expires sessions and supports explicit invalidation.
- [ ] Rate limiter blocks after 5 failed attempts within 5 minutes per client identifier.
- [ ] In-memory session store is safe for concurrent access by Go HTTP handlers.
- [ ] In-memory rate limiter is safe for concurrent access by Go HTTP handlers.
- [ ] Concurrent login attempts from the same client identifier produce deterministic rate-limit behavior and no data races.

**Verification:**

- [ ] Tests pass: `go -C backend test ./...`
- [ ] Race check passes for backend tests that cover auth adapters: `go -C backend test -race ./...`
- [ ] Static checks pass: `go -C backend vet ./...`
- [ ] Targeted adapter tests cover session expiry, logout invalidation, rate-limit reset after the window, sixth failed attempt at `4:59`, and a new failed attempt at `5:01`.

**Dependencies:** Task 3

**Files likely touched:**

- `backend/internal/adapters/security/pin_hash.go`
- `backend/internal/adapters/security/session_store.go`
- `backend/internal/adapters/security/rate_limiter.go`
- `backend/internal/adapters/security/session_id.go`
- `backend/internal/adapters/security/*_test.go`
- `backend/go.mod`

**Estimated scope:** Medium: 5 files

### Checkpoint: Use Cases and Adapters

- [ ] Auth use cases pass with fake ports.
- [ ] Security adapter tests pass.
- [ ] Auth adapter tests pass with `go -C backend test -race ./...`.
- [ ] No PIN or PIN hash appears in logs, frontend files, or API response structs.
- [ ] `go -C backend test ./...` and `go -C backend vet ./...` pass.

### Phase 3: HTTP API and Route Protection

## Task 5: Add Auth HTTP Endpoints

**Description:** Add JSON HTTP handlers for login, logout, and session status. Handlers should translate HTTP requests and cookies into auth use-case calls, set or clear the session cookie, and return stable JSON responses without leaking whether individual PIN digits were correct.

**Acceptance criteria:**

- [ ] `POST /api/auth/login` accepts JSON with a `pin` field and returns success with a session cookie for the correct PIN.
- [ ] Missing `pin`, `pin: null`, numeric `pin`, empty body, invalid JSON, non-6-digit string PINs, and incorrect PINs all return `401 Unauthorized` with `{"error":"invalid_pin"}` and do not create a session.
- [ ] Oversized login bodies are rejected without noisy logs or a `500` response.
- [ ] Unexpected extra JSON fields are ignored and do not change login behavior.
- [ ] Rate-limited login attempts return `429 Too Many Requests` with `{"error":"too_many_attempts"}` and do not perform PIN hash verification.
- [ ] Successful login cookie is `HttpOnly`, `SameSite=Lax` or stricter, path-scoped appropriately, carries `Expires` or `Max-Age` aligned to the server-side session expiry, and is `Secure` when configured for production.
- [ ] Repeated login while already authenticated returns a fresh cookie, invalidates only the previous session from that request, and leaves other browser/device sessions valid.
- [ ] `POST /api/auth/logout` invalidates the current session and clears the cookie using matching cookie attributes.
- [ ] Logout with no current session, a stale cookie, or an already-invalidated session is idempotent and does not return `500`.
- [ ] `GET /api/auth/session` returns `200 OK` with `{ "authenticated": false }` when no valid session exists.
- [ ] `GET /api/auth/session` returns `200 OK` with `{ "authenticated": false }` for tampered, unknown, unparsable, stale, expired, or invalidated session cookies.
- [ ] `GET /api/auth/session` returns `200 OK` with `{ "authenticated": true }` when the current cookie maps to an authenticated session.

**Verification:**

- [ ] Tests pass: `go -C backend test ./...`
- [ ] Integration tests pass: `go -C backend test -tags=integration ./...`
- [ ] HTTP tests cover malformed login bodies, stale cookies, repeated login, repeated logout, oversized bodies, and rate-limit response status/body.
- [ ] Manual check with `curl` can log in, reuse the cookie for session status, and log out.

**Dependencies:** Tasks 3 and 4

**Files likely touched:**

- `backend/internal/adapters/http/router.go`
- `backend/internal/adapters/http/auth_handlers.go`
- `backend/internal/adapters/http/auth_handlers_test.go`
- `backend/internal/adapters/http/router_integration_test.go`
- `backend/cmd/coffee-pos/main.go`

**Estimated scope:** Medium: 5 files

## Task 6: Add Developer PIN Hash Command

**Description:** Add a developer-facing CLI command that generates the configured slow hash for a supplied 6-digit cashier PIN. This should make local and production setup less error-prone without storing secrets in source control or application state.

**Acceptance criteria:**

- [ ] `go -C backend run ./cmd/coffee-pos auth hash-pin 123456` or equivalent prints a bcrypt-compatible hash to stdout.
- [ ] The command validates that the input PIN is exactly 6 numeric digits before hashing.
- [ ] The command does not log, persist, or write the plaintext PIN or generated hash anywhere except stdout.

**Verification:**

- [ ] Tests pass: `go -C backend test ./...`
- [ ] Build succeeds: `go -C backend build ./cmd/coffee-pos`
- [ ] Manual check: generated hash can be supplied as `CASHIER_PIN_HASH` and used for login.

**Dependencies:** Tasks 2 and 4

**Files likely touched:**

- `backend/cmd/coffee-pos/main.go`
- `backend/internal/adapters/security/pin_hash.go`
- `backend/internal/adapters/security/pin_hash_test.go`

**Estimated scope:** Small: 2-3 files

## Task 7: Add Authentication Middleware for Protected POS APIs

**Description:** Add middleware that requires an authenticated server-side session for protected API routes. Since POS routes are not implemented yet, verify the middleware with test-only or minimal protected route wiring rather than implementing menu, order, or report features.

**Acceptance criteria:**

- [ ] `GET /api/health` remains public.
- [ ] `POST /api/auth/login` remains public and rate-limited.
- [ ] Protected API routes return `401 Unauthorized` with a generic JSON response when no valid session cookie is present.
- [ ] Protected API routes call the wrapped handler when a valid, unexpired session exists.
- [ ] Tampered, unknown, unparsable, expired, and invalidated session cookies are rejected without `500` responses.

**Verification:**

- [ ] Tests pass: `go -C backend test ./...`
- [ ] Integration tests pass: `go -C backend test -tags=integration ./...`
- [ ] Manual check: a protected test route or first future POS route returns `401` before login and succeeds after login.

**Dependencies:** Task 5

**Files likely touched:**

- `backend/internal/adapters/http/auth_middleware.go`
- `backend/internal/adapters/http/auth_middleware_test.go`
- `backend/internal/adapters/http/router.go`
- `backend/internal/adapters/http/router_integration_test.go`

**Estimated scope:** Medium: 4 files

### Checkpoint: Backend US-01 Complete

- [ ] Correct 6-digit PIN creates an authenticated server-side session.
- [ ] Incorrect or malformed PIN returns a generic error.
- [ ] Login attempts are rate-limited at 5 failed attempts per 5 minutes per client identifier.
- [ ] Session cookies are `HttpOnly`, production `Secure`, and `SameSite=Lax` or stricter.
- [ ] Session expiry uses Asia/Jakarta and the 12-hour maximum.
- [ ] Browser cookie expiry is aligned with server-side session expiry, and logout clears the cookie with matching attributes.
- [ ] Logout invalidates the session.
- [ ] Repeated login, repeated logout, stale-cookie session checks, malformed login bodies, and oversized login bodies have defined HTTP behavior.
- [ ] In-memory auth adapters pass race checks under concurrent access.
- [ ] Protected POS APIs reject unauthenticated requests.
- [ ] `go -C backend test ./...`, `go -C backend test -race ./...`, `go -C backend test -tags=integration ./...`, and `go -C backend vet ./...` pass.

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| In-memory sessions disappear on restart | Medium | Accept for MVP single-process deployment; keep session storage behind a port so PostgreSQL or Redis can replace it later. |
| In-memory rate limiting is per-process only | Medium | Accept for MVP single backend container; document that multi-instance deployment needs shared rate-limit storage. |
| Slow-hash dependency selection adds external dependency | Low | Use a small established Go package such as `golang.org/x/crypto/bcrypt`; keep it isolated in the security adapter. |
| Client identifier from IP can be inaccurate behind proxies | Medium | Default to `RemoteAddr`; only trust forwarded headers when explicitly configured by deployment. |
| Secure cookies break local HTTP development if always enabled | Medium | Derive cookie `Secure` from environment so production is secure while local development can use plain HTTP. |

## Open Questions

- None. The implementation decisions from this section are captured in Architecture Decisions and task acceptance criteria.

## Parallelization Opportunities

- Tasks 1 and 2 should be sequential because domain expiry depends on config/timezone decisions.
- After Task 3 defines ports, Task 4 adapter implementation and HTTP handler tests can be prepared in parallel as long as the use-case contracts are stable.
- Task 6 can be implemented after the hash adapter exists and does not need to wait for HTTP handlers.
- Task 7 must follow Task 5 because middleware depends on established cookie parsing and session lookup behavior.
