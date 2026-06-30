# Plan Review: US-03 to US-06 Backend Cashier Order Entry

**Date:** 2026-06-30
**Reviewer:** Codex
**Document:** `docs/plan/us-03-us-06-backend-cashier-order-entry-plan.md`

---

## Required Changes

### 1. Modifier validation is partly hardcoded and still leaves real applicability gaps

**File:** `docs/plan/us-03-us-06-backend-cashier-order-entry-plan.md:37,140-147,203-205`

The plan says order creation resolves generic menu and modifier slugs against persisted menu data, and the read model is built around item-specific required modifier groups. But Task 5 then hardcodes the rule to "exactly one Temperature option and one Sugar option" per line.

That is too specific for the architecture the plan describes, and it still misses two important validation rules:

- rejecting a known modifier group that exists globally but is not attached to the selected menu item
- rejecting an option slug that exists, but belongs to a different modifier group than the submitted `groupSlug`

As written, an implementation can satisfy the literal acceptance criteria while still accepting cross-group or cross-item modifier combinations.

Add acceptance criteria and tests for:

- each required modifier group attached to the selected menu item must be present exactly once
- groups not attached to the selected menu item are rejected
- selected option must belong to the submitted group
- the seeded menu specifically proves the Temperature and Sugar case, without hardcoding those names into the generic rule

### 2. The create-order HTTP contract is incomplete for forbidden fields and typed malformed payloads

**File:** `docs/plan/us-03-us-06-backend-cashier-order-entry-plan.md:38,267-270,279`

The plan says the request "does not accept" client-supplied prices, totals, queue numbers, statuses, paid timestamps, or business dates, and that malformed or structurally invalid requests return `400`. In Go, that is still ambiguous unless the plan states whether unknown fields are rejected or silently ignored.

Without an explicit rule here, a handler could ignore `totalRp`, `queueNumber`, or `unitPriceRp` and still claim compliance. That weakens both the client contract and the backend validation story.

Add acceptance criteria for:

- rejecting unknown top-level, line-level, and modifier-level JSON fields
- `paymentMethod: null`, missing `paymentMethod`, and non-string `paymentMethod`
- `lines: null`, non-array `lines`, empty objects inside `lines`, and non-numeric or zero `quantity`
- `note: null` or non-string `note`
- stable error-code mapping for forbidden fields versus semantic validation failures

### 3. `orderId` format and cancellation error mapping are still ambiguous

**File:** `docs/plan/us-03-us-06-backend-cashier-order-entry-plan.md:208,309,366-377,453`

The plan consistently refers to an internal `orderId`, but never defines its type or wire format. Task 10 also says malformed IDs return `400` or `404` "consistently," which leaves the actual contract open.

This affects the route parser, repository types, test fixtures, and frontend integration. It also makes line 453 incomplete, because "numeric database IDs remain internal except for the paid order's internal `orderId`" still does not say whether the API exposes an integer, stringified integer, UUID, or something else.

Add acceptance criteria for:

- the concrete `orderId` API type and format
- one rule for parse failure versus not-found behavior, for example:
  - malformed path value -> `400 Bad Request`
  - well-formed but missing order -> `404 Not Found`
- matching create-order response typing and cancel-order path typing

### 4. Queue and cancellation concurrency semantics are underspecified

**File:** `docs/plan/us-03-us-06-backend-cashier-order-entry-plan.md:234-240,341-345,430-431`

The plan covers queue-number uniqueness during order creation, but it does not define two adjacent behaviors that matter once cancellation exists:

- whether cancelling queue `5` can ever free or reuse queue `5` for a later same-day order
- what happens when two cancellation requests race on the same paid order

The current acceptance criteria would allow an implementation that preserves the cancelled row yet still reuses queue numbers later, or an implementation where two concurrent cancel requests both appear successful.

Add acceptance criteria and tests for:

- cancelled orders retain their original queue number permanently
- later same-day orders continue with the next queue number and never reuse cancelled numbers
- concurrent cancellation of one order yields exactly one successful transition and one conflict/not-eligible result

### 5. The cashier menu contract is missing an ordering guarantee

**File:** `docs/plan/us-03-us-06-backend-cashier-order-entry-plan.md:145,175`

The plan requires the menu API to return categories, items, groups, and options, but it never states whether the response is already ordered for display or whether sort metadata is returned. The existing menu schema already carries `sort_order`, and the cashier screen needs deterministic button ordering.

Add acceptance criteria for one of these two contracts:

- arrays are returned in display order, or
- explicit sort fields are returned for categories, items, groups, and options

Without that, frontend and backend can both be "correct" while producing unstable cashier layouts.

---

## Missing Edge Cases

- Duplicate submit or network retry on `POST /api/pos/orders` after the cashier has already confirmed payment. The plan should either define an idempotency strategy or explicitly mark duplicate-create protection as a non-goal for this slice.
- A known modifier group that exists globally but is not valid for the selected item.
- A known option slug submitted under the wrong `groupSlug`.
- Empty active menu state: `/api/pos/menu` returning no active items, and `POST /api/pos/orders` against a fully inactive menu.
- Extremely large `quantity` values and large aggregate totals. The plan currently specifies positive quantities only, not upper bounds or integer-size assumptions.
- Cancellation at `23:59` versus `00:00` Asia/Jakarta, including an order paid just before midnight and cancelled just after midnight.
- Two create-order requests submitted nearly simultaneously for the same cart by the same cashier session.
- Null and wrong-type JSON cases for every request field, not just malformed JSON syntax.

---

## Resolved In Review Follow-Up

- Duplicate paid-order creation on client retry is not acceptable for this slice. The plan should require idempotency protection for create-order requests, such as a client request ID or idempotency key, and define the persistence/response behavior for retries.
- The backend should define a canonical `PaidOrderDetail` response now and reuse it across create-order, cancel-order, and future order-detail endpoints.
- Recommended canonical `PaidOrderDetail` fields:
  - `orderId` as an API string field
  - `queueNumber`
  - `businessDate`
  - `status`
  - `paymentMethod`
  - `paidAt`
  - nullable `cancelledAt`
  - nullable `note`
  - `totalRp`
  - `lines[]` with persisted `menuItemSlug`, `menuItemName`, `unitPriceRp`, `quantity`, `lineTotalRp`, and `modifiers[]`
  - `modifiers[]` with persisted `groupSlug`, `groupName`, `optionSlug`, `optionName`, and `priceDeltaRp`

These decisions close the review's earlier open questions. The remaining work is to encode them explicitly in the plan acceptance criteria and HTTP contract.

---

## Verdict

**Not ready to implement as written.** The plan is directionally aligned with the MVP spec, and the follow-up decisions now settle idempotency and the canonical paid-order detail response. It still needs tighter plan text around generic modifier validation, create-order request parsing, `orderId` typing and parsing, cancellation concurrency, menu ordering, and the exact idempotency contract. Those gaps are small enough to fix in the plan before implementation, and they will prevent avoidable handler and repository drift.
