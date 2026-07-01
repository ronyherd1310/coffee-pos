# Code Review: US-03 to US-06 Frontend Cashier Order Entry Implementation

**Date:** 2026-07-01
**Reviewer:** Codex
**Document:** `docs/plan/us-03-us-06-frontend-cashier-order-entry-plan.md`
**Status:** No Critical Findings

---

## Findings

### Severity: High

#### 1. `Print Ticket` invokes the browser print flow before the printable ticket is rendered

**File:** `frontend/src/features/cashier/CashierOrderScreen.tsx:118-121`, `frontend/src/features/cashier/CashierOrderScreen.tsx:721-745`

The paid `Print Ticket` action calls `setShowPrintableTicket(true)` and immediately calls `window.print()`. In a real browser, the state update has not rendered `.printable-ticket` yet when `window.print()` runs, so the print dialog is opened against the paid detail screen without the minimal ticket view. A targeted Playwright runtime check confirmed `document.querySelector(".printable-ticket")` was `false` inside the overridden `window.print()`, and the ticket appeared only afterward.

This violates the plan's Task 7 requirement that the paid print action "opens or renders a minimal printable ticket view from the current paid-order detail and invokes the browser print flow," and the spec requirement that confirming payment makes the ticket printable.

**Recommendation:** Split the print flow into two steps: set the printable-ticket state first, then invoke `window.print()` from an effect that runs after the ticket is present, or render the ticket synchronously before printing through a dedicated print view. Add a browser-level test that clicks paid `Print Ticket` and asserts the printable ticket exists at print invocation time.

### Severity: Medium

#### 2. Create/cancel API clients misclassify backend outages as unexpected errors

**File:** `frontend/src/lib/pos.ts:82-100`, `frontend/src/lib/pos.ts:119-133`, `frontend/src/features/cashier/CashierOrderScreen.tsx:813-838`

`getCashierMenu` maps non-OK server responses to `unavailable`, but `createPaidOrder` and `cancelPaidOrder` return `unexpected` for unmapped non-OK responses such as `500` or `503`. The plan requires unavailable failures to be explicit frontend states for order creation and cancellation. In the UI this changes the guidance from retryable service-unavailable copy to "Start a new order and try again" for create failures, even though the draft is still recoverable and the likely action is retry after service recovery.

**Recommendation:** Map `5xx` responses from create and cancel to `unavailable`, keep `unexpected` for malformed/unknown contract responses, and add API client tests for `503` on both endpoints.

### Severity: Low

#### 3. Modal dialogs do not trap focus while marked `aria-modal`

**File:** `frontend/src/features/cashier/CashierOrderScreen.tsx:597-641`, `frontend/src/features/cashier/CashierOrderScreen.tsx:762-787`

The confirm-payment and cancel-order dialogs move focus into the dialog, which is good, but they do not trap focus or make the background inert. Keyboard users can tab from the modal controls into the cashier screen behind the backdrop while the dialog is still open. Because the dialogs use `aria-modal="true"`, assistive technology users are told the modal is active even though the DOM does not enforce modal interaction.

**Recommendation:** Add a small focus trap or mark the rest of the app inert while a dialog is open. At minimum, add keyboard tests that tab through the dialogs and verify focus stays within the modal.

---

## Positive Observations

- The QRIS asset at `frontend/public/qris/static-qris.png` is byte-for-byte identical to `docs/screen-captures/05.qris-payment.png`.
- The order-entry UI keeps unpaid drafts client-side and only calls `POST /api/pos/orders` from the confirm-payment dialog.
- The create-order payload excludes client-calculated prices, totals, queue number, status, timestamps, and other server-owned fields.
- Unit tests cover menu loading, draft validation, modifier selection, QRIS display, payment confirmation, UUID reuse on retry, paid detail, cancellation, and the mocked browser smoke path.

## Verification Commands

```bash
npm --prefix frontend test
npm --prefix frontend run check
npm --prefix frontend run build
npm run test:e2e
cmp -s docs/screen-captures/05.qris-payment.png frontend/public/qris/static-qris.png
```

All commands passed.

Additional targeted browser check:

```bash
npm --prefix frontend run dev -- --host 127.0.0.1
# Playwright script overrode window.print() and recorded whether .printable-ticket existed at print time.
```

Observed result:

```json
{"states":[false],"existsAfter":1}
```

This confirms Finding #1: the ticket is rendered after the print call, not before it.

## Summary

No findings in this review are explicitly marked Critical. Per the requested remediation scope, no source changes were made for the High, Medium, Low, or follow-up items.

**Fix Summary:** No code changes were needed because there were no Critical findings to fix.

**Verification Results:** Confirmed with `rg -n "Critical|Severity:|Status:" docs/reviews/us-03-us-06-frontend-cashier-order-entry-review.md` that the review contains High, Medium, and Low findings only. No frontend or backend test commands were run because no behavior or source code changed.

**Verdict:** No Critical Findings.
