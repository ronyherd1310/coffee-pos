# Implementation Plan: Coffee POS Scaffold Foundation

## Overview

Create the initial project scaffold for the Small Coffee Shop POS MVP without implementing POS application logic. The first working slice is a browser landing page that calls a backend health endpoint through the frontend origin, proving the frontend, backend, tests, and Podman Compose wiring work end to end.

## Scope

In scope:

- Backend Go module and HTTP server scaffold.
- Public backend health endpoint only.
- Frontend Preact + Vite + TypeScript scaffold.
- Simple landing page only.
- Frontend-to-backend health check through `/api/health`.
- Minimal unit, integration, and Playwright end-to-end smoke tests.
- Production-style backend and frontend container builds.
- Podman Compose file with frontend, backend, and PostgreSQL services.
- Developer commands documented in the repo.

Out of scope:

- PIN authentication, sessions, rate limiting, CSRF, or route guards.
- Menu seeding, sqlc queries, migrations with application schema, or repositories.
- Orders, payments, queue numbers, tickets, reports, QRIS flow, or printing.
- Any POS UI beyond a landing page and backend status display.
- Any production secret values committed to source control.

## Architecture Decisions

- Use the spec baseline stack: Go `net/http` for backend, Preact + Vite + TypeScript for frontend, Caddy for static frontend serving, PostgreSQL in Compose, and Podman for image builds/runs.
- Expose `GET /api/health` from the backend with a stable JSON response such as `{ "status": "ok", "service": "coffee-pos-backend" }`.
- Let the frontend call `/api/health` relative to its own origin so the same code works behind Caddy in Compose and with Vite dev proxy locally.
- Include PostgreSQL in Compose as foundation infrastructure, but do not make health check success depend on database connectivity until database-backed features exist.
- Keep scaffold directories aligned with the target structure from `docs/specs/small-coffee-shop-pos-mvp-spec.md`, but create only the files needed for this foundation slice.
- Use the frontend/Caddy service as the only browser-facing Compose service on host port `8080`; keep the backend API on the Compose network without publishing it to the host by default.
- Keep Playwright E2E tooling in a root `package.json` and avoid npm workspaces until the repo has a concrete need for shared package orchestration.
- Treat `/api/health` as process liveness only in this scaffold. Add a separate readiness endpoint later if database readiness becomes necessary.

## Dependency Graph

```text
Backend Go module
  -> backend health handler
  -> backend unit/integration tests
  -> backend Containerfile

Frontend Vite/Preact module
  -> API client for /api/health
  -> landing page status display
  -> frontend unit/build checks
  -> frontend Containerfile + Caddyfile

Backend container + frontend container
  -> compose.yaml with PostgreSQL
  -> Playwright smoke test against composed app
```

## Task List

### Phase 1: Backend Foundation

## Task 1: Scaffold Backend Go Service

**Description:** Create the backend module, command entrypoint, and small HTTP server structure needed to run a Go API process. Keep package boundaries compatible with the future hexagonal architecture, but only wire the health endpoint.

**Acceptance criteria:**

- [ ] `go -C backend run ./cmd/coffee-pos serve` starts an HTTP server.
- [ ] `GET /api/health` returns `200 OK` with JSON health data.
- [ ] Server port is configurable with an environment variable and has a sensible local default.

**Verification:**

- [ ] Command works: `go -C backend run ./cmd/coffee-pos serve`
- [ ] Manual check: `curl http://localhost:8080/api/health`
- [ ] Build succeeds: `go -C backend build ./cmd/coffee-pos`

**Dependencies:** None

**Files likely touched:**

- `backend/go.mod`
- `backend/cmd/coffee-pos/main.go`
- `backend/internal/adapters/http/router.go`
- `backend/internal/config/config.go`

**Estimated scope:** Medium: 3-5 files

## Task 2: Add Backend Smoke Tests

**Description:** Add minimal backend tests proving the health handler and server routing work. These tests are scaffold checks only and must not introduce POS domain behavior.

**Acceptance criteria:**

- [ ] Unit test verifies the health handler returns `200 OK`.
- [ ] Unit test verifies the health response is valid JSON with `status: ok`.
- [ ] `go -C backend test ./...` runs successfully without requiring containers.

**Verification:**

- [ ] Tests pass: `go -C backend test ./...`
- [ ] Static checks pass: `go -C backend vet ./...`

**Dependencies:** Task 1

**Files likely touched:**

- `backend/internal/adapters/http/router_test.go`

**Estimated scope:** Small: 1 file

## Task 3: Add Backend Integration Smoke Test

**Description:** Add a build-tagged backend integration smoke test that exercises the router through an HTTP test server. This proves the backend can be tested at the HTTP boundary without introducing database schema, Testcontainers, or POS behavior yet.

**Acceptance criteria:**

- [ ] Integration test starts an HTTP test server using the backend router.
- [ ] Integration test calls `GET /api/health` over HTTP and verifies `200 OK`.
- [ ] Integration test is runnable separately with an `integration` build tag.

**Verification:**

- [ ] Integration test passes: `go -C backend test -tags=integration ./...`
- [ ] Fast tests still pass without the tag: `go -C backend test ./...`

**Dependencies:** Task 1

**Files likely touched:**

- `backend/internal/adapters/http/router_integration_test.go`

**Estimated scope:** Small: 1 file

### Checkpoint: Backend

- [ ] Backend starts locally.
- [ ] Backend health endpoint responds.
- [ ] Backend unit tests and vet pass.
- [ ] Backend integration smoke test passes.

### Phase 2: Frontend Foundation

## Task 4: Scaffold Frontend Toolchain

**Description:** Create the Preact + Vite + TypeScript frontend project skeleton and scripts without building POS screens. This task establishes install, dev, check, and build commands.

**Acceptance criteria:**

- [ ] `npm --prefix frontend run dev` starts the Vite dev server.
- [ ] `npm --prefix frontend run check` runs TypeScript checks.
- [ ] `npm --prefix frontend run build` produces static assets.

**Verification:**

- [ ] Dev server starts: `npm --prefix frontend run dev`
- [ ] Type checks pass: `npm --prefix frontend run check`
- [ ] Build succeeds: `npm --prefix frontend run build`

**Dependencies:** None

**Files likely touched:**

- `frontend/package.json`
- `frontend/tsconfig.json`
- `frontend/vite.config.ts`
- `frontend/index.html`

**Estimated scope:** Medium: 4 files

## Task 5: Add Landing Page Health Flow

**Description:** Add the simple landing page that displays the product name and backend health status. This is the first vertical slice proving the frontend can call the backend without implementing application workflows.

**Acceptance criteria:**

- [ ] Landing page renders a clear Coffee POS title.
- [ ] Landing page requests `/api/health` and displays backend status.
- [ ] No POS workflow UI or placeholder business screens are added.

**Verification:**

- [ ] Build succeeds: `npm --prefix frontend run build`
- [ ] Manual check: browser shows landing page and backend status.

**Dependencies:** Tasks 1 and 4

**Files likely touched:**

- `frontend/src/main.tsx`
- `frontend/src/App.tsx`
- `frontend/src/lib/health.ts`
- `frontend/src/styles.css`

**Estimated scope:** Medium: 4 files

## Task 6: Add Frontend Smoke Tests

**Description:** Add minimal frontend tests and checks that prove TypeScript, rendering, and the health API client are wired. Keep tests focused on scaffold behavior.

**Acceptance criteria:**

- [ ] Unit test renders the landing page title.
- [ ] Unit test or API-client test verifies successful health response handling with a mocked fetch.
- [ ] `npm --prefix frontend run check` passes.

**Verification:**

- [ ] Tests pass: `npm --prefix frontend test`
- [ ] Type/build checks pass: `npm --prefix frontend run check`
- [ ] Build succeeds: `npm --prefix frontend run build`

**Dependencies:** Task 5

**Files likely touched:**

- `frontend/package.json`
- `frontend/src/App.test.tsx`
- `frontend/src/lib/health.test.ts`
- `frontend/vitest.config.ts`
- `frontend/src/test/setup.ts`

**Estimated scope:** Medium: 3-5 files

### Checkpoint: Local App

- [ ] Backend tests pass.
- [ ] Frontend tests and checks pass.
- [ ] Frontend build succeeds.
- [ ] Vite dev frontend can call backend health through local proxy.

### Phase 3: Container Foundation

## Task 7: Add Backend Container Build

**Description:** Add a backend `Containerfile` that builds a small production image running the compiled Go API binary. Do not include test tools or frontend assets in the backend runtime image.

**Acceptance criteria:**

- [ ] `podman build -f backend/Containerfile -t coffee-pos-backend:dev backend` succeeds.
- [ ] Container runs the backend API process.
- [ ] Container exposes the configured API port.

**Verification:**

- [ ] Image builds: `podman build -f backend/Containerfile -t coffee-pos-backend:dev backend`
- [ ] Manual check: running container responds at `/api/health`.

**Dependencies:** Task 1

**Files likely touched:**

- `backend/Containerfile`
- `backend/.containerignore`

**Estimated scope:** Small: 1-2 files

## Task 8: Add Frontend Static Container Build

**Description:** Add a frontend `Containerfile` and `Caddyfile` that build the Vite static assets and serve them with Caddy. Configure Caddy to reverse-proxy `/api/*` to the backend service.

**Acceptance criteria:**

- [ ] `podman build -f frontend/Containerfile -t coffee-pos-frontend:dev frontend` succeeds.
- [ ] Runtime image serves only static files through Caddy.
- [ ] Caddy proxies `/api/health` to the backend service.

**Verification:**

- [ ] Image builds: `podman build -f frontend/Containerfile -t coffee-pos-frontend:dev frontend`
- [ ] Manual check: frontend container serves the landing page.

**Dependencies:** Tasks 5 and 7

**Files likely touched:**

- `frontend/Containerfile`
- `frontend/Caddyfile`
- `frontend/.containerignore`

**Estimated scope:** Small: 3 files

## Task 9: Add Podman Compose Stack

**Description:** Add a Compose file for local production-style execution with frontend, backend, and PostgreSQL services. Compose should prove service networking and startup health, but the backend scaffold should not implement database-backed behavior yet.

**Acceptance criteria:**

- [ ] `podman compose up --build` starts frontend, backend, and PostgreSQL services.
- [ ] Browser-facing frontend origin serves the landing page.
- [ ] Frontend origin can reach backend health through `/api/health`.
- [ ] PostgreSQL service is present and healthy for future database work.

**Verification:**

- [ ] Stack starts: `podman compose up --build`
- [ ] Manual check: `curl http://localhost:8080/api/health` through frontend/Caddy origin, or the configured frontend port.
- [ ] Manual check: landing page displays backend status from the composed stack.

**Dependencies:** Tasks 7 and 8

**Files likely touched:**

- `compose.yaml`
- `.env.example`

**Estimated scope:** Small: 1-2 files

### Checkpoint: Containers

- [ ] Backend image builds.
- [ ] Frontend image builds.
- [ ] Podman Compose stack starts.
- [ ] Frontend-to-backend health check works through Caddy.

### Phase 4: End-to-End Validation

## Task 10: Add Playwright Smoke Test

**Description:** Add a minimal Playwright test that opens the landing page and verifies the backend health status is visible. This is the first browser workflow test and should run against the local app or composed stack.

**Acceptance criteria:**

- [ ] Playwright can open the landing page.
- [ ] Test verifies Coffee POS title is visible.
- [ ] Test verifies backend health status reaches the page.
- [ ] Test does not depend on auth, orders, payments, database schema, or seeded data.

**Verification:**

- [ ] E2E test passes: `npx playwright test`
- [ ] Test can be run after starting the app locally or through Compose.

**Dependencies:** Tasks 5 and 9

**Files likely touched:**

- `package.json`
- `playwright.config.ts`
- `tests/e2e/landing-health.spec.ts`

**Estimated scope:** Small: 3 files

## Task 11: Document Scaffold Commands

**Description:** Add a short developer README section or scaffold notes file listing the commands required to install dependencies, run tests, build images, and start the Compose stack.

**Acceptance criteria:**

- [ ] Commands from the spec that are now available are documented.
- [ ] Commands clearly distinguish local dev, tests, container builds, and Compose.
- [ ] Documentation states that POS application logic is intentionally not implemented in this scaffold.

**Verification:**

- [ ] Manual review: documented commands match actual scripts.
- [ ] All documented scaffold verification commands have been run at least once.

**Dependencies:** Tasks 1-10

**Files likely touched:**

- `README.md`

**Estimated scope:** Small: 1 file

### Checkpoint: Scaffold Complete

- [ ] `go -C backend test ./...` passes.
- [ ] `go -C backend test -tags=integration ./...` passes.
- [ ] `go -C backend vet ./...` passes.
- [ ] `npm --prefix frontend test` passes.
- [ ] `npm --prefix frontend run check` passes.
- [ ] `npm --prefix frontend run build` passes.
- [ ] Backend container image builds.
- [ ] Frontend container image builds.
- [ ] `podman compose up --build` starts the stack.
- [ ] `npx playwright test` passes against the running app.
- [ ] No POS business logic has been implemented.

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Podman Compose command differs by local installation (`podman compose` vs `podman-compose`) | Medium | Document the tested command and keep `compose.yaml` standards-compatible. |
| Playwright adds dependency weight early | Low | Keep Playwright only at the repo/root dev layer and never include it in production containers. |
| Frontend dev proxy and Caddy proxy drift | Medium | Use the same `/api/health` relative path in frontend code and test both local and composed flows. |
| Scaffold accidentally grows into application logic | High | Keep health endpoint and landing page as the only behavior; defer auth, menu, orders, and reporting to later plans. |
| PostgreSQL is present but unused in scaffold | Low | Treat PostgreSQL as Compose foundation only; add migrations and Testcontainers in the first database-backed feature task. |

## Parallelization Opportunities

- Tasks 1 and 4 can start in parallel after agreeing on the `/api/health` contract.
- Tasks 2 and 3 can run after Task 1 while Tasks 5 and 6 follow the frontend scaffold.
- Tasks 7 and 8 can run in parallel after backend and frontend builds work.
- Task 10 should wait until the frontend, backend, and Compose wiring are stable.

## Open Questions

- None for the scaffold phase.

## Resolved Scaffold Decisions

- Browser-facing Compose port: use frontend/Caddy on host port `8080`; backend remains internal to the Compose network.
- E2E dependency layout: use a root `package.json` for Playwright only; keep frontend dependencies isolated under `frontend/`.
- Health endpoint contract: keep `GET /api/health` as liveness-only for this scaffold; defer database readiness to a future endpoint if needed.

## Review Gate

Before implementation, confirm:

- [ ] The `/api/health` contract is acceptable as the first backend endpoint.
- [ ] The landing page health-status flow is sufficient for the first end-to-end test.
- [ ] PostgreSQL should be included in Compose now even though no database logic is implemented.
- [ ] No task should implement auth, orders, menu, reporting, printing, QRIS, or queue-number behavior.
