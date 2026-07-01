# Plan Review: US-03 to US-06 Frontend Cashier Order Entry

Reviewed plan: `docs/plan/us-03-us-06-frontend-cashier-order-entry-plan.md`

## Findings

1. Required: Resolve the paid-ticket scope mismatch before implementation.

   The plan says it covers US-03 through US-06 and only leaves unpaid print disabled, but it also excludes browser ticket rendering and print CSS (`lines 7`, `20`, `34`, `325`, `508`). The spec says US-04 lets the order "receive a queue number and be printed" and the Cashier Order Entry requirements say confirmation opens Paid Order Detail and "makes the ticket printable" (`docs/specs/small-coffee-shop-pos-mvp-spec.md` lines 348-357 and 397-401). Task 7's paid detail criteria do not include any paid `Print Ticket` action (`lines 397-404`), and Task 9 explicitly avoids ticket print coverage (`lines 471-476`).

   Fix by either narrowing this plan's claim/DoD to "order entry, payment confirmation, and cancellation only; printable tickets arrive in the printing slice" or by adding the minimal paid ticket action/rendering required to satisfy the spec. As written, the plan can be marked done while leaving a US-04 requirement unimplemented.

2. Required: Decide the QRIS asset policy before this is ready for coding.

   Task 5 requires `frontend/public/qris/static-qris.png` before QRIS UI is complete (`lines 322-323`), the architecture section says not to create a fake scannable production QRIS without explicit approval (`line 160`), and the same choice remains an open question (`line 505`). That leaves implementers with a blocker or an incentive to invent an unsafe placeholder.

   Fix by making the plan explicit: either require an owner-provided production QRIS image as an input before Task 5 starts, or approve a clearly non-production placeholder and define its exact visible labeling. Keep the DoD aligned with that decision so no private payment credential or misleading scannable QRIS image is accidentally committed.

3. Required: Keep header navigation scoped to this slice or intentionally update the existing shell/tests.

   Task 3 requires `New Order` and `Today's Orders` actions in the header (`line 255`), while this same plan lists Today's Orders as out of scope (`line 33`) and still leaves the action behavior open (`line 507`). The current shell exposes `New Order` and `Daily Summary`, and the current Playwright smoke test asserts `Daily Summary` remains visible (`frontend/src/features/pos/ProtectedPosShell.tsx` lines 45-48; `tests/e2e/auth-login.spec.ts` lines 49-51). The spec places Today's Orders reachability in US-07, not US-03 through US-06 (`docs/specs/small-coffee-shop-pos-mvp-spec.md` lines 465-479).

   Fix by removing Today's Orders from this plan's acceptance criteria, keeping the existing nav until the US-07 slice, or explicitly adding an inert/disabled placeholder with corresponding test updates. Do not make a later-slice navigation change an implicit requirement for this frontend order-entry slice.

## Open Questions Or Assumptions

- The note limit question (`line 506`) should be answered before Task 5 starts because it affects UI copy, validation, and tests. Matching the backend 500-character limit is the lowest-friction option unless the 120-character mockup limit is a deliberate product constraint.
- If Task 9 remains in scope, specify whether the Playwright smoke test should mock `/api/pos/menu` and `/api/pos/orders` like the current auth smoke test mocks auth, or run against a seeded backend. The current plan allows either, but the setup choice materially changes what the test proves.

## Verification

- Reviewed the plan against the MVP spec, current frontend shell, current e2e smoke test, and current backend order/menu handlers.
- No automated tests were run; this was a documentation review only.
