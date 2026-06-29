# Spec Review: Small Coffee Shop POS MVP

**Date:** 2026-06-28
**Reviewer:** opencode
**Document:** `docs/specs/small-coffee-shop-pos-mvp-spec.md`

---

## Inconsistencies

### 1. Modifier selection UI contradicts per-item requirement

US-03 requires "Cashier must choose one Temperature option and one Sugar option **per item**" (line 288), but the wireframe (lines 358-364) shows a single global set of modifier radio buttons. There is no UI mechanism shown for assigning different modifiers to different items in the same order (e.g., 1 Americano Hot + 1 Latte Iced). This is the most critical gap in the spec — the cashier cannot correctly build multi-item orders with different modifiers as written.

### 2. Wireframe shows Print button before payment

The main cashier wireframe (line 346-365) simultaneously shows `[Confirm Paid]`, `[Print]`, and `Status: Not paid`. Both US-04 and US-08 explicitly state that printing is only allowed after payment is confirmed. The wireframe should show the Print button as disabled/greyed-out or absent until the order is paid.

### 3. Two conflicting post-payment screens

- US-07 wireframe (lines 423-431): "Payment confirmed" screen with `[Print Ticket]` and `[Start New Order]`
- US-06 wireframe (lines 384-395): "Paid Order #001" screen with `[Print Ticket]` and `[Cancel Order]`

These appear to be different views of the same post-payment state but are not reconciled. When does the cashier see one vs. the other? Are they the same screen at different moments?

### 4. "Daily: #001" header is ambiguous

The cashier wireframe (line 346) shows `Daily: #001` in the header. It is unclear if this represents today's order count, the next queue number, or something else. No requirement or user story defines this display.

### 5. Daily summary access scope is ambiguous

- Line 208: "Authenticated cashier can access the cashier order screen **and daily summary**"
- US-10: written as an owner/operator story

Since there is only one shared PIN, both cashier and owner have identical access. This should be stated explicitly rather than implied across two sections.

---

## Missing User Stories / Flows

### 6. No order lookup / listing flow

US-06 requires cancelling paid orders and reprinting tickets, but there is no user story or wireframe for **finding an existing paid order**. How does the cashier locate order #001 to cancel or reprint it? A "Today's Orders" list screen is needed. This is the second most critical gap — without it, post-payment actions are unreachable.

### 7. No "Start New Order" user story

The post-payment confirmation wireframe (line 431) shows a `[Start New Order]` button, but there is no corresponding user story. What happens to the current cart state? Is it simply cleared? This flow needs a user story.

### 8. No navigation user story

Line 208 states the authenticated cashier can access both the cashier order screen and the daily summary, but there is no user story describing how to navigate between them. No tab bar, header link, or button appears in any wireframe.

### 9. No Logout user story

Line 221 requires logout functionality ("Cashier can log out, which invalidates the current session"), but no user story or wireframe shows where the logout button lives (header? settings menu?) or what happens after logout.

### 10. No reprint ticket user story

The correction wireframe (line 391) shows a `[Print Ticket]` button for already-paid orders, implying reprinting is supported. No explicit user story covers reprinting a ticket for an existing paid order. US-08 only covers initial printing after payment.

### 11. No queue-number confirmation screen user story

The "Payment confirmed / Queue Number" screen (line 423-432) has a complete wireframe but no corresponding user story. It appears after payment confirmation but is not tied to any US.

### 12. No confirmation-for-destructive-actions flow

Confirming payment and cancelling an order are irreversible actions, yet the spec says nothing about a confirmation dialog or modal to prevent mis-taps. A mis-tap on "Cancel Order" would be disruptive even with soft-delete.

### 13. Empty order guard not specified

What happens if the cashier clicks "Confirm Paid" with zero items in the cart? No validation rule, error message, or disabled-button state is defined.

### 14. Multiple items with different modifiers

The spec does not clarify: can the cashier add the same drink twice with different modifiers (e.g., 1x Americano Hot + 1x Americano Iced in one order)? If so, each needs its own modifier selection, which the current single-set wireframe does not support.

---

## Missing Details

### 15. Session expiry mid-order

Line 336 states unpaid drafts are frontend-only. If the session expires while the cashier is mid-order, the draft is lost. The spec does not address:
- What the cashier sees when the session expires (redirect to login? timeout message?)
- Whether there is a session renewal/refresh mechanism
- What happens to an in-progress unpaid draft

### 16. No network error / offline handling

No mention of what happens if the backend is unreachable during payment confirmation, ticket printing, or order submission. For a POS optimizing for "fast counter operation," network resilience should be at least acknowledged.

### 17. Rupiah formatting not documented as a rule

Wireframe uses `Rp18.000` (dot thousands separator, no decimal), but the spec does not formalize the display format string. The code example uses `formatRupiah()` without showing the output format. This should be an explicit convention (e.g., `Rp` prefix, dot separator, no decimals).

### 18. Concurrent sessions with shared PIN

The spec does not address whether multiple browser tabs or devices can be logged in simultaneously with the same 6-digit PIN. For a single-cashier shop this may be fine, but it is unacknowledged.

---

## Priority Summary

| Priority | # | Finding |
|----------|---|---------|
| Critical | 1 | Modifier selection per-item UI gap |
| Critical | 6 | No order lookup/listing flow |
| High | 2 | Print button shown before payment |
| High | 3 | Conflicting post-payment screens |
| High | 12 | No confirmation dialogs for destructive actions |
| Medium | 4 | Ambiguous daily queue header |
| Medium | 5 | Daily summary access scope |
| Medium | 7 | Missing "Start New Order" user story |
| Medium | 8 | Missing navigation user story |
| Medium | 9 | Missing logout user story |
| Medium | 10 | Missing reprint user story |
| Medium | 11 | Missing queue confirmation user story |
| Medium | 13 | Empty order guard |
| Medium | 14 | Multiple items with different modifiers |
| Low | 15 | Session expiry mid-order |
| Low | 16 | Network error handling |
| Low | 17 | Rupiah formatting rule |
| Low | 18 | Concurrent sessions |

---

## Additional Findings (Second Review)

### Inconsistencies

#### 12. Queue number format mismatch across wireframes

- Main cashier wireframe (line 347): `Daily: #001` (with `#` prefix)
- Payment confirmed wireframe (line 428): `001` (plain number)
- Ticket wireframe (line 481): `QUEUE NO. 001`
- Daily summary wireframe (line 531): date only, no queue number shown

Pick one canonical format and apply consistently.

#### 13. Session expiry ambiguity

Line 220 and 791: "expires at the end of the Asia/Jakarta business day" — is this calendar midnight (00:00 WIB) or shop closing time? A shop operating 07:00–22:00 would have different operational needs than calendar-day reset. Define "business day" precisely.

#### 14. QRIS image path inconsistency

- Line 18 (Assumptions): `frontend/public/qris/static-qris.png`
- Line 175 (Project Structure): `public/qris/static-qris.png`
- Line 309 (US-05 Acceptance): `/qris/static-qris.png` (served path)

The served path `/qris/...` is correct for Caddy reverse proxy, but the source path in the repo should be consistent (likely `frontend/public/qris/static-qris.png`).

#### 15. Modifier price delta field exists but forced to zero

Line 269: "Modifier options have price delta Rp0 for MVP"
Lines 576, 618: Domain types include `PriceDeltaRp` / `priceDeltaRp` fields

Either remove the field from MVP types (since it's always zero) or clarify it's reserved for future use.

---

### Missing User Stories / Flows

| # | Missing Flow | Why It Matters |
|---|--------------|----------------|
| 16 | **Reprint ticket** | Printer jams/out of paper are common; no way to reprint a paid order's ticket |
| 17 | **View recent paid orders** | Cashier may need to verify/reprint before starting new order; "Paid Order #001" wireframe exists but no navigation to it |
| 18 | **Shift awareness** | Daily reset at midnight ≠ shift change; morning cashier sees yesterday's queue numbers if shop opens after midnight |
| 19 | **Offline/degraded mode** | What happens if backend is unreachable during payment confirmation? |
| 20 | **Printer error handling** | No flow for printer offline, paper out, or print failure |
| 21 | **Cancellation audit trail** | Stores timestamp but not *who* cancelled or *reason* |
| 22 | **Print daily summary (Z-report)** | Owner likely needs a physical Z-report for reconciliation |
| 23 | **Concurrent cashier sessions** | Single PIN shared — what if two cashiers use it simultaneously? |
| 24 | **CSRF protection** | Cookie-based auth + browser forms needs CSRF tokens (not mentioned) |
| 25 | **Health check contract** | Line 223 excludes health checks from auth but no spec for `/health` response format |

---

### Minor Gaps

- **Order ID vs Queue Number**: Wireframe shows "Paid Order #001" — is `#001` the queue number or a separate order ID? They diverge if an order is cancelled (queue number consumed but order remains).
- **Business day definition**: Specify whether "Asia/Jakarta business day" = calendar day (00:00–23:59) or operational hours (e.g., 06:00–05:59).
- **Migration strategy**: Commands exist but no deployment-time migration runbook (auto-run on startup? manual?).
