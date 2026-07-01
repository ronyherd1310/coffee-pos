# Code Review: Frontend Cashier Order Entry Revamp Changes

**Date:** 2026-07-01  
**Reviewer:** Codex  
**Scope:** Frontend changes in `frontend/` plus browser tests in `tests/e2e/`  
**Verdict:** Request changes

## Findings

### 1. Required: Hard-coded 11% tax conflicts with the revamp plan and spec

**Files:**

- `frontend/src/features/cashier/orderDraft.ts:81`
- `frontend/src/features/cashier/CashierOrderScreen.tsx:672`

`calculateDraftBreakdown` calculates `taxRp` as `Math.round(subtotalRp * 0.11)`, and the current order panel always renders the tax row. The revamp plan explicitly says tax/service charge is not implemented and must not hard-code `11%` until the shop policy is approved.

This can make the cashier collect or display `Rp19.980` while the backend persists an `Rp18.000` order total, because the order-create payload correctly omits client-calculated totals and the backend remains authoritative for pricing.

**Recommendation:** Return subtotal-only totals for now, with `taxRp` absent or zero, and update tests so they document that tax is not hard-coded from the screenshot.

### 2. Required: The current order panel shows a fixed fake draft reference

**File:** `frontend/src/features/cashier/CashierOrderScreen.tsx:601`

The current order panel renders `#ORD-0142` for every draft. The revamp plan calls draft references out as not implemented/open, and customer-facing post-payment references should continue to use backend queue numbers.

**Recommendation:** Remove the fixed draft reference, or replace it only after an explicitly approved client-only draft label behavior exists.

### 3. Required: Printable ticket timing bug is still present

**File:** `frontend/src/features/cashier/CashierOrderScreen.tsx:138`

The print handler calls `setShowPrintableTicket(true)` and immediately calls `window.print()`. Preact has not necessarily rendered the printable ticket before `print()` runs, so the print dialog can open on the paid detail view instead of the ticket.

**Recommendation:** Trigger printing from an effect after `showPrintableTicket` has rendered, or render the print view before invoking `window.print()`. Add a regression test that records whether the printable ticket exists inside the mocked `window.print()` call.

### 4. Consider: The visible grid/list affordance has no behavior

**File:** `frontend/src/features/cashier/CashierOrderScreen.tsx:328`

The catalog renders a visual grid/list toggle as `aria-hidden`, but there is no list-view behavior. The plan says not to render a grid/list toggle for this revamp.

**Recommendation:** Remove the visual affordance unless list view is implemented.

## Verification

Passed:

```sh
npm --prefix frontend test -- src/lib/pos.test.ts src/features/cashier/catalogView.test.ts src/features/cashier/orderDraft.test.ts src/features/cashier/CashierOrderScreen.test.tsx src/features/pos/ProtectedPosShell.test.tsx src/App.test.tsx
npm --prefix frontend run check
npm --prefix frontend run build
npm run test:e2e -- tests/e2e/auth-login.spec.ts tests/e2e/cashier-order-entry.spec.ts
```

## Summary

The frontend checks pass, but the tests currently encode behavior that conflicts with the revamp plan, especially the hard-coded 11% tax. Address the required findings before merging the frontend changes.
