# Plan Review: US-01 Backend Cashier PIN Authentication

**Date:** 2026-06-29
**Reviewer:** Codex
**Document:** `docs/plan/us-01-backend-cashier-pin-auth-plan.md`

---

## Required Changes

### 1. Invalid auth configuration is not fully covered by acceptance criteria

Task 1 only requires support for `CASHIER_PIN_HASH` and rejection of plaintext PIN configuration, but it does not require startup-time validation that the configured hash is actually usable or that production cookie settings cannot be weakened by config drift (lines 70-78). As written, the service could start with a malformed hash or insecure production cookie config and fail only on the first login attempt.

Add acceptance criteria for:

- rejecting malformed or unsupported `CASHIER_PIN_HASH` values at startup
- forcing production sessions to be `Secure` even if `SESSION_COOKIE_SECURE` is mis-set
- failing clearly when cookie config is internally inconsistent

### 2. Session lifecycle policy is underspecified

The plan requires successful login to create a session and logout to invalidate the current session, but it never states what happens when the shared PIN is used from multiple browsers/devices or when an already-authenticated client logs in again (lines 132-134, 196-199). That leaves a material behavior gap for a shared-PIN system.

Add acceptance criteria for:

- whether concurrent sessions are allowed for the shared cashier PIN
- whether a new login invalidates prior sessions or only creates another session
- whether successful login always issues a fresh session ID rather than reusing any existing cookie value

### 3. Rate-limit behavior after a successful login is still ambiguous

Task 3 says a successful login "resets or does not increment" failed-attempt state (line 132). Those are different behaviors with different operator impact. One implementation could clear the lockout window; another could leave the caller blocked even after a correct PIN.

This needs a single rule in the acceptance criteria:

- successful login resets the failure window, or
- successful login leaves the current failure window intact

The plan should also state the expected HTTP status/body for a rate-limited login attempt so handler and frontend tests converge on one contract.

### 4. The HTTP contract is incomplete for malformed bodies and bad cookies

Task 5 covers "malformed" login attempts and Task 7 covers missing/expired sessions, but the plan does not enumerate the concrete cases the handlers must treat as ordinary auth failures instead of `500` errors (lines 196-201, 253-257). That leaves too much room for accidental handler behavior.

Add acceptance criteria for:

- missing `pin` field
- `pin: null`, numeric `pin`, empty body, and invalid JSON
- tampered, unknown, or unparsable session cookies
- logout with no current session or with an already-invalidated session
- session status on a stale cookie still returning `200` with `{ "authenticated": false }`

### 5. The in-memory adapters need explicit concurrency requirements

The plan chooses in-memory session storage and rate limiting for a Go HTTP server (lines 37, 153-160), but no acceptance criteria require those adapters to be safe under concurrent access. That is a real implementation risk, not an optimization detail.

Add acceptance criteria or verification for:

- race-free concurrent access to session storage
- race-free concurrent access to rate-limit counters
- concurrent login attempts from the same client identifier behaving deterministically

---

## Missing Edge Cases

- Login exactly near the boundary conditions: `23:59` Asia/Jakarta, exactly midnight, and exactly 12 hours before midnight. The expiry rule should be proven not to create an already-expired session.
- Cookie lifetime alignment: if the server-side session expires earlier than browser shutdown, does the cookie also carry `Expires`/`Max-Age`, and does logout clear it with matching attributes so the browser actually removes it?
- Double actions: repeated logout, repeated login while already authenticated, and repeated session checks after the session has been invalidated.
- Infrastructure failure paths: hash verifier error, session-store write failure, random session ID generation failure, and rate-limiter storage failure should have defined non-auth failure behavior.
- Request-shape abuse: oversized login bodies and unexpected extra fields should not produce divergent auth behavior or noisy logs.
- Rate-limit window rollover: the sixth failed attempt at `4:59` versus `5:01`, and mixed success/failure sequences from the same client identifier.

---

## Verdict

**Ready to implement**. The requested gaps around config validation, session lifecycle, rate-limit behavior, malformed input handling, cookie behavior, infrastructure failure paths, and concurrent in-memory adapter behavior have been addressed in `docs/plan/us-01-backend-cashier-pin-auth-plan.md`.
