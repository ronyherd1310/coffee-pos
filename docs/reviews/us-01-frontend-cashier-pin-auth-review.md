# Code Review: US-01 Frontend Cashier PIN Authentication Implementation

**Date:** 2026-06-30
**Reviewer:** opencode
**Document:** `docs/plan/us-01-frontend-cashier-pin-auth-plan.md`
**Status:** No Critical Findings

---

## Verification Commands

```bash
npm --prefix frontend test
npm --prefix frontend run check
npm --prefix frontend run build
npm run test:e2e
```

All commands pass with zero failures. 32 unit tests pass across 6 test files. E2E Playwright test passes in Chromium.

---

## Findings

### Severity: Low

#### 1. E2E test mocks the entire backend instead of exercising a real auth stack

**File:** `tests/e2e/auth-login.spec.ts:6-34`

The Playwright E2E test stubs all three API routes (`/api/auth/session`, `/api/auth/login`, `/api/auth/logout`) with `page.route()`, simulating the backend entirely in the browser. This means the E2E test verifies the frontend routing flow but does not exercise real backend auth, cookie handling, or network behavior. The plan's Task 7 acceptance criteria state "Test setup does not embed the real production PIN or PIN hash" and the plan acknowledges "E2E depends on backend auth runtime config" as a risk. The current approach is acceptable as a smoke gate, but it is closer to an integration-style component test than a true E2E test against the backend.

**Recommendation:** Acceptable for MVP with mocked backend. Document that the E2E test is a frontend-only flow test and that a full-stack auth smoke test should be added once backend US-01 is deployed locally for integration testing.

#### 2. Login screen renders an "or" divider that has no second action

**File:** `frontend/src/features/auth/LoginScreen.tsx:92-96`, `frontend/src/styles.css:304-322`

The login screen renders a styled divider with the text "or" between the Sign In button and the error alert. There is no secondary action (e.g., guest mode, different login method) below this divider. The plan's mockup reference (`02.login-pin.png`) does not mention an "or" divider, and the design system component guidelines do not reference this pattern. This appears to be a leftover from a template or earlier design iteration.

**Recommendation:** Remove the "or" divider from `LoginScreen.tsx` and its associated CSS rules (`.login-or`, `.login-or span`, `.login-or strong`) unless a second auth method is planned for a future slice. If the mockup does include this element, confirm against `docs/screen-captures/02.login-pin.png`.

#### 3. `useAuthSession` cleanup does not cancel in-flight session requests

**File:** `frontend/src/features/auth/useAuthSession.ts:9-33`

The `refresh` callback sets `isCurrent = false` in its cleanup function, which prevents state updates after unmount. However, the `getSession()` promise continues to execute even after `isCurrent` is set to `false`. This is a minor resource concern: if the component unmounts while the network request is in flight, the fetch completes uselessly. The `isCurrent` guard correctly prevents stale state updates, so this is not a correctness bug.

**Recommendation:** Acceptable for MVP. If the session check is slow or unreliable, consider using an `AbortController` to cancel the fetch when the cleanup runs. This is an optimization, not a fix.

#### 4. `health.ts` is still in the codebase but unused in the app flow

**File:** `frontend/src/lib/health.ts`

The original `health.ts` module is still present in the repository. It is no longer imported by `App.tsx` or any other active code. The plan's Task 5 acceptance criteria state "Existing backend health display is either removed from the primary UX or moved into a small developer/status area." The current `App.tsx` no longer renders the health check, so the UX requirement is met, but the dead module remains.

**Recommendation:** Acceptable for MVP. Remove `health.ts` and `health.test.ts` in a follow-up cleanup if the health endpoint is not needed for development/debugging.

---

## Positive Observations

### Architecture and Design

- **Auth API client is well-isolated:** `frontend/src/lib/auth.ts` centralizes all fetch details, response parsing, and error normalization. Components never touch `fetch` directly for auth operations. This matches the plan's architecture decision exactly.
- **Typed discriminated unions:** `SessionResult`, `LoginResult`, and `LogoutResult` use string-literal status fields, making exhaustive switch/if-checking natural and type-safe.
- **Session bootstrap is clean:** `useAuthSession` manages loading/authenticated/unauthenticated/unavailable states with a single hook. The `markAuthenticated` and `markUnauthenticated` callbacks allow child components to update auth state without prop drilling or context.
- **Route guard logic is explicit in `App.tsx`:** The four states (loading, unavailable, authenticated, unauthenticated) are handled as sequential if-returns, making the render path unambiguous.

### PIN Input Component

- **Single hidden input with visual boxes pattern:** The `PinInput` component uses one `<input type="password">` overlaid on six visual `<span>` boxes. This preserves native keyboard, paste, autocomplete, and screen-reader behavior while achieving the segmented visual design. The `aria-hidden="true"` on the boxes container correctly prevents duplicate announcements.
- **Input normalization is defensive:** `normalizePin` strips non-digits and slices to 6 characters. The component accepts this through `onInput`, so paste, drag-drop, and programmatic value changes all pass through normalization.
- **Accessibility is correct:** The input has `aria-describedby` pointing to the error element, `aria-invalid` is set dynamically, and `autoComplete="current-password"` is appropriate for a PIN field.

### Login Screen

- **Error focus recovery:** After a failed login, the PIN is cleared and focus is returned to the PIN input via `pinInputRef.current?.focus()`. The `queueMicrotask` wrapper ensures DOM updates complete before focus moves. The test (`LoginScreen.test.tsx:60-81`) verifies both the error message and focus state.
- **Submit gating is correct:** `canSubmit` requires exactly 6 digits AND not submitting. The button is disabled via this condition, and the Enter key handler also checks `canSubmit`.
- **Error messages are generic:** Invalid PIN, rate limit, and network failure messages do not reveal backend internals, matching the plan's security requirements.

### Protected POS Shell

- **Logout handles failure gracefully:** If `POST /api/auth/logout` fails, the shell shows a recoverable error and does not call `onSignedOut`. The user remains authenticated and can retry. This matches the plan's acceptance criteria.
- **Navigation placeholders are clear:** "New Order" and "Daily Summary" links provide obvious insertion points for US-03 and later slices.

### CSS and Design System

- **Design tokens are used consistently:** Colors (`--color-accent`, `--color-danger`), radii (`--radius-shell`, `--radius-pill`), spacing (`--space-*`), shadows (`--shadow-shell`, `--shadow-focus`), and gradients (`--gradient-app`) are all referenced from `:root`. No hard-coded color values appear in component styles.
- **Responsive behavior is handled:** The login panel uses `min(100%, 720px)` for fluid width, and media queries at `768px` adjust padding, gaps, and font sizes. The `320px` minimum is set on `html` and `body`.
- **Reduced motion is respected:** The `@media (prefers-reduced-motion: reduce)` block sets `transition-duration: 0.01ms`, ensuring animations do not hide state changes.
- **Focus-visible styling is present:** Both `.button:focus-visible` and `.pos-shell__nav a:focus-visible` use `box-shadow: var(--shadow-focus)` for visible keyboard focus indicators.

### Test Coverage

- **Unit tests are comprehensive:** 32 tests cover the auth API client (10), PIN input (6), login screen (7), protected shell (3), app bootstrap (5), and health check (1). All plan Task acceptance criteria have corresponding test cases.
- **E2E smoke test covers the full flow:** The Playwright test walks through unauthenticated → invalid PIN → error display → valid login → protected shell → logout → login screen, verifying the critical user journey.
- **Type checks and build pass:** `tsc -b --noEmit` and `vite build` both succeed with no errors.

---

## Critical Findings Resolution

No findings in this review are marked Critical. All findings are Low-severity observations about test strategy, dead code, and minor design artifacts. No blocking issues were found.

**Fix summary:** No code changes were needed because there were no Critical findings to reproduce or fix. The review status was updated from "Ready to be checked" to "No Critical Findings" per the requested workflow.

**Verification:** No behavior tests were required for this resolution because no source or test files changed. Review-document verification confirmed that the findings list contains only Low severity entries and no Critical entries.

---

## Summary

The implementation faithfully follows the plan across all nine tasks. The auth API client, PIN input component, login screen, session bootstrap, protected shell, logout flow, E2E test, styling, and accessibility are all implemented and verified. The four Low-severity findings are: (1) the E2E test mocks the backend rather than testing against a real server, (2) an orphaned "or" divider in the login screen, (3) no fetch cancellation in the session hook cleanup, and (4) the unused `health.ts` module. None of these block the US-01 frontend deliverable. All 32 unit tests and 1 E2E test pass. Type checks and production build succeed.

**Verdict:** Approved with minor findings. No implementation changes required for the current scope.
