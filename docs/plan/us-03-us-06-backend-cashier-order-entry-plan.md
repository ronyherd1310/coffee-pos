# Implementation Plan: US-03 to US-06 Backend Cashier Order Entry

**Status:** Revised after 2026-06-30 plan review

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
- Idempotency protection for paid-order creation retries after the cashier confirms payment.
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
- Use stable menu and modifier slugs in cashier API requests, such as `menuItemSlug`, `groupSlug`, and `optionSlug`. The backend resolves those slugs against modifier groups attached to the selected item and stores database foreign keys plus immutable name/price snapshots on the paid order.
- Do not accept client-supplied prices, subtotals, totals, queue numbers, paid timestamps, statuses, order IDs, or business dates in create-order requests.
- Strictly reject unknown JSON fields at the top level, line level, and modifier level so forbidden fields cannot be silently ignored.
- Persist only paid orders. There is no draft-order table or unpaid-order API in this slice.
- Require a client-generated `clientRequestId` on paid-order creation. The value is a canonical lowercase UUID string, the frontend must reuse the same ID for retries of the same confirmed draft, and the backend uses it as the idempotency key.
- Cap order notes at 500 characters in backend validation so payloads and receipt tickets stay manageable.
- Cap line quantity at 99 and use `int64`/PostgreSQL `bigint` for calculated line totals and order totals. Reject totals that overflow the chosen storage type.
- Store queue numbers as integers with a `(business_date, queue_number)` uniqueness constraint. The frontend can format display labels such as `001`; the backend returns the numeric `queueNumber`.
- Cancelled orders retain their original queue number permanently. Later same-day orders continue from the daily counter and never reuse cancelled queue numbers.
- Allocate the next queue number inside the same PostgreSQL transaction that inserts the paid order, using a daily counter row with row-level locking or an equivalent atomic upsert strategy.
- Derive `business_date` from the injected clock and the configured Asia/Jakarta location, not from the client.
- `POST /api/pos/orders` represents "cashier has confirmed paid" after the frontend confirmation dialog. The backend should not create a separate pre-payment or pre-confirmation record.
- Cancellation is a status transition from `paid` to `cancelled`; paid orders are never deleted. Cancellation is allowed only on the same Asia/Jakarta business date as the paid order, and no cashier-entered cancellation reason is required for MVP.
- Expose `orderId` as an API string containing the positive base-10 decimal representation of the internal PostgreSQL `orders.id`, with no sign and no leading zeroes. Path values outside that format fail parsing with `400 Bad Request`; well-formed missing IDs return `404 Not Found`.
- Reuse one canonical `PaidOrderDetail` response shape for create-order, cancel-order, and future order-detail endpoints.
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

## API Contracts

### Create Paid Order Request

`POST /api/pos/orders` accepts only these JSON fields:

- `clientRequestId`: required canonical lowercase UUID string idempotency key generated by the frontend for one confirmed draft.
- `paymentMethod`: required string, either `cash` or `qris`.
- `note`: optional string, omitted or at most 500 characters. `null` is not accepted.
- `lines`: required non-empty array of order lines.
- `lines[].menuItemSlug`: required string.
- `lines[].quantity`: required integer from 1 through 99.
- `lines[].modifiers`: required array of modifier selections.
- `lines[].modifiers[].groupSlug`: required string.
- `lines[].modifiers[].optionSlug`: required string.

Unknown fields return `400 Bad Request` with code `unknown_field`. Forbidden server-owned fields, including client-supplied prices, totals, queue numbers, statuses, paid timestamps, order IDs, and business dates, return `400 Bad Request` with code `forbidden_field`.

### PaidOrderDetail Response

Create-order, cancel-order, and future paid-order detail endpoints return the same `PaidOrderDetail` shape:

- `orderId`: string positive base-10 internal order ID.
- `queueNumber`: integer daily queue number.
- `businessDate`: Asia/Jakarta business date formatted as `YYYY-MM-DD`.
- `status`: `paid` or `cancelled`.
- `paymentMethod`: `cash` or `qris`.
- `paidAt`: RFC3339 timestamp.
- `cancelledAt`: nullable RFC3339 timestamp.
- `note`: nullable string.
- `totalRp`: integer rupiah total.
- `lines[]`: persisted line snapshots with `menuItemSlug`, `menuItemName`, `unitPriceRp`, `quantity`, `lineTotalRp`, and `modifiers[]`.
- `lines[].modifiers[]`: persisted modifier snapshots with `groupSlug`, `groupName`, `optionSlug`, `optionName`, and `priceDeltaRp`.

### Error Code Mapping

- `400 Bad Request`: malformed JSON (`invalid_json`), unknown fields (`unknown_field`), forbidden server-owned fields (`forbidden_field`), missing required JSON fields (`missing_field`), null or wrong-typed JSON fields (`invalid_field_type`), malformed `orderId` (`invalid_order_id`), malformed `clientRequestId` (`invalid_client_request_id`), or non-integer quantity (`invalid_field_type`).
- `401 Unauthorized`: missing or invalid cashier session.
- `404 Not Found`: well-formed `orderId` does not exist.
- `409 Conflict`: idempotency key reused with a different request (`idempotency_conflict`), already-cancelled order, previous-day cancellation, or concurrent cancellation loser (`order_not_cancellable`).
- `422 Unprocessable Entity`: well-typed request that fails order semantics (`invalid_order`), such as inactive menu item, missing required modifier group, unattached modifier group, option under the wrong group, invalid payment method, or quantity outside 1 through 99.

## Task List

### Phase 1: Order Foundation

## Task 1: Add Order Domain Rules

**Description:** Add pure order domain types and validation for paid order drafts. This should cover Cash/QRIS payment methods, bounded quantities, optional notes, per-line modifiers, total calculation, paid/cancelled statuses, and Asia/Jakarta business-date derivation without importing HTTP, SQL, config, or menu repository packages.

**Acceptance criteria:**

- [ ] `internal/domain/orders` defines Cash and QRIS as the only valid payment methods.
- [ ] Domain validation rejects empty orders, quantities outside 1 through 99, missing item names, negative prices, duplicate modifier groups on one line, invalid payment methods, and notes longer than 500 characters.
- [ ] Total calculation uses `int64`, applies `(unit price + modifier deltas) * quantity` per line, rejects arithmetic overflow, and supports two lines for the same menu item with different modifiers without merging them.
- [ ] Domain code can derive the Asia/Jakarta business date from an injected `time.Time` and `*time.Location`.
- [ ] Domain code defines paid and cancelled statuses without allowing deletion as a correction path.

**Verification:**

- [ ] Targeted tests pass: `go -C backend test ./internal/domain/orders`
- [ ] Backend tests pass: `go -C backend test ./...`
- [ ] Unit tests cover Cash/QRIS validation, invalid payment methods, multiple lines for the same item, total calculation, quantity `0`, quantity `100`, 500-character note limit, total overflow, and Asia/Jakarta date boundary cases.

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
- [ ] `orders` stores `business_date`, `queue_number`, `status`, `payment_method`, `paid_at`, `cancelled_at`, `note`, `total_rp`, `client_request_id`, `request_hash`, timestamps, and an internal primary key.
- [ ] `order_lines` stores menu item foreign keys plus immutable snapshots for item name, slug, unit price, quantity, line total, and display order.
- [ ] `order_line_modifiers` stores modifier group and option foreign keys plus immutable snapshots for group name, option name, slugs, price delta, and display order.
- [ ] Rupiah totals and line totals use PostgreSQL `bigint`; menu item and modifier prices may remain integer rupiah values.
- [ ] Database constraints prevent duplicate `(business_date, queue_number)`, duplicate `client_request_id`, invalid statuses, invalid payment methods, quantities outside 1 through 99, and negative rupiah totals.
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
- [ ] Create-order idempotency uniqueness is enforced by the database schema.
- [ ] `go -C backend test ./...`, `go -C backend vet ./...`, and relevant integration tests pass.

### Phase 2: Menu Read And Paid Order Creation

## Task 3: Add Cashier Menu Read Use Case And PostgreSQL Read Model

**Description:** Extend the existing menu application and PostgreSQL adapter so the cashier backend can return active menu items with their required modifier groups and options. This read model is also the source used by order creation to validate submitted slugs and recalculate prices.

**Acceptance criteria:**

- [ ] `internal/app/menu` exposes a read use case for the cashier menu.
- [ ] The read use case returns active menu categories, active menu items, required modifier groups, selection type, modifier options, slugs, and rupiah prices.
- [ ] The read use case returns categories, items, groups, and options already sorted for display by `sort_order` and then stable row identity.
- [ ] PostgreSQL queries join `menu_item_modifier_groups`, `modifier_groups`, and `modifier_options` so each menu item exposes the modifier groups that apply to it.
- [ ] Inactive menu items are excluded from the cashier menu response and from order validation.
- [ ] Generated sqlc types remain inside the PostgreSQL adapter boundary.

**Verification:**

- [ ] Targeted tests pass: `go -C backend test ./internal/app/menu ./internal/adapters/postgres`
- [ ] Integration tests pass: `go -C backend test -tags=integration ./internal/adapters/postgres`
- [ ] Unit tests cover empty menu, active seeded menu, required modifier groups, deterministic display ordering, and repository errors.

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
- [ ] Authenticated requests return active menu categories, items, modifier groups, options, slugs, and rupiah prices in backend-defined display order.
- [ ] Unauthenticated requests return the existing protected-route `401 Unauthorized` shape.
- [ ] Handler DTOs live in the HTTP adapter and do not leak sqlc rows.
- [ ] HTTP response does not include PIN data, session data, inactive items, or menu management actions.

**Verification:**

- [ ] Tests pass: `go -C backend test ./...`
- [ ] HTTP tests cover authenticated success, unauthenticated access, empty menu, deterministic response ordering, and repository failure mapping.
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

- [ ] The use case accepts canonical lowercase UUID `clientRequestId`, `paymentMethod`, optional `note`, and lines with `menuItemSlug`, quantity from 1 through 99, and modifier selections containing `groupSlug` and `optionSlug`.
- [ ] The use case rejects empty carts, invalid payment methods, unknown menu item slugs, inactive items, unknown modifier groups, unknown modifier options, duplicate modifier groups on a line, missing required groups, groups not attached to the selected menu item, options that do not belong to the submitted group, and fully inactive menu state.
- [ ] Each required modifier group attached to the selected menu item must be selected exactly once; the seeded menu tests must prove the Temperature and Sugar case without hardcoding those group names into the generic rule.
- [ ] The use case recalculates line totals and order total from resolved menu prices and modifier deltas.
- [ ] The use case derives `businessDate` and `paidAt` from an injected clock using Asia/Jakarta.
- [ ] The use case returns the canonical `PaidOrderDetail` result with `orderId` as an API string, numeric `queueNumber`, `businessDate`, `paidAt`, nullable `cancelledAt`, payment method, status, item lines, modifiers, nullable note, and total.
- [ ] Retrying the same `clientRequestId` with the same canonical request returns the original `PaidOrderDetail` without creating another order or allocating another queue number.
- [ ] Reusing the same `clientRequestId` with a different canonical request returns an idempotency conflict.
- [ ] Unpaid drafts are not persisted because no draft use case exists.

**Verification:**

- [ ] Targeted tests pass: `go -C backend test ./internal/app/orders`
- [ ] Backend tests pass: `go -C backend test ./...`
- [ ] Unit tests use fake ports for successful Cash order, successful QRIS order, same item with different modifiers, missing required group, unattached group, wrong-group option, duplicate group, unknown slug, inactive item, fully inactive menu, invalid payment method, empty cart, idempotent retry, idempotency conflict, repository failure, and clock/date boundaries.

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
- [ ] Cancelled orders retain their original queue number permanently, and later same-day orders never reuse cancelled queue numbers.
- [ ] Queue allocation and order insertion happen in one database transaction.
- [ ] Concurrent paid-order creation cannot produce duplicate queue numbers for the same business date.
- [ ] Concurrent create-order calls with the same `clientRequestId` and same canonical request produce exactly one persisted order and return the same `PaidOrderDetail`.
- [ ] Concurrent or later create-order calls with the same `clientRequestId` and a different canonical request return an idempotency conflict and do not allocate a queue number.
- [ ] The repository stores line and modifier snapshots using backend-resolved names, slugs, and prices.
- [ ] Repository output maps persisted rows back to the canonical `PaidOrderDetail` result.
- [ ] A database uniqueness violation on `(business_date, queue_number)` is handled as an internal consistency error, not as a successful duplicate order.

**Verification:**

- [ ] Targeted integration tests pass: `go -C backend test -tags=integration ./internal/adapters/postgres`
- [ ] Backend tests pass: `go -C backend test ./...`
- [ ] Race-sensitive or concurrent integration test creates multiple paid orders for the same date and asserts unique sequential queue numbers.
- [ ] Integration tests verify same-key retry returns the original detail, same-key different payload conflicts without consuming a queue number, and separate client request IDs create separate paid orders.
- [ ] Integration tests verify Cash and QRIS orders persist with correct totals, line snapshots, modifier snapshots, canonical detail shape, and no unpaid drafts.

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
- [ ] Request JSON accepts only canonical lowercase UUID `clientRequestId`, `paymentMethod`, optional `note` of at most 500 characters, and `lines` with `menuItemSlug`, `quantity`, and modifier selections by `groupSlug` and `optionSlug`.
- [ ] Unknown top-level, line-level, and modifier-level JSON fields are rejected with `400 Bad Request` and code `unknown_field`.
- [ ] Request JSON rejects client-supplied prices, totals, queue numbers, statuses, paid timestamps, order IDs, and business dates with `400 Bad Request` and code `forbidden_field` instead of silently ignoring them.
- [ ] Missing `clientRequestId`, `clientRequestId: null`, non-string `clientRequestId`, malformed UUID `clientRequestId`, `paymentMethod: null`, missing `paymentMethod`, non-string `paymentMethod`, `note: null`, non-string `note`, `lines: null`, non-array `lines`, empty `lines`, empty line objects, missing `menuItemSlug`, `menuItemSlug: null`, non-string `menuItemSlug`, missing `quantity`, non-numeric quantity, zero quantity, `modifiers: null`, non-array `modifiers`, missing `groupSlug`, `groupSlug: null`, non-string `groupSlug`, missing `optionSlug`, `optionSlug: null`, and non-string `optionSlug` return `400 Bad Request` with stable request-shape error codes.
- [ ] Semantically invalid orders return `422 Unprocessable Entity` with a stable error code, without leaking SQL details.
- [ ] Reusing `clientRequestId` with a different canonical request returns `409 Conflict` with a stable idempotency error code.
- [ ] Valid first-time Cash and QRIS requests return `201 Created` with canonical `PaidOrderDetail`.
- [ ] Valid retries with the same `clientRequestId` and same canonical request return `200 OK` with the original canonical `PaidOrderDetail`, original queue number, and no duplicate order.
- [ ] Unauthenticated requests return the existing protected-route `401 Unauthorized` shape.
- [ ] `runServe` wires the database-backed menu and order services and closes database resources during shutdown.

**Verification:**

- [ ] Tests pass: `go -C backend test ./...`
- [ ] Integration tests pass: `go -C backend test -tags=integration ./...`
- [ ] HTTP tests cover unauthenticated access, malformed JSON, unknown fields at every level, forbidden fields, missing/null/wrong-typed `clientRequestId`, malformed UUID `clientRequestId`, null and wrong-typed cases for every request field, empty cart, missing required modifiers, invalid payment method, inactive menu item, Cash success, QRIS success, same-key retry, same-key conflict, and handler error mapping.
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
- [ ] Idempotent create-order retries return the original order without consuming new queue numbers.
- [ ] Queue numbers are unique and sequential under concurrent paid-order creation.
- [ ] `go -C backend test ./...`, `go -C backend vet ./...`, and relevant integration tests pass.

### Phase 3: Same-Day Cancellation

## Task 8: Add Cancel Paid Order Use Case

**Description:** Add application behavior for cancelling an accidental paid order. The use case should enforce same-day cancellation using Asia/Jakarta business dates, reject invalid status transitions, and return an updated order detail while preserving all original order, line, modifier, and queue-number data.

**Acceptance criteria:**

- [ ] Cancellation accepts an internal order ID represented at the API boundary as a positive base-10 string and uses the injected clock/location to determine the current Asia/Jakarta business date.
- [ ] A paid order from the same business date can transition to `cancelled`.
- [ ] Cancellation records `cancelledAt` and preserves `paidAt`, `queueNumber`, `businessDate`, payment method, total, lines, modifiers, and note.
- [ ] Cancellation does not require or store a cashier-entered cancellation reason in MVP.
- [ ] Already-cancelled orders cannot be cancelled again.
- [ ] Orders from previous business dates cannot be cancelled through this use case.
- [ ] An order paid at `23:59` Asia/Jakarta cannot be cancelled after `00:00` Asia/Jakarta the next business date.
- [ ] Missing orders return a not-found application result.
- [ ] Concurrent cancellation attempts for the same paid order yield exactly one successful transition and at least one conflict/not-eligible result.
- [ ] The use case exposes status values that future daily-summary code can exclude with `status = cancelled`.

**Verification:**

- [ ] Targeted tests pass: `go -C backend test ./internal/app/orders`
- [ ] Backend tests pass: `go -C backend test ./...`
- [ ] Unit tests use fake ports for successful same-day cancellation, already-cancelled conflict, previous-day conflict, order paid before midnight and cancelled after midnight, missing order, concurrent cancellation conflict mapping, repository failure, and Asia/Jakarta boundary cases.

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
- [ ] Cancellation never decrements the daily queue counter or makes the cancelled order's queue number available for reuse.
- [ ] A later paid order on the same business date receives the next queue number after the highest allocated number, even if earlier queue numbers were cancelled.
- [ ] Already-cancelled orders and previous-day orders are distinguishable from successful cancellation for application error mapping.
- [ ] Concurrent cancellation attempts against the same paid order produce exactly one database update from `paid` to `cancelled`; the loser maps to the already-cancelled/not-eligible path.
- [ ] Cancelled orders remain queryable by internal order ID.
- [ ] The repository returns updated canonical `PaidOrderDetail` after cancellation.

**Verification:**

- [ ] Targeted integration tests pass: `go -C backend test -tags=integration ./internal/adapters/postgres`
- [ ] Backend tests pass: `go -C backend test ./...`
- [ ] Integration tests verify same-day cancellation, previous-day rejection, order paid before midnight and cancelled after midnight rejection, already-cancelled rejection, concurrent cancellation race, missing order, preserved snapshots, retained queue number, no queue reuse, and no row deletion.

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

- [ ] `POST /api/pos/orders/{orderId}/cancel` requires the existing session middleware and accepts `orderId` as a positive base-10 string path value.
- [ ] The endpoint assumes the frontend has already shown the confirmation dialog; it does not create a separate confirmation state.
- [ ] Successful cancellation returns `200 OK` with updated canonical `PaidOrderDetail` and status `cancelled`.
- [ ] Malformed order IDs, including empty, non-numeric, zero, negative, decimal, or overflowed path values, return `400 Bad Request` with a stable malformed-ID error code.
- [ ] Well-formed but missing order IDs return `404 Not Found`.
- [ ] Already-cancelled or previous-day orders return `409 Conflict`.
- [ ] Unauthenticated requests return the existing protected-route `401 Unauthorized` shape.
- [ ] Response bodies never imply that the order was deleted.

**Verification:**

- [ ] Tests pass: `go -C backend test ./...`
- [ ] Integration tests pass: `go -C backend test -tags=integration ./...`
- [ ] HTTP tests cover unauthenticated access, malformed order ID, missing order, already-cancelled conflict, previous-day conflict, concurrent cancellation loser mapping, and successful cancellation.
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
- [ ] Docs clarify that backend persists paid orders only, recalculates totals server-side, requires `clientRequestId` idempotency keys, exposes `orderId` as a string, and reuses the canonical `PaidOrderDetail` response shape.
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
- [ ] Required per-line modifier groups are validated backend-side from item applicability; seeded Temperature and Sugar groups are covered by tests.
- [ ] Same menu item can appear as separate lines with different modifiers.
- [ ] Backend recalculates totals from persisted menu data.
- [ ] Create-order idempotency prevents duplicate paid orders on retry.
- [ ] Daily queue numbers reset by Asia/Jakarta business date and are unique under concurrency.
- [ ] Cancelled queue numbers are retained and never reused.
- [ ] Concurrent cancellation attempts produce one successful transition and conflict results for the rest.
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
| Duplicate paid orders from network retry | High | Require `clientRequestId`, store a request hash, and return the original `PaidOrderDetail` for same-key retries. |
| Frontend sends stale or manipulated prices | High | Do not accept prices or totals in request DTOs. Resolve menu slugs server-side and snapshot backend-calculated names/prices/totals. |
| Cross-item or cross-group modifier manipulation | High | Validate selected groups against the chosen item's attached groups and selected options against their submitted groups. Add wrong-group and unattached-group tests. |
| US-02 menu/database work is not merged when this starts | Medium | Treat US-02 as a prerequisite. Start with domain tests if database foundation is still in review, then rebase the schema/query work after US-02 lands. |
| Menu changes after an order is paid could alter historical tickets | Medium | Store immutable item and modifier snapshots on each paid order. Use menu foreign keys for traceability only. |
| Same-day cancellation around midnight is ambiguous | Medium | Use only the injected backend clock and Asia/Jakarta location. Add tests for just before and after midnight. |
| Concurrent cancellation race | Medium | Use an atomic `paid` to `cancelled` update and map zero-row updates to conflict/not-eligible results. |
| Endpoint scope drifts into Today's Orders or reporting | Medium | Keep list/detail/reprint/daily summary out of this plan except for returning create/cancel detail responses needed by this flow. |
| Server startup changes from auth-only to database-backed | Medium | Document `DATABASE_URL`, migration, and seed requirements. Keep database config errors explicit and covered by tests. |

## Resolved Decisions

- Order notes are capped at 500 characters in backend validation.
- Cashier menu/order APIs use stable slugs for menu item, modifier group, and modifier option selections. Modifier validation is generic: every required group attached to the selected item must be selected exactly once, unattached groups are rejected, and each option must belong to the submitted group.
- Cashier menu responses return arrays already sorted for display by `sort_order` and stable row identity.
- Create paid order requires a canonical lowercase UUID `clientRequestId`. Same-key same-request retries return the original `PaidOrderDetail`; same-key different-request retries return `409 Conflict`.
- Numeric database IDs remain internal except for the paid order's `orderId`, exposed as a positive base-10 API string with no sign and no leading zeroes. Malformed path IDs return `400`; well-formed missing IDs return `404`.
- The canonical `PaidOrderDetail` response is shared by create-order, cancel-order, and future paid-order detail endpoints.
- Cancelled orders retain their queue numbers permanently; later same-day orders never reuse cancelled queue numbers.
- Cancellation does not capture a cashier-entered reason in MVP. The backend stores status `cancelled` and `cancelledAt` only.
- `serve` does not auto-migrate or repair schema state. Operators and local setup must run `db migrate` before `serve`, then `db seed` before using cashier order entry.

## Parallelization Opportunities

- Tasks 1 and 2 can proceed in parallel once the order API contract is accepted.
- Task 3 can proceed in parallel with Task 1 if US-02 menu schema is stable.
- Task 4 should wait for Task 3.
- Tasks 5 through 7 should remain sequential because the HTTP endpoint depends on the application contract and PostgreSQL persistence behavior.
- Tasks 8 through 10 should remain sequential after paid-order creation is working.
- Task 11 can be drafted during Tasks 7 through 10, then finalized after endpoint names and runtime behavior are implemented.
