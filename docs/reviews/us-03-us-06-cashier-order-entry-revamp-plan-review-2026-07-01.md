# Plan Review: US-03 to US-06 Cashier Order Entry Revamp

Reviewed plan: `docs/plan/us-03-us-06-cashier-order-entry-revamp-plan.md`

## Findings

1. Required: Resolve the backend seed and metadata dependency before treating this as an implementation-ready US-03 plan.

   The plan frames the revamp as frontend-first and treats expanded seeded catalog data and filter metadata as optional follow-up backend work (`docs/plan/us-03-us-06-cashier-order-entry-revamp-plan.md` lines 82-86 and 109-118). That does not match the current spec. The cashier order-entry section requires category tabs for All, Coffee, Tea, Snacks, and Seasonal, plus quick filters backed by seeded item metadata where available (`docs/specs/small-coffee-shop-pos-mvp-spec.md` lines 365-369 and 416-421). The seed requirements also already call for those categories, a larger starter catalog, and optional display metadata such as image paths and filter flags (`docs/specs/small-coffee-shop-pos-mvp-spec.md` lines 314-319 and 323-347). The current backend seed still only provides Coffee with Americano and Latte (`backend/internal/domain/menu/seed.go` lines 8-38).

   Fix by either adding backend seed/contract work as an explicit dependency of this revamp plan, or narrowing the plan and its definition of done to a visual shell that does not claim full US-03 acceptance. As written, the plan can finish with a UI that still cannot satisfy the spec's seeded catalog behavior.

2. Required: Do not ship a non-functional grid/list toggle as an accepted revamp control.

   Task 4 allows the grid/list affordance to render even if list behavior is deferred (`docs/plan/us-03-us-06-cashier-order-entry-revamp-plan.md` lines 211-220), and the open questions still leave that undecided (`docs/plan/us-03-us-06-cashier-order-entry-revamp-plan.md` line 531). In the spec, image-card view is required, while list view is optional only if the UI actually supports it; unsupported sort or view options must not be shown (`docs/specs/small-coffee-shop-pos-mvp-spec.md` lines 370 and 421).

   Fix by choosing one path in the plan: implement list view in this slice, or remove/disable the toggle from scope and from acceptance criteria. Leaving a visible dead control in a cashier flow is not a safe default.

3. Required: Add explicit cancellation verification for US-06 after the paid-detail and modal refactor.

   The plan says current paid-order detail and cancellation behavior will be preserved (`docs/plan/us-03-us-06-cashier-order-entry-revamp-plan.md` line 88), but the task list does not give cancellation its own acceptance or regression checkpoint after the paid-detail, shell, and modal changes. Task 13 only requires e2e coverage for draft preservation, QRIS modal cancel, successful payment, and auth smoke (`docs/plan/us-03-us-06-cashier-order-entry-revamp-plan.md` lines 475-490). Current component tests do cover same-screen paid-order cancellation and non-cancellable error handling (`frontend/src/features/cashier/CashierOrderScreen.test.tsx` lines 457-524), so this omission creates a real regression gap for US-06.

   Fix by adding a dedicated task or checkpoint that preserves and updates cancellation tests for the revamp UI, including the happy path and the non-cancellable error path.

## Open Questions Or Assumptions

- This review assumes the revamp plan is intended to satisfy the current MVP spec rather than serve as a visual-only prototype slice.
- If the intended scope is narrower than the spec, the plan should say so directly and stop claiming US-03 through US-06 completeness.

## Verification

- Reviewed the revamp plan against the current MVP spec, backend seed data, and current cashier frontend/unit/e2e coverage.
- No automated tests were run; this was a documentation review only.
