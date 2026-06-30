# Code Review: US-03 to US-06 Backend Cashier Order Entry Implementation

**Date:** 2026-06-30
**Reviewer:** opencode
**Document:** `docs/plan/us-03-us-06-backend-cashier-order-entry-plan.md`
**Status:** No Critical Findings

---

## Verification Commands

```bash
go -C backend test ./...
go -C backend vet ./...
```

All unit-level commands pass. Integration tests (`-tags=integration`) could not be run because no order-specific integration tests exist (see Finding #1).

2026-06-30 Critical-finding follow-up: re-read this review and found no findings explicitly marked `Critical`; the review contains only Medium and Low findings. No source changes or behavior tests were needed for this follow-up, and no verification commands were rerun because there were no Critical fixes to validate.

---

## Findings

### Severity: Medium

#### 1. No integration tests for the order repository

**Files:** `backend/internal/adapters/postgres/order_repository.go` (entire file)

The plan requires integration tests covering queue allocation under concurrency, idempotency retry/conflict behavior, Cash and QRIS persistence with correct totals, same-day cancellation, previous-day rejection, and preserved snapshots (Tasks 6 and 9, lines 368-374, 485-488). None of these exist. The menu repository has integration tests (`menu_repository_integration_test.go`, `migrations_integration_test.go`), but no equivalent file exists for orders.

This is the most significant gap. The PostgreSQL order repository contains non-trivial transactional logic — queue allocation via atomic upsert, idempotency fallback on unique violations, and cancellation via conditional update — that unit tests with fakes cannot exercise. A concurrency bug in queue allocation or an idempotency edge case would only surface against a real database.

**Recommendation:** Create `backend/internal/adapters/postgres/order_repository_integration_test.go` covering:
- Sequential queue number allocation for same business date
- Concurrent queue allocation produces unique sequential numbers
- Same `clientRequestId` + same payload returns existing order
- Same `clientRequestId` + different payload returns idempotency conflict
- Cash and QRIS order creation with correct line/modifier snapshots
- Same-day cancellation, already-cancelled rejection, previous-day rejection
- Preserved queue number after cancellation; no reuse

#### 2. Application-layer use case tests have significant coverage gaps

**File:** `backend/internal/app/orders/usecases_test.go`

The plan specifies unit tests for: same item with different modifiers, unattached modifier group, wrong-group option, inactive item, fully inactive menu, invalid payment method, empty cart, repository failure, and clock/date boundaries (Task 5, line 338). The existing tests cover only 5 scenarios:

- Successful Cash order (`TestServiceCreatePaidOrderResolvesMenuAndCalculatesTotals`)
- Missing required modifier (`TestServiceCreatePaidOrderRejectsMissingRequiredModifier`)
- Idempotency conflict mapping (`TestServiceCreatePaidOrderMapsIdempotencyConflict`)
- Malformed `clientRequestId` (`TestServiceCreatePaidOrderRejectsMalformedClientRequestID`)
- Cancel delegation (`TestServiceCancelPaidOrderDelegatesSameDayBusinessDate`)
- Malformed order ID on cancel (`TestServiceCancelPaidOrderRejectsMalformedOrderID`)

Missing coverage:
- QRIS payment method success path
- Same menu item as two lines with different modifiers (the spec's core scenario at line 326)
- Unattached modifier group (group slug not attached to the item)
- Wrong-group option (option belongs to a different group)
- Duplicate modifier group on a single line
- Inactive/unknown menu item slug
- Fully empty menu (no items in menu response)
- Repository failure / error propagation
- Clock boundary: order paid at 23:59 Jakarta, cancelled after 00:00
- Cancel result mapping for not-found and not-cancellable

**Recommendation:** Add the missing test cases listed above to `usecases_test.go`.

#### 3. HTTP handler tests have significant coverage gaps

**File:** `backend/internal/adapters/http/order_handlers_test.go`

The plan specifies HTTP tests for: unauthenticated access, malformed JSON, unknown fields at every level, forbidden fields, missing/null/wrong-typed `clientRequestId`, null and wrong-typed cases for every request field, empty cart, missing required modifiers, invalid payment method, inactive menu item, Cash success, QRIS success, same-key retry, same-key conflict, and handler error mapping (Task 7, line 409). The existing tests cover only:

- Unauthenticated access (`TestCreatePaidOrderRequiresAuthentication`)
- Unknown and forbidden fields (`TestCreatePaidOrderRejectsUnknownAndForbiddenFields`)
- Created detail response (`TestCreatePaidOrderReturnsCreatedDetail`)
- Cancel malformed ID (`TestCancelPaidOrderRejectsMalformedOrderID`)
- Cancel success (`TestCancelPaidOrderReturnsUpdatedDetail`)

Missing coverage:
- Malformed JSON body
- Null `clientRequestId`, non-string, empty string, malformed UUID
- Null/missing `paymentMethod`
- Null `note`, non-string `note`
- Null `lines`, non-array `lines`, empty `lines`
- Null `menuItemSlug`, missing `menuItemSlug`
- Null/missing `quantity`, non-integer `quantity`, zero, >99
- Null `modifiers`, non-array `modifiers`, null `groupSlug`, null `optionSlug`
- Cash vs. QRIS response differentiation
- Same-key retry returning `200 OK` with original detail
- Same-key different payload returning `409 Conflict`
- Service error mapping (invalid order → 422, not found → 404, not cancellable → 409)

**Recommendation:** Add the missing HTTP test cases listed above.

### Severity: Low

#### 4. `validOrderID` is duplicated across packages

**Files:** `backend/internal/domain/orders/order.go:272-278`, `backend/internal/app/orders/usecases.go:301-310`, `backend/internal/adapters/http/order_handlers.go:327-337`

The same order ID validation logic (positive base-10 string, no leading zeros) is implemented independently in three packages. The domain and application versions check `value[0] == '0'` which panics on empty strings, but the empty check is before it so it is safe. However, the domain version uses `strconv.ParseInt` (rejects overflow), while the application and HTTP versions use a character-range loop (allows values exceeding `int64` range).

This inconsistency means the HTTP layer accepts order IDs that the domain layer would reject, and the application layer accepts order IDs that `strconv.ParseInt` in the repository would reject.

**Recommendation:** Consolidate into the domain package (`domain/orders.ValidOrderID`) and reuse from application and HTTP layers.

#### 5. `resolvedDraft` returns `ErrInvalidOrder` for all validation failures

**File:** `backend/internal/app/orders/usecases.go:195-227`

The `resolveDraft` and `resolveLine` functions return `ErrInvalidOrder` for unknown item slugs, unknown modifier groups, unknown options, unattached groups, missing required groups, and duplicate groups. This loses specificity — the HTTP layer maps all of these to `422 Unprocessable Entity` with code `invalid_order`, but the caller cannot distinguish between "unknown item" and "missing required modifier."

The plan does not require distinct error codes for each sub-case, so this is acceptable. However, the uniform error mapping makes debugging harder and may complicate future frontend error messages.

**Recommendation:** Acceptable for MVP. Consider introducing sub-error types if the frontend needs to show different messages per failure reason.

#### 6. `GetPaidOrder` does not use sqlc

**File:** `backend/internal/adapters/postgres/order_repository.go:90-171`

The `GetPaidOrder` method uses raw SQL via `db.QueryContext` with a hardcoded `paidOrderDetailSQL` constant, while the rest of the order queries are defined in `queries/orders.sql` and would be generated by sqlc. The `orders.sql` file contains a `GetPaidOrder` query that matches this raw SQL. This is a minor inconsistency — the generated sqlc code exists but is not used.

**Recommendation:** Use the sqlc-generated `GetPaidOrder` query instead of the raw SQL constant. This ensures the query stays in sync with the schema and benefits from sqlc type checking.

#### 7. `checkedMul` does not check for negative overflow

**File:** `backend/internal/domain/orders/order.go:287-295`

The `checkedMul` function only checks for positive overflow (`left > math.MaxInt64/right`). Since prices and quantities are always non-negative in this domain, this is safe. However, if a negative `PriceDeltaRp` were ever introduced (currently rejected by domain validation), the multiplication could silently produce incorrect results. The domain validation prevents this today.

**Recommendation:** Acceptable as-is given current invariants. Add a brief comment if desired.

---

## Positive Observations

### Architecture and Design

- **Clean hexagonal architecture:** Domain rules (`internal/domain/orders`) have zero infrastructure dependencies. Application ports (`internal/app/orders/ports.go`) define `MenuReader`, `Repository`, and `Clock` interfaces. Adapters implement those interfaces. HTTP handlers depend only on the application service.
- **Domain invariants are enforced server-side:** Client-supplied prices, totals, queue numbers, statuses, and timestamps are never accepted in create-order requests. The backend resolves menu slugs, recalculates totals, derives business dates, and allocates queue numbers.
- **Idempotency is correctly designed:** `clientRequestId` is required, stored as a UUID, and used as the idempotency key with a request hash. Same-key retries return the original order; different payloads conflict.
- **Queue allocation is transaction-safe:** The atomic upsert into `daily_queue_counters` within the same transaction as order insertion prevents duplicate queue numbers under concurrency.
- **Cancellation preserves audit history:** Paid orders are never deleted; cancellation only transitions status and sets `cancelled_at`. Queue numbers are never reused.

### Domain Rules

- **Order domain** (`domain/orders/order.go`): Correctly validates payment methods (cash/qris only), quantities (1-99), notes (≤500 chars), non-negative prices, no duplicate modifier groups, and total overflow. Line total calculation uses checked arithmetic.
- **Business date derivation** (`domain/orders/order.go:46-52`): Correctly uses injected clock and Asia/Jakarta location. Falls back to UTC when location is nil.
- **Cancellation** (`domain/orders/order.go:193-202`): Correctly rejects cancellation of already-cancelled orders and preserves all original order data.

### Application Layer

- **Menu resolution** (`app/orders/usecases.go:195-281`): Correctly resolves menu slugs against the read model, validates required modifier groups, detects duplicate groups, and verifies option membership in the correct group.
- **Request hash** (`app/orders/usecases.go:292-299`): SHA-256 hash of the full canonical request for idempotency comparison.
- **Error mapping** (`app/orders/usecases.go:185-192`): Cancel results are correctly mapped to application errors before reaching the HTTP layer.

### Persistence Layer

- **Database schema** (`migrations/000002_create_order_schema.sql`): Correctly enforces `(business_date, queue_number)` uniqueness, `client_request_id` uniqueness, status/payment_method check constraints, positive quantities, non-negative totals, and the `paid`/`cancelled` status ↔ `cancelled_at` invariant.
- **Queue allocation** (`postgres/order_repository.go:190-203`): Atomic upsert with `ON CONFLICT` correctly increments the counter and returns the new value within a transaction.
- **Idempotency fallback** (`postgres/order_repository.go:53-66`): On unique violation during insert, correctly re-checks idempotency and returns the existing order or a conflict.
- **Cancellation** (`postgres/order_repository.go:229-262`): Conditional update with `AND status = 'paid'` prevents double-cancellation. Zero-row result correctly distinguishes between not-found and not-cancellable.

### HTTP Layer

- **Strict JSON parsing** (`http/order_handlers.go:92-126`): Manual field-level parsing with forbidden field detection, unknown field rejection, null type checking, and nested field validation. This matches the plan's strict rejection requirements.
- **Response shape** (`http/order_handlers.go:253-317`): Canonical `PaidOrderDetail` response is reused for create and cancel endpoints.
- **Body size limit** (`http/order_handlers.go:13`): 16KB limit prevents oversized payload attacks.

### Test Coverage

- Domain tests cover: total calculation, invalid input rejection (empty order, invalid payment, quantity bounds, missing names, negative prices, duplicate groups, note length, overflow), business date derivation, and cancellation data preservation.
- Application tests cover: menu resolution, required modifier validation, idempotency conflict, malformed clientRequestId, cancel delegation, and malformed order ID.
- HTTP tests cover: authentication requirement, unknown/forbidden field rejection, created response, cancel malformed ID, and cancel success.

---

## Summary

The implementation follows the plan's architecture and domain rules correctly. The hexagonal layering is clean, domain invariants are enforced server-side, and the persistence layer correctly handles transactional queue allocation and idempotency.

The primary gap is **test coverage**: no integration tests exist for the order repository, and both application and HTTP handler tests are missing many of the scenarios specified in the plan. These gaps mean the most critical behavioral properties (concurrent queue allocation, idempotency under race conditions, full HTTP contract validation) have no automated verification.

Critical-finding follow-up fix summary: no findings were explicitly marked Critical, so no implementation changes were made. The review status was updated from `Ready to be checked` to `No Critical Findings`.

**Verdict:** Approved with findings. Integration tests and expanded unit test coverage should be added before the implementation is considered complete per the plan's acceptance criteria.
