# AGENTS.md

Guidance for AI coding agents working in this repository.

## Project Overview

Coffee POS is a small coffee shop point-of-sale MVP. The current codebase is a foundation scaffold plus the first backend authentication slice:

- Go backend API in `backend/`
- Preact + Vite + TypeScript frontend in `frontend/`
- Playwright browser smoke tests in `tests/e2e/`
- Product specs, plans, and reviews in `docs/`
- Local agent skills in `skills/`

The MVP direction is defined in `docs/specs/small-coffee-shop-pos-mvp-spec.md`. Treat that spec as product/architecture intent, but verify against current source before changing code because the implementation is still early.

## Required Skill Workflow

This repository uses skill-driven execution. If a request matches a skill in `skills/<skill-name>/SKILL.md`, read and follow that skill before implementing.

Intent mapping:

- Feature or new functionality: `spec-driven-development`, then `incremental-implementation`, then `test-driven-development`
- Planning or task breakdown: `planning-and-task-breakdown`
- Bug, failure, or unexpected behavior: `debugging-and-error-recovery`
- Code review: `code-review-and-quality`
- Refactoring or simplification: `code-simplification`
- API or interface design: `api-and-interface-design`
- UI work: `frontend-ui-engineering`
- Security-sensitive changes: `security-and-hardening`
- Documentation or architecture context: `documentation-and-adrs` and/or `context-engineering`
- Shipping/deployment preparation: `shipping-and-launch`

Do not skip the skill because a task looks small. Read the relevant `SKILL.md`, follow its workflow, and keep the change scoped.

## Commands

Backend:

```sh
go -C backend test ./...
go -C backend test -tags=integration ./...
go -C backend vet ./...
go -C backend run ./cmd/coffee-pos serve
go -C backend run ./cmd/coffee-pos auth hash-pin <6-digit-pin>
```

Frontend:

```sh
npm --prefix frontend install
npm --prefix frontend run dev
npm --prefix frontend test
npm --prefix frontend run check
npm --prefix frontend run build
```

End-to-end:

```sh
npm install
npm run test:e2e
```

Containers:

```sh
podman build -f backend/Containerfile -t coffee-pos-backend:dev backend
podman build -f frontend/Containerfile -t coffee-pos-frontend:dev frontend
export CASHIER_PIN_HASH="$(go -C backend run ./cmd/coffee-pos auth hash-pin 123456)"
podman compose up --build -d
podman ps --filter label=io.podman.compose.project=coffee-pos
curl -i http://localhost:8080/api/health
podman compose down
```

The example above uses `123456` as a local development PIN only. For any shared or production-like environment, generate a different 6-digit PIN hash and provide it through the shell environment or secret manager instead of committing it.

Run the smallest relevant verification for the change. For backend behavior, prefer targeted `go test` first, then broader `go -C backend test ./...`. For frontend behavior, use Vitest/type checks; use Playwright when browser workflow behavior is involved.

## Runtime Configuration

The backend currently requires:

- `CASHIER_PIN_HASH`: bcrypt hash of the 6-digit cashier PIN. Generate it with `go -C backend run ./cmd/coffee-pos auth hash-pin <pin>`.
- `PORT`: defaults to `8080`.
- `APP_ENV`: defaults to `development`; `production` forces secure session cookies.
- `SESSION_COOKIE_NAME`: defaults to `coffee_pos_session`.
- `SESSION_COOKIE_SECURE`: optional boolean outside production.

Never embed the cashier PIN or PIN hash in the frontend. Do not commit real secrets or local `.env` files.

Known deployment gotcha: `compose.yaml` requires `CASHIER_PIN_HASH` from the shell environment. Generate a local development hash before running Compose, and do not store the hash in source control.

## Architecture

### Backend

The backend follows a small hexagonal architecture:

```text
HTTP / CLI adapters  ->  application use cases  ->  domain model/services
security adapters    ->  application ports      ->  domain model/services
```

Current packages:

- `backend/cmd/coffee-pos/`: CLI entrypoint. Supports `serve` and `auth hash-pin`.
- `backend/internal/domain/auth/`: pure auth domain rules such as PIN format and session expiry.
- `backend/internal/app/auth/`: auth use cases and port interfaces.
- `backend/internal/adapters/http/`: `net/http` router, JSON handlers, cookies, auth middleware.
- `backend/internal/adapters/security/`: bcrypt PIN hashing, session IDs, in-memory sessions, in-memory rate limiting.
- `backend/internal/config/`: environment loading and validation.

Backend rules:

- Keep domain packages free of `net/http`, environment access, persistence details, and DTO concerns.
- Define ports in application packages when use cases need infrastructure.
- Implement infrastructure in adapters.
- HTTP handlers should translate requests/responses and call use cases; they should not own business rules.
- Time-dependent behavior should go through a clock seam/port so Asia/Jakarta behavior is testable.
- Current sessions and rate limiting are in-memory. Treat that as an early implementation detail, not durable storage.
- Use Go standard library `net/http` patterns already present in `router.go`.

### Frontend

The frontend is a Preact app built with Vite:

- `frontend/src/App.tsx`: current scaffold UI and backend health display.
- `frontend/src/lib/`: frontend API helpers.
- `frontend/src/test/`: Vitest setup.
- `frontend/Caddyfile`: production static serving plus `/api/*` reverse proxy.

Frontend rules:

- Keep production runtime static: Caddy serves built assets; Node/Vite are build-time and dev-time only.
- Call backend APIs through relative `/api/...` URLs to preserve same-origin cookie behavior.
- Use TypeScript and small Preact components. Avoid large UI/runtime frameworks for the MVP.
- Keep CSS plain and responsive unless the project intentionally adopts a different styling approach.

### API And Auth Surface

Current backend endpoints:

- `GET /api/health`
- `POST /api/auth/login`
- `POST /api/auth/logout`
- `GET /api/auth/session`
- `GET /api/pos/ping` protected by session middleware

Auth behavior:

- Cashier PIN must be exactly 6 ASCII digits.
- PIN verification is backend-only using bcrypt.
- Login sets an HttpOnly session cookie.
- Session expiry is the earlier of 12 hours or the end of the Asia/Jakarta business day.
- Login failures are rate-limited through the configured rate limiter.

## Documentation Sources

Use these docs before making product or architecture changes:

- `docs/specs/small-coffee-shop-pos-mvp-spec.md`: MVP requirements and target architecture.
- `docs/plan/`: implementation plans for slices.
- `docs/reviews/`: prior review findings and risk notes.
- `docs/ideas/`: original product ideas and refinements.

If implementation differs from docs, surface the mismatch. Update the relevant doc when the change intentionally alters architecture, commands, or product behavior.

## Known Inconsistencies To Handle Deliberately

- The spec recommends Go 1.26.x, while `backend/go.mod` and `backend/Containerfile` currently use Go 1.25.0. If touching build images or Go tooling, resolve this intentionally instead of copying the mismatch forward.
- `README.md` still describes the scaffold as having no POS application logic, but backend auth logic now exists. Update docs when working in that area.
- PostgreSQL is present in Compose for future database-backed work, but current backend auth/session storage is in-memory and does not use the database.

## Git And Change Discipline

- Check `git status --short` before editing and do not overwrite user changes.
- Use `apply_patch` or normal editor-style edits for manual file changes.
- Keep changes narrow and avoid drive-by refactors.
- Do not edit generated/build output such as `frontend/dist/`, `node_modules/`, or `test-results/`.
- Do not commit secrets, `.env`, or local machine configuration.

Before finishing, summarize changed files, verification performed, and any skipped checks with the reason.
