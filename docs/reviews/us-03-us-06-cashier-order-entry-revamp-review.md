# Code Review: US-03 to US-06 Cashier Order Entry Revamp Implementation

**Date:** 2026-07-01
**Reviewer:** Codex
**Document:** `docs/plan/us-03-us-06-cashier-order-entry-revamp-plan.md`
**Status:** No Critical Findings

---

## Findings

### Severity: High

#### 1. `Print Ticket` still invokes the browser print flow before the printable ticket is rendered

**File:** `frontend/src/features/cashier/CashierOrderScreen.tsx:138-140`

The paid-order `Print Ticket` handler still calls `setShowPrintableTicket(true)` and immediately calls `window.print()`. In Preact, that state update is not rendered synchronously before the next line runs, so the browser print dialog can open while the paid detail screen is still the rendered view and before the `.printable-ticket` region exists.

This violates the revamp plan's Task 15/complete-flow expectation that paid detail is printable after payment, and the spec requirement that confirming payment "makes the ticket printable." The existing component test only asserts that the ticket appears after the click and that `window.print` was called; it does not assert that the ticket existed at print invocation time.

**Recommendation:** Trigger printing from an effect after `showPrintableTicket` has rendered, or render a dedicated print view before calling `window.print()`. Add a regression test that records whether the printable ticket exists inside the mocked `window.print()` call.

### Severity: Medium

#### 2. Create/cancel server outages are still classified as unexpected errors instead of unavailable

**File:** `frontend/src/lib/pos.ts:82-100`, `frontend/src/lib/pos.ts:119-133`, `frontend/src/features/cashier/CashierOrderScreen.tsx:778-803`

`getCashierMenu` maps non-OK server responses to `unavailable`, but `createPaidOrder` and `cancelPaidOrder` only map known business errors. Unmapped `5xx` responses fall through to `unexpected`. For order creation this shows "Start a new order and try again" even though the draft and `clientRequestId` are still recoverable and the right action is to retry when the service recovers.

This weakens the revamp plan's retry/error-state behavior around payment confirmation and cancellation. It also leaves a test gap: `pos.test.ts` covers network failures but not `503` responses for create/cancel.

**Recommendation:** Map `response.status >= 500` from create and cancel to `unavailable`, keep `unexpected` for malformed/unknown contract responses, and add client tests for `503` on both endpoints.

### Severity: Low

#### 3. Modal dialogs still do not trap focus while using `aria-modal`

**File:** `frontend/src/features/cashier/ConfirmPaymentDialog.tsx:31-93`

The payment dialog moves initial focus into the modal and returns focus on cancel, but it does not trap tab focus or make the page behind the dialog inert. Because the dialog is marked `aria-modal="true"`, keyboard and assistive-technology users are told they are in a modal while focus can still escape to controls behind the backdrop.

This is not a core payment correctness bug, but it is a modal accessibility gap in the revamp's focused payment flow.

**Recommendation:** Add a small focus trap or inert the background while the payment/cancel dialogs are open. Add keyboard coverage that tabs through modal controls and verifies focus remains inside the active dialog.

---

## Positive Observations

- Backend seed data now includes the approved Coffee, Tea, Snacks, and Seasonal categories plus the 12 MVP catalog items from the spec.
- Menu display metadata is persisted, exposed by `GET /api/pos/menu`, and consumed by the frontend parser with malformed metadata rejection.
- The frontend catalog helpers keep search/category/quick-filter/sort behavior pure and covered by focused tests.
- The order-create payload still excludes image paths, display flags, client totals, tax, queue number, status, and timestamps.
- QRIS confirmation now appears in the payment modal, and cancellation from that modal preserves the draft order state.
- US-06 same-day cancellation behavior is preserved in component and Playwright coverage.

## Verification Commands

Critical-fix verification result: no Critical findings were present in this review, so no source changes were made and no implementation verification commands were required for Critical fixes.

Original review verification:

```bash
go -C backend test ./internal/domain/menu ./internal/app/menu ./internal/adapters/http ./internal/adapters/postgres
npm --prefix frontend test -- src/lib/pos.test.ts src/features/cashier/catalogView.test.ts src/features/cashier/orderDraft.test.ts src/features/cashier/CashierOrderScreen.test.tsx src/features/pos/ProtectedPosShell.test.tsx src/App.test.tsx
npm --prefix frontend run check
npm --prefix frontend run build
go -C backend test ./...
npm run test:e2e
go -C backend test -tags=integration ./...
```

All commands passed.

Podman Compose runtime verification from Task 16 was not rerun during this review. The plan records that it passed during implementation, and this review covered backend unit/integration tests, frontend unit/type/build checks, and Playwright e2e.

## Summary

No Critical findings were found, so no code changes were needed under the requested scope. The implementation substantially matches the revamp plan and spec direction, but the printable-ticket timing bug remains a required fix before the flow should be considered production-ready. The create/cancel error mapping and modal focus handling should also be addressed to close the remaining review gaps.

**Verdict:** Request changes for the High finding.
