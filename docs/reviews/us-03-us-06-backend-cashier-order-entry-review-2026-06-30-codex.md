# Code Review: US-03 to US-06 Backend Cashier Order Entry

**Date:** 2026-06-30
**Reviewer:** Codex
**Scope:** Changes related to `docs/plan/us-03-us-06-backend-cashier-order-entry-plan.md`
**Verdict:** Request changes

## Findings

### 1. Unique-constraint handling in the order repository misclassifies internal consistency failures as idempotency conflicts

**Files:** `backend/internal/adapters/postgres/order_repository.go:52-66`, `backend/internal/adapters/postgres/order_repository.go:319-321`

`CreatePaidOrder` treats every PostgreSQL `23505` as an idempotency race and falls back to `getIdempotency`. That is only correct for `orders_client_request_id_key`. The same branch will also catch uniqueness failures on `(business_date, queue_number)` and the line/modifier snapshot uniqueness constraints.

That breaks the plan contract. The plan explicitly says a uniqueness failure on `(business_date, queue_number)` must be handled as an internal consistency error, not as a successful duplicate or `idempotency_conflict`. In the current code, a queue-number collision with no matching `clientRequestId` row becomes `409 idempotency_conflict`, which hides the real failure mode and makes diagnosis harder.

The fix is to inspect the violated constraint name and only run the idempotency fallback for `client_request_id`. Other uniqueness failures should surface as internal errors.

### 2. The Postgres integration suite does not exercise the order repository at all

**Files:** `backend/internal/adapters/postgres/order_repository.go:26-262`, `backend/internal/adapters/postgres/migrations_integration_test.go:10-37`, `backend/internal/adapters/postgres/menu_repository_integration_test.go:14-69`

`go -C backend test -tags=integration ./...` passes, but the only adapter-level integration tests in this slice are for migrations and menu seeding. There is no `order_repository_integration_test.go`, and nothing in the existing integration files touches `CreatePaidOrder`, queue allocation, idempotency fallback, or `CancelPaidOrder`.

That leaves the highest-risk behavior in this change unverified against a real database:

- sequential and concurrent queue allocation
- same-key retry returning the existing order
- same-key different-payload conflict behavior
- persisted line/modifier snapshots
- same-day cancellation vs previous-day rejection

Given how much of the implementation depends on transactional SQL behavior, this is a required gap, not a nice-to-have.

### 3. The order use-case and HTTP tests still cover only a narrow slice of the contract

**Files:** `backend/internal/app/orders/usecases_test.go:12-150`, `backend/internal/adapters/http/order_handlers_test.go:18-115`

The current tests prove one happy-path create, one missing-required-modifier case, malformed `clientRequestId`, malformed cancel `orderId`, and basic create/cancel success responses. They do not cover a large part of the behavior the plan calls out for this slice.

Missing cases include:

- wrong-group option, unattached group, duplicate group, unknown/inactive item, empty active menu
- QRIS success path and same item with different modifiers
- repository failure and cancel not-found/not-cancellable mappings
- same-key retry returning `200 OK`
- same-key different-payload conflict returning `409`
- malformed JSON and null/wrong-type request fields across create-order payload parsing

This matters because the implementation is doing a lot of manual request validation and semantic menu resolution. Without those cases in tests, the current green suite is weaker than it looks.

## Assumptions

- I reviewed the current worktree, including untracked order-related files, as the implementation under review.
- I treated the plan document as the contract for this slice and checked the code against that contract.

## Verification

```sh
go -C backend test ./...
go -C backend test -tags=integration ./...
go -C backend vet ./...
```

All three commands passed in the current workspace.
