# Implementation Plan: US-01 Frontend Cashier PIN Authentication

**Status:** Ready for code review

## Overview

Implement the frontend portion of `US-01: Sign In With Cashier PIN` from `docs/specs/small-coffee-shop-pos-mvp-spec.md`. This plan covers the cashier PIN login screen, session bootstrapping, authenticated route guarding, logout wiring, frontend auth API clients, frontend validation for cashier ergonomics, and styling aligned with `docs/screen-captures/02.login-pin.png` and the design-system tokens in `docs/design-system/`. The backend auth API is assumed to follow `docs/plan/us-01-backend-cashier-pin-auth-plan.md`.

## Scope

In scope:

- Frontend-only PIN login UI.
- Frontend PIN entry ergonomics for exactly 6 digits.
- Auth API client calls to `GET /api/auth/session`, `POST /api/auth/login`, and `POST /api/auth/logout`.
- Session bootstrap state that decides whether to show login or authenticated POS shell.
- Route guard behavior for current and future POS workflows.
- Generic invalid PIN and rate-limit error messages.
- Logout action that clears frontend auth state after backend logout.
- Unit tests for API clients, PIN entry behavior, session bootstrap, login failures, and route guard behavior.
- Styling aligned with the supplied login mockup and the project design system.

Out of scope:

- Backend auth implementation, PIN hash generation, session storage, cookies, and rate limiting.
- Storing or validating the real cashier PIN in frontend code.
- Menu, order entry, payment, reports, ticket printing, QRIS, or daily summary implementation.
- Individual cashier accounts, roles, PIN rotation UI, password reset, or PIN management.
- A production router dependency unless the implementation deliberately introduces one in a separate reviewed decision.

## Source Inputs

- Spec section: `docs/specs/small-coffee-shop-pos-mvp-spec.md` `#### US-01: Sign In With Cashier PIN`.
- Backend contract: `docs/plan/us-01-backend-cashier-pin-auth-plan.md`.
- UI reference: `docs/screen-captures/02.login-pin.png`.
- Design system: `docs/design-system/README.md`, `docs/design-system/tokens.md`, and `docs/design-system/component-guidelines.md`.
- Current frontend scaffold: `frontend/src/App.tsx`, `frontend/src/lib/health.ts`, `frontend/src/styles.css`, and existing Vitest setup.

## Architecture Decisions

- Keep auth API calls in `frontend/src/lib/auth.ts` so Preact components do not duplicate fetch details or response-shape handling.
- Use relative `/api/...` URLs and `credentials: "same-origin"` so browser-managed HttpOnly session cookies work behind Vite and Caddy.
- Treat `GET /api/auth/session` returning `{ "authenticated": false }` as an expected signed-out state, not an error.
- Map `POST /api/auth/login` `401` with `{"error":"invalid_pin"}` to one generic UI message: `Invalid PIN. Try again.`
- Map `POST /api/auth/login` `429` with `{"error":"too_many_attempts"}` to a rate-limit message that does not reveal PIN details.
- Do only client-side input shaping for usability, such as accepting digits and limiting entry to 6 characters. The frontend must not decide whether the PIN is valid beyond enabling submission when 6 digits are entered.
- Use a visually segmented six-box PIN control backed by a single accessible input, unless implementation proves separate inputs are simpler without hurting paste, keyboard, and screen-reader behavior.
- Avoid storing the PIN outside transient component state. Clear the PIN after failed login, logout, and unmount.
- Use small local state or context for auth bootstrap instead of adding a global state library.
- Until US-03 and reports are implemented, authenticated state may render a minimal protected POS shell that proves access and gives future routes a stable insertion point.

## API Contract Assumptions

Expected backend responses from the backend plan:

- `GET /api/auth/session`
  - `200 OK` with `{ "authenticated": false }` when no valid session exists.
  - `200 OK` with `{ "authenticated": true }` when the current cookie maps to a valid session.
- `POST /api/auth/login`
  - Request JSON: `{ "pin": "123456" }`.
  - Success: `200 OK` or another documented success status with a session cookie. Frontend should depend on `response.ok`, not a specific success body unless the backend exposes one.
  - Invalid PIN or invalid format: `401 Unauthorized` with `{ "error": "invalid_pin" }`.
  - Rate limited: `429 Too Many Requests` with `{ "error": "too_many_attempts" }`.
- `POST /api/auth/logout`
  - Idempotent success for missing, stale, or valid sessions. Frontend should treat any `2xx` response as signed out.

If the implemented backend success body differs from this plan, update the frontend auth client types before implementing UI behavior.

## UI Design Direction

The login screen should match the supplied `02.login-pin.png` direction while remaining responsive and accessible:

- Full viewport soft mint/cream gradient background.
- Centered rounded login panel with subtle glass surface, white border, and soft shadow.
- Coffee cup brand icon above `Coffee POS`.
- Subtitle: `Sign in to continue`.
- Divider line before the PIN form.
- Section title: `Cashier PIN`.
- Helper text: `Enter your 6-digit PIN`.
- Six large square PIN boxes with green borders and masked dot states.
- Full-width green primary `Sign In` button with stable loading state.
- Error alert below the action area using soft red background, red border, alert icon, and text.
- Responsive behavior at `320px`, `768px`, `1024px`, and `1440px`; the panel should shrink without horizontal scrolling.

## Dependency Graph

```text
Backend auth contract and existing frontend fetch conventions
  -> frontend auth response types and API client
      -> auth session bootstrap state
          -> accessible PIN input component
              -> login screen composition and styling
                  -> protected POS shell and route guard behavior
                      -> logout wiring
                          -> Vitest coverage and manual browser verification
```

## Task List

### Phase 1: Auth Client Foundation

## Task 1: Add Frontend Auth API Client

**Description:** Add a typed auth API client for session status, login, and logout. The client should hide fetch details from UI components, use relative URLs, send JSON for login, and normalize backend error codes into frontend-safe result types.

**Acceptance criteria:**

- [ ] `getSession` calls `GET /api/auth/session` and returns authenticated true or false.
- [ ] `loginWithPin` calls `POST /api/auth/login` with JSON `{ pin }` and `credentials: "same-origin"`.
- [ ] `logout` calls `POST /api/auth/logout` with `credentials: "same-origin"`.
- [ ] `401 {"error":"invalid_pin"}` maps to an invalid-PIN result without exposing backend detail in component code.
- [ ] `429 {"error":"too_many_attempts"}` maps to a rate-limited result.
- [ ] Network failures and unexpected response shapes map to an explicit unavailable or unknown-error result.

**Verification:**

- [ ] Tests pass: `npm --prefix frontend test`
- [ ] Type checks pass: `npm --prefix frontend run check`
- [ ] Targeted tests mock `fetch` for authenticated session, unauthenticated session, login success, invalid PIN, rate limit, logout success, and network failure.

**Dependencies:** None

**Files likely touched:**

- `frontend/src/lib/auth.ts`
- `frontend/src/lib/auth.test.ts`

**Estimated scope:** Small: 2 files

## Task 2: Add Auth Bootstrap State

**Description:** Introduce a small auth state layer that checks the current session on app startup and exposes loading, signed-out, signed-in, and unavailable states to the app shell. Keep this local to the app or a small provider; do not add a state-management dependency.

**Acceptance criteria:**

- [ ] App startup calls `getSession` once and displays a loading state while the session check is pending.
- [ ] Authenticated session renders the protected POS shell placeholder.
- [ ] Unauthenticated session renders the login screen.
- [ ] Session-check network failure renders a recoverable error state with a retry action.
- [ ] The auth state can be updated after successful login and logout without a full page refresh.

**Verification:**

- [ ] Tests pass: `npm --prefix frontend test`
- [ ] Type checks pass: `npm --prefix frontend run check`
- [ ] Component tests cover loading, signed-out, signed-in, and session-check failure states.

**Dependencies:** Task 1

**Files likely touched:**

- `frontend/src/App.tsx`
- `frontend/src/features/auth/useAuthSession.ts`
- `frontend/src/features/auth/useAuthSession.test.ts`
- `frontend/src/features/auth/types.ts`

**Estimated scope:** Medium: 3-4 files

### Checkpoint: Auth Foundation

- [ ] Auth API client tests cover the backend response contract.
- [ ] App can distinguish authenticated, unauthenticated, loading, and unavailable states.
- [ ] `npm --prefix frontend test` and `npm --prefix frontend run check` pass.

### Phase 2: Login Experience

## Task 3: Build Accessible PIN Entry Component

**Description:** Build the cashier PIN entry control as a reusable auth component. It should visually render six boxes like the mockup while preserving simple keyboard, paste, screen-reader, and mobile numeric-keypad behavior.

**Acceptance criteria:**

- [ ] Cashier can enter only digits.
- [ ] The component stores at most 6 digits.
- [ ] Pasting a longer or formatted value keeps only the first 6 digits.
- [ ] Backspace, delete, selection replacement, and keyboard navigation behave predictably.
- [ ] The visual boxes show masked filled states and never reveal PIN digits.
- [ ] The control has a visible label or accessible name and exposes error text through `aria-describedby` when present.

**Verification:**

- [ ] Tests pass: `npm --prefix frontend test`
- [ ] Type checks pass: `npm --prefix frontend run check`
- [ ] Component tests cover typing digits, rejecting non-digits, paste, clearing after failure, disabled state, and accessible label/error association.

**Dependencies:** None

**Files likely touched:**

- `frontend/src/features/auth/PinInput.tsx`
- `frontend/src/features/auth/PinInput.test.tsx`

**Estimated scope:** Small: 2 files

## Task 4: Build Login Screen UI

**Description:** Compose the PIN entry component into the full login screen shown in `docs/screen-captures/02.login-pin.png`. The screen should submit to the auth API, show stable loading and error states, and avoid storing the PIN after completion.

**Acceptance criteria:**

- [ ] Login screen matches the main layout of the mockup: centered glass panel, cup icon, title, subtitle, divider, PIN section, six-box PIN control, green sign-in button, and red alert area.
- [ ] Sign In is disabled until 6 digits are entered and while a login request is pending.
- [ ] Pressing Enter submits when exactly 6 digits are present.
- [ ] Successful login updates auth state and renders the protected POS shell without exposing the PIN.
- [ ] Invalid PIN shows `Invalid PIN. Try again.`, clears the PIN, and returns focus to PIN entry.
- [ ] Rate limit shows a clear non-secret message such as `Too many attempts. Try again in a few minutes.`
- [ ] Network failure shows a recoverable backend-unavailable message without clearing a valid typed PIN unless the request was sent.

**Verification:**

- [ ] Tests pass: `npm --prefix frontend test`
- [ ] Type checks pass: `npm --prefix frontend run check`
- [ ] Component tests cover disabled submit, Enter submit, success, invalid PIN, rate limit, network failure, loading label, PIN clearing, and focus recovery.
- [ ] Manual visual check at `320px`, `768px`, `1024px`, and `1440px`.

**Dependencies:** Tasks 1, 2, and 3

**Files likely touched:**

- `frontend/src/features/auth/LoginScreen.tsx`
- `frontend/src/features/auth/LoginScreen.test.tsx`
- `frontend/src/App.tsx`
- `frontend/src/styles.css`

**Estimated scope:** Medium: 4 files

### Checkpoint: Login Flow

- [ ] Unauthenticated startup renders the styled PIN login screen.
- [ ] Correct 6-digit PIN reaches authenticated frontend state when backend login succeeds.
- [ ] Incorrect PIN and rate-limit responses show generic errors.
- [ ] PIN digits are never displayed in clear text.
- [ ] `npm --prefix frontend test`, `npm --prefix frontend run check`, and `npm --prefix frontend run build` pass.

### Phase 3: Protected Shell and Logout

## Task 5: Add Protected POS Shell Placeholder

**Description:** Replace the scaffold-only health page with a minimal authenticated POS shell placeholder that proves the route guard works while keeping US-03 order entry and reports out of scope. This shell should provide clear insertion points for future Cashier and Daily Summary screens.

**Acceptance criteria:**

- [ ] Authenticated users see a protected shell instead of the PIN screen.
- [ ] The protected shell includes `Coffee POS`, a current access/status label, and placeholder navigation entries for `New Order` and `Daily Summary` without implementing their workflows.
- [ ] Unauthenticated users never see protected shell content after session bootstrap resolves.
- [ ] Session-check failure does not accidentally render protected content.
- [ ] Existing backend health display is either removed from the primary UX or moved into a small developer/status area that does not block POS access.

**Verification:**

- [ ] Tests pass: `npm --prefix frontend test`
- [ ] Type checks pass: `npm --prefix frontend run check`
- [ ] Component tests prove protected content appears only when the session is authenticated.

**Dependencies:** Task 2

**Files likely touched:**

- `frontend/src/features/pos/ProtectedPosShell.tsx`
- `frontend/src/features/pos/ProtectedPosShell.test.tsx`
- `frontend/src/App.tsx`
- `frontend/src/styles.css`

**Estimated scope:** Medium: 4 files

## Task 6: Add Logout Flow

**Description:** Add a logout action to the protected shell that calls the backend logout endpoint, clears frontend auth state, and returns the browser to the PIN login screen.

**Acceptance criteria:**

- [ ] Logout button is visible only in authenticated state.
- [ ] Clicking logout calls `POST /api/auth/logout`.
- [ ] Successful logout returns to the PIN login screen.
- [ ] Repeated logout or stale-session logout does not leave the app stuck in authenticated state.
- [ ] Logout network failure shows a recoverable error and does not silently claim the cashier is signed out unless the frontend intentionally falls back to local sign-out.

**Verification:**

- [ ] Tests pass: `npm --prefix frontend test`
- [ ] Type checks pass: `npm --prefix frontend run check`
- [ ] Component tests cover logout success, logout failure, and signed-out render after logout.

**Dependencies:** Tasks 1, 2, and 5

**Files likely touched:**

- `frontend/src/features/pos/ProtectedPosShell.tsx`
- `frontend/src/features/pos/ProtectedPosShell.test.tsx`
- `frontend/src/App.tsx`
- `frontend/src/lib/auth.test.ts`

**Estimated scope:** Medium: 3-4 files

### Checkpoint: Protected Access

- [ ] Unauthenticated sessions render login only.
- [ ] Authenticated sessions render protected shell only.
- [ ] Logout returns to login.
- [ ] Future POS screens have a clear guarded shell to plug into.
- [ ] `npm --prefix frontend test`, `npm --prefix frontend run check`, and `npm --prefix frontend run build` pass.

### Phase 4: Integration Polish and Browser Coverage

## Task 7: Add Auth-Focused E2E Smoke Test

**Description:** Add a Playwright smoke test for the frontend auth flow once backend auth is runnable locally. Keep the test narrow: unauthenticated redirect/login screen, successful login, protected shell access, and logout.

**Acceptance criteria:**

- [ ] E2E test starts from an unauthenticated browser context and sees the PIN login screen.
- [ ] Valid login reaches protected POS shell.
- [ ] Logout returns to the login screen.
- [ ] Invalid PIN keeps the browser on login and shows the generic invalid-PIN message.
- [ ] Test setup does not embed the real production PIN or PIN hash; local test credentials are generated/configured for test runtime only.

**Verification:**

- [ ] E2E passes: `npm run test:e2e`
- [ ] Manual check: run backend with local `CASHIER_PIN_HASH`, run frontend dev server, complete login and logout in the browser.

**Dependencies:** Tasks 1-6 and backend US-01 implementation

**Files likely touched:**

- `tests/e2e/auth-login.spec.ts`
- `package.json`
- Playwright test support files if already present or needed

**Estimated scope:** Small to Medium: 2-3 files

## Task 8: Final Accessibility and Responsive Pass

**Description:** Verify the login screen and protected shell against the design-system checklist and frontend accessibility expectations. Fix layout, focus, status announcements, and text overflow issues discovered during manual browser testing.

**Acceptance criteria:**

- [ ] Login panel has no horizontal overflow at `320px`.
- [ ] Keyboard-only users can complete PIN entry, submit, recover from errors, and logout.
- [ ] Focus is visible on PIN entry, Sign In, retry, and logout controls.
- [ ] Error and loading messages are announced through appropriate `role="alert"` or `role="status"` regions.
- [ ] Login UI uses design-system color, radius, spacing, shadow, and gradient tokens rather than unrelated hard-coded styles.
- [ ] Reduced-motion preference does not leave essential state changes hidden.

**Verification:**

- [ ] Tests pass: `npm --prefix frontend test`
- [ ] Type checks pass: `npm --prefix frontend run check`
- [ ] Build succeeds: `npm --prefix frontend run build`
- [ ] Manual browser check at `320px`, `768px`, `1024px`, and `1440px`.
- [ ] Manual keyboard check covers login success, login error recovery, and logout.

**Dependencies:** Tasks 1-6

**Files likely touched:**

- `frontend/src/styles.css`
- `frontend/src/features/auth/LoginScreen.tsx`
- `frontend/src/features/auth/PinInput.tsx`
- `frontend/src/features/pos/ProtectedPosShell.tsx`

**Estimated scope:** Medium: 4 files

## Task 9: Final Chrome Visual Verification Against UI Design

**Description:** Capture the implemented login screen in Chrome and compare it directly against `docs/screen-captures/02.login-pin.png`. This is a final design-quality gate, not a functional test. The implementation should be visually similar to the supplied design in layout, spacing, color treatment, hierarchy, and visible states before US-01 frontend work is considered complete.

**Acceptance criteria:**

- [ ] A fresh Chrome screenshot of the implemented signed-out login screen is captured after the production or dev build is running.
- [ ] The implementation screenshot is compared against `docs/screen-captures/02.login-pin.png` at a matching desktop viewport, preferably `1512x1080` or the closest practical Chrome viewport.
- [ ] The main visual structure matches the design: centered rounded glass panel, soft gradient background, cup icon, `Coffee POS` title, subtitle, divider, PIN section, six PIN boxes, green Sign In button, and red error alert state.
- [ ] Differences are limited to acceptable implementation constraints such as exact font rendering, icon stroke details, and minor browser antialiasing.
- [ ] Any obvious mismatch in panel sizing, alignment, radius, shadow, color palette, text hierarchy, PIN box spacing, button shape, or error-alert styling is fixed before marking the task complete.
- [ ] A final screenshot is saved under `docs/screen-captures/` with a descriptive name such as `02.login-pin-implementation.png` for review history.

**Verification:**

- [ ] Chrome screenshot captured using Playwright, Chrome DevTools, or an equivalent browser screenshot tool.
- [ ] Manual visual comparison completed against `docs/screen-captures/02.login-pin.png`.
- [ ] If visual fixes are made, rerun `npm --prefix frontend test`, `npm --prefix frontend run check`, and `npm --prefix frontend run build`.
- [ ] Manual check confirms the invalid-PIN error state is visible in the screenshot, since the reference design includes that state.

**Dependencies:** Tasks 1-8

**Files likely touched:**

- `frontend/src/styles.css`
- `frontend/src/features/auth/LoginScreen.tsx`
- `frontend/src/features/auth/PinInput.tsx`
- `docs/screen-captures/02.login-pin-implementation.png`

**Estimated scope:** Small to Medium: 1-4 files plus one screen capture

### Checkpoint: Frontend US-01 Complete

- [ ] Cashier can enter exactly 6 digits through the frontend PIN UI.
- [ ] Correct PIN creates an authenticated browser session through the backend API.
- [ ] Incorrect PIN shows a generic error with no digit-level feedback.
- [ ] Rate-limited login shows a clear frontend error.
- [ ] Authenticated session can access the protected POS shell.
- [ ] Unauthenticated browser session sees the PIN login screen, not protected content.
- [ ] Cashier can log out and return to the PIN login screen.
- [ ] Frontend never contains the real PIN or PIN hash.
- [ ] Login UI matches the supplied design direction and design-system tokens.
- [ ] Chrome screenshot comparison confirms the implemented login screen is visually similar to `docs/screen-captures/02.login-pin.png`.
- [ ] `npm --prefix frontend test`, `npm --prefix frontend run check`, and `npm --prefix frontend run build` pass.
- [ ] When backend auth is available, `npm run test:e2e` passes for the auth smoke flow.

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Backend success response body differs from frontend assumptions | Medium | Keep the auth client tolerant of any `2xx` success and update types once backend implementation is confirmed. |
| PIN UI becomes six disconnected inputs with poor paste or screen-reader behavior | Medium | Prefer one semantic input with six visual boxes; test paste, keyboard, and accessible names explicitly. |
| Frontend accidentally embeds test PIN values in production code | High | Keep PIN only in tests or local runtime setup; never put a real PIN or hash in source files, static assets, logs, or UI copy. |
| Session bootstrap flashes protected content before auth resolves | High | Render a neutral loading state until `GET /api/auth/session` resolves; never default to signed-in. |
| Secure HttpOnly cookie cannot be inspected by frontend | Low | Treat cookies as backend/browser-managed and rely only on session endpoint results. |
| Login screen looks good on desktop but breaks on small cashier displays | Medium | Verify at `320px` and avoid fixed panel widths without responsive `max-width` and padding. |
| E2E depends on backend auth runtime config | Medium | Gate E2E auth smoke until backend US-01 is implemented and test setup can generate a local hash safely. |

## Open Questions

- Should authenticated US-01 render a minimal protected POS shell placeholder, or should it immediately route to a future `/cashier` path that remains empty until US-03? This plan recommends the protected shell placeholder so route guarding can be verified without implementing order entry.
- Should the login button remain disabled until 6 digits are present, or allow submission and rely entirely on backend validation? This plan recommends disabled-until-6 for cashier ergonomics while still treating backend validation as authoritative.

## Parallelization Opportunities

- Task 1 can be implemented independently of Task 3.
- Task 3 can be built and tested while Task 2 defines auth bootstrap state.
- Task 5 can start after Task 2 and does not need Task 4 internals.
- Task 7 should wait for Tasks 1-6 and the backend US-01 implementation.
- Task 8 should be the final pass after the login and protected shell UI are in place.
- Task 9 must be last because it verifies the final rendered UI after all styling and accessibility fixes.
