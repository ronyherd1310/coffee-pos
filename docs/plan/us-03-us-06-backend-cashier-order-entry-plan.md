# Implementation Plan: US-03 to US-06 Backend Cashier Order Entry

**Status:** Draft for review

## Overview

Implement the backend portion of the `Cashier Order Entry` section from `docs/specs/small-coffee-shop-pos-mvp-spec.md`, covering US-03 through US-06. This plan provides the backend menu read model needed by the cashier screen, backend validation for order lines and required modifiers, paid-order creation for Cash and QRIS, transaction-safe Asia/Jakarta daily queue numbers, and same-day cancellation of accidental paid orders. Frontend cart state, confirmation dialogs, QRIS image display, ticket print CSS, today's order list, and daily reporting are intentionally out of scope for this backend slice.

## Scope

In scope:

- Protected backend API for the cashier screen to read active seeded menu items and required modifiers.
- Backend-only order domain rules for payment methods, quantities, per-line modifiers, notes, totals, business dates, queue numbers, and cancellation status.
- PostgreSQL schema for paid orders, order lines, line modifier snapshots, and daily queue counters.
- Backend validation that resolves submitted menu and modifier selections against persisted menu data.
- Backend recalculation of prices and totals from database-backed menu data, not client-supplied prices.
- Paid-order creation endpoint for Cash and QRIS.
- Queue number allocation that resets by Asia/Jakarta business date and is safe under concurrent order creation.
- Same-day paid-order cancellation endpoint that keeps audit history and never deletes paid orders.
- Unit, HTTP, and PostgreSQL integration tests for backend behavior.

Out of scope:

- Frontend order-entry components, cart state, dialogs, redirects, or QRIS image rendering.
- Static QRIS asset management beyond accepting `qris` as a valid payment method.
- Browser ticket rendering and receipt-printer CSS.
- Today's Orders list, paid-order detail lookup, reprint flow, and search by queue number.
- Daily sales summary endpoint and aggregation, except that cancelled orders must be stored with a status that later reporting can exclude.
- Refunds, partial cancellation, editing paid orders, service type labels, inventory, payment gateway integration, or digital queue displays.
- Menu management CRUD.

## Architecture Decisions

- Build the order slice with the existing hexagonal shape: `internal/domain/orders` for pure rules, `internal/app/orders` for use cases and ports, PostgreSQL code under `internal/adapters/postgres`, and HTTP DTOs under `internal/adapters/http`.
- Keep `internal/domain/orders` independent from HTTP, config, SQL, sqlc, and menu persistence. Domain code should calculate totals and validate pure order concepts only.
- Use stable menu and modifier slugs in cashier API requests, such as `menuItemSlug`, `groupSlug`, and `optionSlug`. The backend resolves those slugs to current active menu rows and stores database foreign keys plus immutable name/price snapshots on the paid order.
- Do not accept client-supplied prices, subtotals, totals, queue numbers, paid timestamps, statuses, or business dates in create-order requests.
- Persist only paid orders. There is no draft-order table or unpaid-order API in this slice.
- Cap order notes at 500 characters in backend validation so payloads and receipt tickets stay manageable.
- Store queue numbers as integers with a `(business_date, queue_number)` uniqueness constraint. The frontend can format display labels such as `001`; the backend returns the numeric `queueNumber`.
- Allocate the next queue number inside the same PostgreSQL transaction that inserts the paid order, using a daily counter row with row-level locking or an equivalent atomic upsert strategy.
- Derive `business_date` from the injected clock and the configured Asia/Jakarta location, not from the client.
- `POST /api/pos/orders` represents "cashier has confirmed paid" after the frontend confirmation dialog. The backend should not create a separate pre-payment or pre-confirmation record.
- Cancellation is a status transition from `paid` to `cancelled`; paid orders are never deleted. Cancellation is allowed only on the same Asia/Jakarta business date as the paid order, and no cashier-entered cancellation reason is required for MVP.
- Keep database migrations explicit. The server should not auto-migrate or perform schema-version repair on `serve`; deployment and local setup must run database commands such as `db migrate` and `db seed` before order-entry endpoints are used.

## Dependency Graph

```text
US-01 auth middleware and session protection
US-02 database config, migrations, seeded menu data
  -> cashier menu read model
  -> order domain rules and API contracts
      -> order persistence schema and sqlc queries
          -> create paid order application use case
              -> PostgreSQL queue allocation and paid-order repository
                  -> protected create paid order HTTP endpoint
                      -> cancellation use case
                          -> PostgreSQL cancellation repository method
                              -> protected cancel order HTTP endpoint
```

## Task List

### Phase 1: Order Foundation

## Task 1: Add Order Domain Rules

**Description:** Add pure order domain types and validation for paid order drafts. This should cover Cash/QRIS payment methods, positive quantities, optional notes, per-line modifiers, total calculation, paid/cancelled statuses, and Asia/Jakarta business-date derivation without importing HTTP, SQL, config, or menu repository packages.

**Acceptance criteria:**

- [ ] `internal/domain/orders` defines Cash and QRIS as the only valid payment methods.
- [ ] Domain validation rejects empty orders, non-positive quantities, missing item names, negative prices, duplicate modifier groups on one line, invalid payment methods, and notes longer than 500 characters.
- [ ] Total calculation uses `(unit price + modifier deltas) * quantity` per line and supports two lines for the same menu item with different modifiers without merging them.
- [ ] Domain code can derive the Asia/Jakarta business date from an injected `time.Time` and `*time.Location`.
- [ ] Domain code defines paid and cancelled statuses without allowing deletion as a correction path.

**Verification:**

- [ ] Targeted tests pass: `go -C backend test ./internal/domain/orders`
- [ ] Backend tests pass: `go -C backend test ./...`
- [ ] Unit tests cover Cash/QRIS validation, invalid payment methods, multiple lines for the same item, total calculation, required positive quantity, 500-character note limit, and Asia/Jakarta date boundary cases.

**Dependencies:** US-01 auth foundation and US-02 menu seed plan completed or in progress; no order-code dependency.

**Files likely touched:**

- `backend/internal/domain/orders/order.go`
- `backend/internal/domain/orders/total.go`
- `backend/internal/domain/orders/order_test.go`
- `backend/internal/domain/orders/total_test.go`

**Estimated scope:** Medium: 4 files

## Task 2: Add Order Persistence Schema And Query Foundation

**Description:** Add the database foundation for paid orders, order line snapshots, line modifier snapshots, and daily queue counters. The schema should preserve audit history, support future reporting, and enforce database-level invariants for payment method, status, positive quantities, and unique queue numbers per business date.

**Acceptance criteria:**

- [ ] A new migration creates `orders`, `order_lines`, `order_line_modifiers`, and `daily_queue_counters` or an equivalent transaction-safe counter table.
- [ ] `orders` stores `business_date`, `queue_number`, `status`, `payment_method`, `paid_at`, `cancelled_at`, `note`, `total_rp`, timestamps, and an internal primary key.
- [ ] `order_lines` stores menu item foreign keys plus immutable snapshots for item name, slug, unit price, quantity, line total, and display order.
- [ ] `order_line_modifiers` stores modifier group and option foreign keys plus immutable snapshots for group name, option name, slugs, price delta, and display order.
- [ ] Database constraints prevent duplicate `(business_date, queue_number)`, invalid statuses, invalid payment methods, non-positive quantities, and negative rupiah totals.
- [ ] sqlc query files include the initial insert/detail/counter queries needed by later repository tasks.
- [ ] Existing menu migrations and seed behavior remain compatible.

**Verification:**

- [ ] Tests pass: `go -C backend test ./...`
- [ ] Integration tests pass: `go -C backend test -tags=integration ./...`
- [ ] Migration test proves applying migrations from an empty database creates the order tables.
- [ ] Manual check: `go -C backend run ./cmd/coffee-pos db migrate` can be run twice against a local PostgreSQL database without duplicating schema state.

**Dependencies:** US-02 database migration foundation.

**Files likely touched:**

- `backend/migrations/000002_create_order_schema.sql`
- `backend/queries/orders.sql`
- `backend/internal/adapters/postgres/sqlc/*`
- `backend/internal/adapters/postgres/migrations_test.go`

**Estimated scope:** Medium: 4 files

### Checkpoint: Order Foundation

- [ ] Order domain tests pass.
- [ ] Order persistence migration applies cleanly after the menu migration.
- [ ] Queue-number uniqueness is enforced by the database schema.
- [ ] `go -C backend test ./...`, `go -C backend vet ./...`, and relevant integration tests pass.

### Phase 2: Menu Read And Paid Order Creation

## Task 3: Add Cashier Menu Read Use Case And PostgreSQL Read Model

**Description:** Extend the existing menu application and PostgreSQL adapter so the cashier backend can return active menu items with their required modifier groups and options. This read model is also the source used by order creation to validate submitted slugs and recalculate prices.

**Acceptance criteria:**

- [ ] `internal/app/menu` exposes a read use case for the cashier menu.
- [ ] The read use case returns active menu categories, active menu items, required modifier groups, selection type, modifier options, slugs, and rupiah prices.
- [ ] PostgreSQL queries join `menu_item_modifier_groups`, `modifier_groups`, and `modifier_options` so each menu item exposes the modifier groups that apply to it.
- [ ] Inactive menu items are excluded from the cashier menu response and from order validation.
- [ ] Generated sqlc types remain inside the PostgreSQL adapter boundary.

**Verification:**

- [ ] Targeted tests pass: `go -C backend test ./internal/app/menu ./internal/adapters/postgres`
- [ ] Integration tests pass: `go -C backend test -tags=integration ./internal/adapters/postgres`
- [ ] Unit tests cover empty menu, active seeded menu, required modifier groups, and repository errors.

**Dependencies:** Task 2 and US-02 seeded menu data.

**Files likely touched:**

- `backend/internal/app/menu/ports.go`
- `backend/internal/app/menu/usecases.go`
- `backend/internal/app/menu/usecases_test.go`
- `backend/internal/adapters/postgres/menu_repository.go`
- `backend/queries/menu.sql`

**Estimated scope:** Medium: 5 files

## Task 4: Expose Protected Cashier Menu API

**Description:** Add a protected HTTP endpoint that returns the cashier menu read model to authenticated sessions. The endpoint should stay read-only and should not expose menu management fields or database implementation details beyond stable slugs and prices needed by the cashier UI.

**Acceptance criteria:**

- [ ] `GET /api/pos/menu` requires the existing session middleware.
- [ ] Authenticated requests return active menu categories, items, modifier groups, options, slugs, and rupiah prices.
- [ ] Unauthenticated requests return the existing protected-route `401 Unauthorized` shape.
- [ ] Handler DTOs live in the HTTP adapter and do not leak sqlc rows.
- [ ] HTTP response does not include PIN data, session data, inactive items, or menu management actions.

**Verification:**

- [ ] Tests pass: `go -C backend test ./...`
- [ ] HTTP tests cover authenticated success, unauthenticated access, empty menu, and repository failure mapping.
- [ ] Manual check with `curl` can log in and fetch `/api/pos/menu` with the session cookie.

**Dependencies:** Task 3.

**Files likely touched:**

- `backend/internal/adapters/http/menu_handlers.go`
- `backend/internal/adapters/http/router.go`
- `backend/internal/adapters/http/menu_handlers_test.go`
- `backend/cmd/coffee-pos/main.go`

**Estimated scope:** Medium: 4 files

## Task 5: Add Create Paid Order Use Case

**Description:** Add `internal/app/orders` use cases and ports for creating a paid order. The use case should translate a cashier command into a validated paid order by resolving menu slugs, enforcing required modifiers per line, recalculating totals, deriving the business date, and delegating atomic persistence plus queue allocation through application ports.

**Acceptance criteria:**

- [ ] The use case accepts lines with `menuItemSlug`, positive `quantity`, and modifier selections containing `groupSlug` and `optionSlug`.
- [ ] The use case rejects empty carts, invalid payment methods, unknown menu item slugs, inactive items, unknown modifier groups, unknown modifier options, duplicate modifier groups on a line, and missing required groups.
- [ ] Each order line must select exactly one Temperature option and one Sugar option when those required groups apply to the item.
- [ ] The use case recalculates line totals and order total from resolved menu prices and modifier deltas.
- [ ] The use case derives `businessDate` and `paidAt` from an injected clock using Asia/Jakarta.
- [ ] The use case returns a paid order detail result with internal `orderId`, numeric `queueNumber`, `businessDate`, `paidAt`, payment method, status, item lines, modifiers, note, and total.
- [ ] Unpaid drafts are not persisted because no draft use case exists.

**Verification:**

- [ ] Targeted tests pass: `go -C backend test ./internal/app/orders`
- [ ] Backend tests pass: `go -C backend test ./...`
- [ ] Unit tests use fake ports for successful Cash order, successful QRIS order, same item with different modifiers, missing Temperature, missing Sugar, unknown slug, inactive item, invalid payment method, empty cart, repository failure, and clock/date boundaries.

**Dependencies:** Tasks 1 and 3.

**Files likely touched:**

- `backend/internal/app/orders/ports.go`
- `backend/internal/app/orders/usecases.go`
- `backend/internal/app/orders/usecases_test.go`
- `backend/internal/domain/orders/order.go`

**Estimated scope:** Medium: 4 files

## Task 6: Implement PostgreSQL Paid Order Creation And Queue Allocation

**Description:** Implement the orders application persistence port with PostgreSQL. The adapter should allocate the next queue number and insert the order, lines, and modifier snapshots in one transaction so a paid order never exists without a queue number and duplicate same-day queue numbers cannot occur.

**Acceptance criteria:**

- [ ] The repository allocates queue number `1` for the first paid order of a new Asia/Jakarta business date.
- [ ] Queue numbers increment by `1` for later paid orders on the same business date.
- [ ] Queue allocation and order insertion happen in one database transaction.
- [ ] Concurrent paid-order creation cannot produce duplicate queue numbers for the same business date.
- [ ] The repository stores line and modifier snapshots using backend-resolved names, slugs, and prices.
- [ ] Repository output maps persisted rows back to the application paid order detail result.
- [ ] A database uniqueness violation on `(business_date, queue_number)` is handled as an internal consistency error, not as a successful duplicate order.

**Verification:**

- [ ] Targeted integration tests pass: `go -C backend test -tags=integration ./internal/adapters/postgres`
- [ ] Backend tests pass: `go -C backend test ./...`
- [ ] Race-sensitive or concurrent integration test creates multiple paid orders for the same date and asserts unique sequential queue numbers.
- [ ] Integration tests verify Cash and QRIS orders persist with correct totals, line snapshots, modifier snapshots, and no unpaid drafts.

**Dependencies:** Tasks 2 and 5.

**Files likely touched:**

- `backend/internal/adapters/postgres/order_repository.go`
- `backend/internal/adapters/postgres/order_repository_integration_test.go`
- `backend/queries/orders.sql`
- `backend/internal/adapters/postgres/sqlc/*`

**Estimated scope:** Medium: 4 files

## Task 7: Expose Create Paid Order HTTP Endpoint

**Description:** Add the protected HTTP endpoint that the frontend calls only after the cashier confirms payment. The handler should parse the cashier request, reject malformed or invalid payloads, call the create paid order use case, and return the paid order detail needed by the frontend post-payment screen.

**Acceptance criteria:**

- [ ] `POST /api/pos/orders` requires the existing session middleware.
- [ ] Request JSON accepts `paymentMethod`, optional `note` of at most 500 characters, and `lines` with `menuItemSlug`, `quantity`, and modifier selections by `groupSlug` and `optionSlug`.
- [ ] Request JSON does not accept client-supplied prices, totals, queue numbers, statuses, paid timestamps, or business dates.
- [ ] Malformed JSON and structurally invalid requests return `400 Bad Request` with a stable error code.
- [ ] Semantically invalid orders return `422 Unprocessable Entity` with a stable error code, without leaking SQL details.
- [ ] Valid Cash and QRIS requests return `201 Created` with paid order detail including internal `orderId`, numeric `queueNumber`, status `paid`, `paidAt`, payment method, lines, modifiers, note, and total.
- [ ] Unauthenticated requests return the existing protected-route `401 Unauthorized` shape.
- [ ] `runServe` wires the database-backed menu and order services and closes database resources during shutdown.

**Verification:**

- [ ] Tests pass: `go -C backend test ./...`
- [ ] Integration tests pass: `go -C backend test -tags=integration ./...`
- [ ] HTTP tests cover unauthenticated access, malformed JSON, empty cart, missing required modifiers, invalid payment method, Cash success, QRIS success, and handler error mapping.
- [ ] Manual check with `curl` can log in, fetch the menu, create a paid Cash order, and see a queue number in the response.

**Dependencies:** Tasks 4 and 6.

**Files likely touched:**

- `backend/internal/adapters/http/order_handlers.go`
- `backend/internal/adapters/http/order_handlers_test.go`
- `backend/internal/adapters/http/router.go`
- `backend/cmd/coffee-pos/main.go`

**Estimated scope:** Medium: 4 files

### Checkpoint: Paid Order Creation

- [ ] Authenticated cashier menu read works.
- [ ] Authenticated create paid order works for Cash and QRIS.
- [ ] Backend rejects invalid carts and never trusts client prices or totals.
- [ ] Queue numbers are unique and sequential under concurrent paid-order creation.
- [ ] `go -C backend test ./...`, `go -C backend vet ./...`, and relevant integration tests pass.

### Phase 3: Same-Day Cancellation

## Task 8: Add Cancel Paid Order Use Case

**Description:** Add application behavior for cancelling an accidental paid order. The use case should enforce same-day cancellation using Asia/Jakarta business dates, reject invalid status transitions, and return an updated order detail while preserving all original order, line, modifier, and queue-number data.

**Acceptance criteria:**

- [ ] Cancellation accepts an internal `orderId` and uses the injected clock/location to determine the current Asia/Jakarta business date.
- [ ] A paid order from the same business date can transition to `cancelled`.
- [ ] Cancellation records `cancelledAt` and preserves `paidAt`, `queueNumber`, `businessDate`, payment method, total, lines, modifiers, and note.
- [ ] Cancellation does not require or store a cashier-entered cancellation reason in MVP.
- [ ] Already-cancelled orders cannot be cancelled again.
- [ ] Orders from previous business dates cannot be cancelled through this use case.
- [ ] Missing orders return a not-found application result.
- [ ] The use case exposes status values that future daily-summary code can exclude with `status = cancelled`.

**Verification:**

- [ ] Targeted tests pass: `go -C backend test ./internal/app/orders`
- [ ] Backend tests pass: `go -C backend test ./...`
- [ ] Unit tests use fake ports for successful same-day cancellation, already-cancelled conflict, previous-day conflict, missing order, repository failure, and Asia/Jakarta boundary cases.

**Dependencies:** Task 5.

**Files likely touched:**

- `backend/internal/app/orders/usecases.go`
- `backend/internal/app/orders/ports.go`
- `backend/internal/app/orders/usecases_test.go`
- `backend/internal/domain/orders/order.go`

**Estimated scope:** Medium: 4 files

## Task 9: Implement PostgreSQL Cancellation Persistence

**Description:** Implement the cancellation persistence behavior for PostgreSQL. The adapter should update only eligible paid orders, keep all child rows intact, and return the same detail shape used by paid-order creation.

**Acceptance criteria:**

- [ ] Cancelling a paid same-day order updates `status` to `cancelled`, sets `cancelled_at`, and updates `updated_at`.
- [ ] Cancellation does not delete or mutate order lines, line modifiers, payment method, total, paid timestamp, business date, or queue number.
- [ ] Already-cancelled orders and previous-day orders are distinguishable from successful cancellation for application error mapping.
- [ ] Cancelled orders remain queryable by internal order ID.
- [ ] The repository returns updated order detail after cancellation.

**Verification:**

- [ ] Targeted integration tests pass: `go -C backend test -tags=integration ./internal/adapters/postgres`
- [ ] Backend tests pass: `go -C backend test ./...`
- [ ] Integration tests verify same-day cancellation, previous-day rejection, already-cancelled rejection, missing order, preserved snapshots, and no row deletion.

**Dependencies:** Tasks 6 and 8.

**Files likely touched:**

- `backend/internal/adapters/postgres/order_repository.go`
- `backend/internal/adapters/postgres/order_repository_integration_test.go`
- `backend/queries/orders.sql`
- `backend/internal/adapters/postgres/sqlc/*`

**Estimated scope:** Medium: 4 files

## Task 10: Expose Cancel Paid Order HTTP Endpoint

**Description:** Add the protected HTTP endpoint that the frontend calls after the cashier confirms cancellation. The handler should parse the internal order ID, call the cancellation use case, and map application results to stable HTTP statuses.

**Acceptance criteria:**

- [ ] `POST /api/pos/orders/{orderId}/cancel` requires the existing session middleware.
- [ ] The endpoint assumes the frontend has already shown the confirmation dialog; it does not create a separate confirmation state.
- [ ] Successful cancellation returns `200 OK` with updated order detail and status `cancelled`.
- [ ] Missing or malformed order IDs return `400 Bad Request` or `404 Not Found` consistently.
- [ ] Missing orders return `404 Not Found`.
- [ ] Already-cancelled or previous-day orders return `409 Conflict`.
- [ ] Unauthenticated requests return the existing protected-route `401 Unauthorized` shape.
- [ ] Response bodies never imply that the order was deleted.

**Verification:**

- [ ] Tests pass: `go -C backend test ./...`
- [ ] Integration tests pass: `go -C backend test -tags=integration ./...`
- [ ] HTTP tests cover unauthenticated access, malformed order ID, missing order, already-cancelled conflict, previous-day conflict, and successful cancellation.
- [ ] Manual check with `curl` can create a paid order and then cancel it on the same business date.

**Dependencies:** Task 9.

**Files likely touched:**

- `backend/internal/adapters/http/order_handlers.go`
- `backend/internal/adapters/http/order_handlers_test.go`
- `backend/internal/adapters/http/router.go`

**Estimated scope:** Medium: 3 files

## Task 11: Update Backend Documentation For Runtime And API Changes

**Description:** Update project documentation to reflect that backend serve mode now depends on PostgreSQL-backed menu and order services, and document the new backend API surfaces enough for frontend implementation to integrate without reading adapter internals.

**Acceptance criteria:**

- [ ] README or backend docs mention required database setup for cashier order-entry endpoints.
- [ ] Docs list `GET /api/pos/menu`, `POST /api/pos/orders`, and `POST /api/pos/orders/{orderId}/cancel`.
- [ ] Docs clarify that `db migrate` and `db seed` must run before using cashier order entry.
- [ ] Docs clarify that backend persists paid orders only and recalculates totals server-side.
- [ ] Docs do not include real PINs, PIN hashes, QRIS credentials, or local `.env` values.

**Verification:**

- [ ] Documentation diff contains no secrets.
- [ ] Commands in documentation match implemented command names.
- [ ] Smallest relevant backend tests still pass after doc-only changes: `go -C backend test ./...`

**Dependencies:** Tasks 7 and 10.

**Files likely touched:**

- `README.md`
- `docs/plan/us-03-us-06-backend-cashier-order-entry-plan.md`

**Estimated scope:** Small: 2 files

### Checkpoint: Backend Cashier Order Entry Complete

- [ ] Cashier menu read API is protected and returns the seeded active menu.
- [ ] Paid Cash and QRIS orders persist only after confirmation-triggered API calls.
- [ ] Required per-line Temperature and Sugar modifiers are validated backend-side.
- [ ] Same menu item can appear as separate lines with different modifiers.
- [ ] Backend recalculates totals from persisted menu data.
- [ ] Daily queue numbers reset by Asia/Jakarta business date and are unique under concurrency.
- [ ] Same-day cancellation marks paid orders cancelled without deletion.
- [ ] Cancelled status is available for later reporting exclusion.
- [ ] `go -C backend test ./...` passes.
- [ ] `go -C backend vet ./...` passes.
- [ ] Relevant integration tests pass: `go -C backend test -tags=integration ./...`
- [ ] Human has reviewed and approved the plan before implementation starts.

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Queue number duplicates under concurrent payment confirmation | High | Allocate numbers inside the same database transaction as order insertion and enforce `(business_date, queue_number)` uniqueness. Add concurrent integration tests. |
| Frontend sends stale or manipulated prices | High | Do not accept prices or totals in request DTOs. Resolve menu slugs server-side and snapshot backend-calculated names/prices/totals. |
| US-02 menu/database work is not merged when this starts | Medium | Treat US-02 as a prerequisite. Start with domain tests if database foundation is still in review, then rebase the schema/query work after US-02 lands. |
| Menu changes after an order is paid could alter historical tickets | Medium | Store immutable item and modifier snapshots on each paid order. Use menu foreign keys for traceability only. |
| Same-day cancellation around midnight is ambiguous | Medium | Use only the injected backend clock and Asia/Jakarta location. Add tests for just before and after midnight. |
| Endpoint scope drifts into Today's Orders or reporting | Medium | Keep list/detail/reprint/daily summary out of this plan except for returning create/cancel detail responses needed by this flow. |
| Server startup changes from auth-only to database-backed | Medium | Document `DATABASE_URL`, migration, and seed requirements. Keep database config errors explicit and covered by tests. |

## Resolved Decisions

- Order notes are capped at 500 characters in backend validation.
- Cashier menu/order APIs use stable slugs for menu item, modifier group, and modifier option selections. Numeric database IDs remain internal except for the paid order's internal `orderId`, which can be used for API routing but must not be displayed as the customer-facing order number.
- Cancellation does not capture a cashier-entered reason in MVP. The backend stores status `cancelled` and `cancelledAt` only.
- `serve` does not auto-migrate or repair schema state. Operators and local setup must run `db migrate` before `serve`, then `db seed` before using cashier order entry.

## Parallelization Opportunities

- Tasks 1 and 2 can proceed in parallel once the order API contract is accepted.
- Task 3 can proceed in parallel with Task 1 if US-02 menu schema is stable.
- Task 4 should wait for Task 3.
- Tasks 5 through 7 should remain sequential because the HTTP endpoint depends on the application contract and PostgreSQL persistence behavior.
- Tasks 8 through 10 should remain sequential after paid-order creation is working.
- Task 11 can be drafted during Tasks 7 through 10, then finalized after endpoint names and runtime behavior are implemented.
