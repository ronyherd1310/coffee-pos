# Implementation Plan: US-03 to US-06 Frontend Cashier Order Entry

**Status:** Ready for code review

## Overview

Implement the frontend portion of the `Cashier Order Entry` section from `docs/specs/small-coffee-shop-pos-mvp-spec.md`, covering US-03 through US-06. This plan builds the authenticated cashier screen shown in `docs/screen-captures/03.order.png`, the manual payment confirmation dialog shown in `docs/screen-captures/04.order-confirm-payment.png`, and the QRIS asset flow using `docs/screen-captures/05.qris-payment.png`. The implementation is frontend-only: it consumes the current protected menu and order APIs, keeps unpaid drafts in browser state, persists only after `Confirm Paid`, exposes a minimal paid `Print Ticket` action from the newly created paid-order detail, and supports same-day cancellation only for a paid order already available in frontend state.

## Scope

In scope:

- Frontend order-entry screen inside the existing authenticated `ProtectedPosShell`.
- Cashier menu loading from `GET /api/pos/menu`.
- Menu item selection for seeded Americano and Latte data returned by the backend.
- Per-line required modifier selection for Temperature and Sugar.
- Frontend draft cart state, quantities, item removal, order note, subtotal, total, and payment method.
- Cash and QRIS payment method controls only.
- Static QRIS display at `/qris/static-qris.png` when QRIS is selected.
- Disabled unpaid print action until payment is confirmed.
- Minimal paid `Print Ticket` action after payment confirmation, using the paid-order detail currently held in frontend state.
- Manual `Confirm Paid` dialog before calling the backend create-order endpoint.
- Paid-order creation through `POST /api/pos/orders` with a frontend-generated `clientRequestId`.
- Post-payment paid-order detail state populated from the create response.
- Same-day cancellation confirmation and `POST /api/pos/orders/{orderId}/cancel` for the paid order currently held by the frontend.
- Frontend API clients, type guards/parsers, formatting helpers, component tests, and responsive styling.

Out of scope:

- Backend changes.
- Menu management UI.
- Persisting unpaid draft orders.
- Payment gateway integration or automatic QRIS payment verification.
- Today's Orders list, paid-order lookup after refresh, queue-number search, and reprint lookup.
- Full receipt-printer ticket polish, 80mm-specific print CSS, persisted reprint lookup, and printer troubleshooting flows. Those remain for the printing and Today's Orders slices.
- Daily sales summary.
- Dine-in, takeaway, delivery, customer names, service type labels, refunds, partial cancellation, or editing paid orders.

## Source Inputs

- Spec section: `docs/specs/small-coffee-shop-pos-mvp-spec.md` `### Cashier Order Entry`.
- UI references: `docs/screen-captures/03.order.png`, `docs/screen-captures/04.order-confirm-payment.png`, and `docs/screen-captures/05.qris-payment.png`.
- Backend implementation plan: `docs/plan/us-03-us-06-backend-cashier-order-entry-plan.md`.
- Current frontend auth shell: `frontend/src/App.tsx`, `frontend/src/features/pos/ProtectedPosShell.tsx`, and `frontend/src/styles.css`.
- Current frontend test conventions: Testing Library with Vitest under `frontend/src/**/*.test.tsx`.

## Current Backend Implementation Check

Checked current backend source and targeted tests before writing this plan.

Implemented and available:

- `GET /api/pos/menu`, protected by the existing auth middleware.
- `POST /api/pos/orders`, protected by auth, creates a paid order, returns `201 Created` for a new order and `200 OK` for an idempotent retry.
- `POST /api/pos/orders/{orderId}/cancel`, protected by auth, cancels a paid same-day order.
- Backend rejects client-supplied server-owned fields such as prices, totals, status, queue number, paid timestamp, and business date.
- Backend requires a canonical lowercase UUID `clientRequestId` for paid-order creation.
- Backend response shape includes `orderId`, `queueNumber`, `businessDate`, `status`, `paymentMethod`, `paidAt`, `cancelledAt`, `note`, `totalRp`, and order `lines`.
- Targeted backend verification passed: `go -C backend test ./internal/adapters/http ./internal/app/orders ./internal/domain/orders ./internal/app/menu`.

Not currently available:

- No frontend `public` directory or `frontend/public/qris/static-qris.png` asset exists yet. The approved source image for this plan is `docs/screen-captures/05.qris-payment.png`; Task 5 copies that file into the Vite public asset path.
- No router endpoint currently exposes `GET /api/pos/orders/{orderId}` or Today's Orders list. For this frontend slice, the paid detail screen must use the create/cancel response already in memory. Reload-safe lookup belongs to the Today's Orders slice.
- No frontend order-entry API client, cashier feature directory, QRIS asset, or order-entry UI exists yet.

## Backend API Contract Used By This Frontend Slice

### `GET /api/pos/menu`

Expected successful response:

```json
{
  "categories": [
    {
      "name": "Coffee",
      "slug": "coffee",
      "items": [
        {
          "name": "Americano",
          "slug": "americano",
          "priceRp": 18000,
          "modifierGroups": [
            {
              "name": "Temperature",
              "slug": "temperature",
              "required": true,
              "selectionType": "single",
              "options": [
                { "name": "Hot", "slug": "hot", "priceDeltaRp": 0 }
              ]
            }
          ]
        }
      ]
    }
  ]
}
```

Frontend requirements:

- Use relative URL `/api/pos/menu`.
- Use `credentials: "same-origin"`.
- Treat `401` as an expired or invalid session and return the cashier to auth handling.
- Validate the response shape enough that malformed menu data creates a recoverable error state instead of a broken screen.

### `POST /api/pos/orders`

Request shape:

```json
{
  "clientRequestId": "11111111-1111-4111-8111-111111111111",
  "paymentMethod": "cash",
  "note": "Optional order note",
  "lines": [
    {
      "menuItemSlug": "americano",
      "quantity": 1,
      "modifiers": [
        { "groupSlug": "temperature", "optionSlug": "hot" },
        { "groupSlug": "sugar", "optionSlug": "normal" }
      ]
    }
  ]
}
```

Frontend requirements:

- Generate `clientRequestId` with `crypto.randomUUID()`, lowercase canonical UUID format.
- Generate the UUID only when the cashier starts the confirm-payment submission, then reuse it for retries of the same confirmed draft.
- Omit `note` when the trimmed note is empty.
- Never send client-calculated `priceRp`, `unitPriceRp`, `lineTotalRp`, `totalRp`, `queueNumber`, `businessDate`, `status`, or timestamps.
- Map `400 invalid_client_request_id`, `409 idempotency_conflict`, `422 invalid_order`, `401 unauthorized`, and network failures to explicit frontend states.

### `POST /api/pos/orders/{orderId}/cancel`

Frontend requirements:

- Use only the internal `orderId` returned by the create response.
- Show a confirmation dialog before calling the endpoint.
- On success, replace local paid-order detail with the cancelled response.
- On `409 order_not_cancellable`, show that the order can no longer be cancelled from the current screen.
- Do not delete local order history or fabricate cancelled totals.

## Architecture Decisions

- Add order-entry code under `frontend/src/features/cashier/` to match the spec's intended frontend structure.
- Keep API calls in `frontend/src/lib/pos.ts` or equivalent small modules so components do not duplicate fetch details.
- Use TypeScript union types for `PaymentMethod` (`cash` and `qris`) and paid-order status (`paid` and `cancelled`).
- Use integer rupiah values only. Formatting belongs in a small frontend helper, for example `formatRupiah(43000) -> "Rp43.000"`.
- Keep unpaid draft state local to the cashier screen. Do not use localStorage, IndexedDB, or backend draft persistence.
- Keep each cart line independent, even when two lines use the same menu item with different modifiers.
- Recalculate draft totals from backend-provided menu prices and modifier deltas on every render or state update.
- Build accessible components with native buttons, form labels, `aria-live` for async errors/statuses, and focus management for modals.
- Use plain CSS and existing design tokens in `frontend/src/styles.css`; do not add a UI framework.
- The screen should be responsive. The desktop layout can follow the three-column reference image; tablet and mobile should stack the menu, current order, and payment/QRIS panel without horizontal scrolling.
- Use `docs/screen-captures/05.qris-payment.png` as the owner-provided QRIS source for this slice and copy it to `frontend/public/qris/static-qris.png`. Do not generate or substitute any other scannable QRIS image.
- Use a 120-character frontend order-note limit to match the order screen mockup. The backend's 500-character limit remains the outer safety limit.
- Keep the existing protected-shell navigation scoped to this slice. Show `New Order` and keep the existing `Daily Summary` navigation placeholder; do not add a functional `Today's Orders` action until US-07.

## Dependency Graph

```text
Existing auth shell and protected backend APIs
  -> frontend POS API client and response types
      -> rupiah / queue / modifier formatting helpers
          -> cashier menu loading state
              -> selected-item modifier form
                  -> draft cart state and totals
                      -> payment method and QRIS panel
                          -> confirm-payment dialog and create-order call
                              -> paid-order detail state
                                  -> minimal paid print action and cancel-order call
                                      -> responsive styling and browser verification
```

## Task List

### Phase 1: API And Type Foundation

## Task 1: Add Cashier API Client And Types

**Description:** Add typed frontend API helpers for cashier menu, paid-order creation, and paid-order cancellation. These helpers should normalize backend success and error responses into frontend-safe result types.

**Acceptance criteria:**

- [ ] `getCashierMenu` calls `GET /api/pos/menu` with `credentials: "same-origin"` and returns typed categories, items, modifier groups, and options.
- [ ] `createPaidOrder` calls `POST /api/pos/orders` with only backend-accepted fields.
- [ ] `cancelPaidOrder` calls `POST /api/pos/orders/{orderId}/cancel`.
- [ ] API helpers map unauthorized, invalid order, idempotency conflict, not cancellable, unavailable, and unexpected responses to explicit result states.
- [ ] Type definitions match the current backend JSON names exactly.

**Verification:**

- [ ] Tests pass: `npm --prefix frontend test`
- [ ] Type checks pass: `npm --prefix frontend run check`
- [ ] API client tests mock success, `401`, `409`, `422`, malformed JSON, and network failure.

**Dependencies:** Existing backend menu/order endpoints.

**Files likely touched:**

- `frontend/src/lib/pos.ts`
- `frontend/src/lib/pos.test.ts`
- `frontend/src/features/cashier/types.ts`

**Estimated scope:** Medium: 3 files

## Task 2: Add Draft Order Helpers

**Description:** Add small pure helpers for rupiah formatting, queue-number formatting, required modifier validation, line totals, order totals, and request payload creation from frontend draft state.

**Acceptance criteria:**

- [ ] Rupiah formatting displays Indonesian rupiah with no cents, such as `Rp43.000`.
- [ ] Queue formatting displays `Queue No. 001` for queue number `1`.
- [ ] Draft validation requires at least one cart line, one selected option for every required single-select modifier group, quantity from 1 through 99, and one payment method.
- [ ] Draft totals use backend-provided `priceRp` and `priceDeltaRp` integer values.
- [ ] Same drink with different modifier selections remains two separate lines.
- [ ] Create-order payload excludes all server-owned fields and omits empty notes.

**Verification:**

- [ ] Tests pass: `npm --prefix frontend test`
- [ ] Type checks pass: `npm --prefix frontend run check`
- [ ] Unit tests cover rupiah formatting, queue formatting, missing modifiers, quantity boundaries, total calculation, duplicate menu item lines with different modifiers, and create-order payload shape.

**Dependencies:** Task 1 types.

**Files likely touched:**

- `frontend/src/features/cashier/orderDraft.ts`
- `frontend/src/features/cashier/orderDraft.test.ts`
- `frontend/src/lib/format.ts`
- `frontend/src/lib/format.test.ts`

**Estimated scope:** Medium: 4 files

### Checkpoint: Foundation

- [ ] Frontend API client contract tests pass.
- [ ] Pure draft-order helper tests pass.
- [ ] `npm --prefix frontend test` and `npm --prefix frontend run check` pass.
- [ ] No UI has been wired to persist drafts yet.

### Phase 2: Cashier Screen Draft Flow

## Task 3: Replace Protected Placeholder With Cashier Screen Container

**Description:** Replace the current protected placeholder with a cashier screen container that loads the cashier menu, handles loading/error/empty states, and preserves the logout behavior already present in `ProtectedPosShell`.

**Acceptance criteria:**

- [ ] Authenticated users see the Coffee POS shell with the existing `New Order` and `Daily Summary` navigation placeholders.
- [ ] No functional `Today's Orders` action is added in this slice; US-07 owns that navigation change.
- [ ] Cashier screen fetches menu data after the authenticated shell renders.
- [ ] Loading, backend-unavailable, unauthorized, and empty-menu states are visible and recoverable.
- [ ] Unauthorized menu response can hand control back to auth by calling the existing signed-out callback.
- [ ] Logout still calls `POST /api/auth/logout` and returns to unauthenticated state.

**Verification:**

- [ ] Tests pass: `npm --prefix frontend test`
- [ ] Type checks pass: `npm --prefix frontend run check`
- [ ] Component tests cover menu loading success, retryable menu error, unauthorized menu response, and logout success.

**Dependencies:** Task 1.

**Files likely touched:**

- `frontend/src/features/pos/ProtectedPosShell.tsx`
- `frontend/src/features/pos/ProtectedPosShell.test.tsx`
- `frontend/src/features/cashier/CashierOrderScreen.tsx`
- `frontend/src/features/cashier/CashierOrderScreen.test.tsx`
- `frontend/src/styles.css`

**Estimated scope:** Medium: 5 files

## Task 4: Build Menu And Selected-Item Modifier Form

**Description:** Build the left-side menu and selected-item editor from the UI reference. The cashier can choose Americano or Latte, choose required modifiers for the selected item, adjust the pre-add quantity, and add the configured item as a new cart line.

**Acceptance criteria:**

- [ ] Menu renders backend-returned items with names, prices, and stable selection state.
- [ ] Selecting an item shows its modifier groups and options.
- [ ] Required Temperature and Sugar groups are single-select controls.
- [ ] Quantity stepper cannot go below 1 or above 99.
- [ ] `Add Item To Order` is disabled until all required modifiers are selected.
- [ ] Adding an item creates a new cart line and resets only the selected-item form state that should reset for the next line.

**Verification:**

- [ ] Tests pass: `npm --prefix frontend test`
- [ ] Type checks pass: `npm --prefix frontend run check`
- [ ] Component tests cover item selection, required modifier selection, quantity controls, disabled add state, and adding two lines for the same item with different modifiers.
- [ ] Keyboard check: menu item buttons, modifier options, steppers, and add button are reachable and operable.

**Dependencies:** Tasks 2 and 3.

**Files likely touched:**

- `frontend/src/features/cashier/CashierOrderScreen.tsx`
- `frontend/src/features/cashier/MenuPanel.tsx`
- `frontend/src/features/cashier/SelectedItemPanel.tsx`
- `frontend/src/features/cashier/CashierOrderScreen.test.tsx`
- `frontend/src/styles.css`

**Estimated scope:** Medium: 5 files

## Task 5: Build Draft Cart, Note, Totals, Payment, And QRIS Panel

**Description:** Build the current-order cart and right-side payment panel. The cashier can adjust cart-line quantities, remove lines, enter an optional note, select Cash or QRIS, and see QRIS instructions/image when QRIS is selected.

**Acceptance criteria:**

- [ ] Current order lists each cart line with quantity, item name, selected modifiers, line total, quantity controls, and remove action.
- [ ] Cart quantity controls update totals and respect the 1 through 99 range.
- [ ] Removing a line updates subtotal and total.
- [ ] Order note supports up to 120 characters and shows a `0 / 120` style character count.
- [ ] Payment method controls offer only Cash and QRIS.
- [ ] Selecting QRIS displays `/qris/static-qris.png` and manual-check helper text.
- [ ] Static QRIS asset exists at `frontend/public/qris/static-qris.png`, copied from `docs/screen-captures/05.qris-payment.png`, before QRIS UI is considered complete.
- [ ] `Confirm Paid` is disabled until the draft has at least one valid line and a payment method.
- [ ] `Print Ticket` is visibly disabled or unavailable for unpaid draft orders.

**Verification:**

- [ ] Tests pass: `npm --prefix frontend test`
- [ ] Type checks pass: `npm --prefix frontend run check`
- [ ] Component tests cover line quantity changes, remove line, 120-character note limit/count, payment method selection, QRIS image path, disabled confirm state, and disabled unpaid print state.
- [ ] Manual visual check at `320px`, `768px`, `1024px`, and `1440px`.

**Dependencies:** Tasks 2 through 4.

**Files likely touched:**

- `frontend/src/features/cashier/CurrentOrderPanel.tsx`
- `frontend/src/features/cashier/PaymentPanel.tsx`
- `frontend/src/features/cashier/CashierOrderScreen.tsx`
- `frontend/src/features/cashier/CashierOrderScreen.test.tsx`
- `frontend/public/qris/static-qris.png`
- `frontend/src/styles.css`

**Estimated scope:** Medium: 5-6 files

### Checkpoint: Draft Order UX

- [ ] Cashier can build an unpaid Americano or Latte order with required modifiers.
- [ ] Cashier can add the same drink twice with different modifiers as separate lines.
- [ ] Totals update from integer menu prices, modifier deltas, and quantities.
- [ ] Cash and QRIS controls work, and QRIS displays the static asset path.
- [ ] `Confirm Paid` and `Print Ticket` states match the spec for unpaid drafts.
- [ ] `frontend/public/qris/static-qris.png` exists and is copied from `docs/screen-captures/05.qris-payment.png`.
- [ ] `npm --prefix frontend test`, `npm --prefix frontend run check`, and `npm --prefix frontend run build` pass.

### Phase 3: Paid Order Submission And Correction

## Task 6: Add Confirm-Payment Dialog And Create-Order Submission

**Description:** Add the modal confirmation flow shown in `docs/screen-captures/04.order-confirm-payment.png`. Confirming the modal creates the paid order through the backend and transitions the frontend from editable draft to paid detail state.

**Acceptance criteria:**

- [ ] Clicking `Confirm Paid` opens a dialog with total and payment method.
- [ ] Dialog copy makes clear that the order will be persisted and cannot be edited after confirmation.
- [ ] `Back` closes the dialog without calling the backend.
- [ ] `Confirm Paid` in the dialog generates/reuses one `clientRequestId` for the confirmed draft and calls `POST /api/pos/orders`.
- [ ] Successful create response clears the unpaid draft and shows paid-order detail for the returned queue number.
- [ ] Idempotent retry success (`200 OK`) is treated as the same paid order.
- [ ] Invalid order, idempotency conflict, unauthorized, and unavailable errors are shown without silently losing the cashier's draft when recovery is possible.
- [ ] Dialog focus moves into the modal when opened and returns to the triggering control when closed.

**Verification:**

- [ ] Tests pass: `npm --prefix frontend test`
- [ ] Type checks pass: `npm --prefix frontend run check`
- [ ] Component tests cover open dialog, back/cancel, successful create, idempotent success, API error, request payload fields, UUID reuse on retry, and focus behavior.
- [ ] Manual keyboard check confirms modal focus behavior.

**Dependencies:** Tasks 1 through 5.

**Files likely touched:**

- `frontend/src/features/cashier/ConfirmPaymentDialog.tsx`
- `frontend/src/features/cashier/CashierOrderScreen.tsx`
- `frontend/src/features/cashier/CashierOrderScreen.test.tsx`
- `frontend/src/styles.css`

**Estimated scope:** Medium: 4 files

## Task 7: Add Paid Detail State, Minimal Print Action, And Same-Day Cancellation Flow

**Description:** Add a frontend paid-order detail state for the newly created order, expose the paid `Print Ticket` action required by the Cashier Order Entry flow, and support same-day cancellation through the current backend cancel endpoint. Because there is no order-detail GET endpoint yet, this detail view is intentionally in-memory for the newly created order only.

**Acceptance criteria:**

- [ ] After payment confirmation, the screen shows `Queue No. 001` style queue formatting, paid status, payment method, total, item lines, modifiers, note when present, and paid timestamp.
- [ ] Paid detail is read-only; no line, modifier, note, or payment editing controls remain active.
- [ ] `Print Ticket` is enabled for paid orders and unavailable for cancelled orders.
- [ ] The paid `Print Ticket` action opens or renders a minimal printable ticket view from the current paid-order detail and invokes the browser print flow.
- [ ] Minimal ticket content includes queue number, paid timestamp, item lines, modifiers, note when present, total, and payment method.
- [ ] Full 80mm receipt-printer CSS polish and reprint lookup remain explicitly deferred to the printing and Today's Orders slices.
- [ ] `Start New` returns to a fresh unpaid draft order.
- [ ] `Cancel Order` is available for a paid order currently shown in detail state.
- [ ] Clicking `Cancel Order` opens a confirmation dialog before calling the backend.
- [ ] Successful cancellation updates local detail status to cancelled and removes the ability to cancel again.
- [ ] `409 order_not_cancellable` and `404 not_found` produce clear non-destructive error states.
- [ ] The implementation documents that reload-safe paid order lookup waits for the Today's Orders slice.

**Verification:**

- [ ] Tests pass: `npm --prefix frontend test`
- [ ] Type checks pass: `npm --prefix frontend run check`
- [ ] Component tests cover paid detail rendering, enabled paid print action, disabled cancelled print action, minimal ticket content, start new, cancel dialog back action, successful cancel, already/not cancellable error, and cancelled detail state.

**Dependencies:** Task 6.

**Files likely touched:**

- `frontend/src/features/cashier/PaidOrderDetail.tsx`
- `frontend/src/features/cashier/PrintableTicket.tsx`
- `frontend/src/features/cashier/CancelOrderDialog.tsx`
- `frontend/src/features/cashier/CashierOrderScreen.tsx`
- `frontend/src/features/cashier/CashierOrderScreen.test.tsx`
- `frontend/src/styles.css`

**Estimated scope:** Medium: 5-6 files

### Checkpoint: Paid Order Flow

- [ ] Cashier confirms Cash payment through a dialog and lands on paid detail.
- [ ] Cashier confirms QRIS payment through a dialog after seeing `/qris/static-qris.png`.
- [ ] Backend create request contains slugs, quantities, modifiers, payment method, optional note, and `clientRequestId` only.
- [ ] Paid `Print Ticket` action is enabled after confirmation and renders a minimal ticket from the created order.
- [ ] Cashier can cancel the newly created paid order after a confirmation dialog.
- [ ] Cancelled orders remain visible as cancelled in local detail state.
- [ ] `npm --prefix frontend test`, `npm --prefix frontend run check`, and `npm --prefix frontend run build` pass.

### Phase 4: Visual Polish And Browser Verification

## Task 8: Align Responsive Visual Design With The Order Mockups

**Description:** Finalize styling so the cashier order screen is visually aligned with the supplied mockups while staying accessible and responsive. Keep the interface utilitarian and cashier-focused rather than marketing-like.

**Acceptance criteria:**

- [ ] Desktop layout follows the reference: header, menu/selected item column, current order column, and payment/QRIS panel.
- [ ] Mobile and tablet layouts stack cleanly without horizontal overflow.
- [ ] Buttons, steppers, radio-like options, dialogs, error states, and disabled states have clear visual and text affordances.
- [ ] Text does not overlap or overflow controls at `320px`, `768px`, `1024px`, and `1440px`.
- [ ] Color is not the only indicator for selected, disabled, paid, or cancelled states.
- [ ] Modal dialogs are visually centered, keyboard accessible, and have clear backdrop behavior.
- [ ] Minimal ticket view is readable in the browser print flow, without claiming final 80mm printer optimization.

**Verification:**

- [ ] Tests pass: `npm --prefix frontend test`
- [ ] Type checks pass: `npm --prefix frontend run check`
- [ ] Build succeeds: `npm --prefix frontend run build`
- [ ] Manual browser check covers `320px`, `768px`, `1024px`, and `1440px`.
- [ ] Manual keyboard check can complete the draft-to-confirm flow without a mouse.

**Dependencies:** Tasks 3 through 7.

**Files likely touched:**

- `frontend/src/styles.css`
- Existing cashier component files as needed for class names or accessibility labels.

**Estimated scope:** Small: 1-3 files

## Task 9: Add Focused Browser Smoke Coverage

**Description:** Add a minimal Playwright smoke test only after the frontend feature is stable. This should cover the critical cashier happy paths without duplicating every Vitest component case.

**Acceptance criteria:**

- [ ] Smoke test signs in or starts from an authenticated fixture according to existing e2e conventions.
- [ ] Smoke test mocks `/api/auth/session`, `/api/auth/login`, `/api/pos/menu`, `/api/pos/orders`, and cancellation only as needed; it does not require a seeded backend or database.
- [ ] Cashier creates one order with two lines whose modifiers differ.
- [ ] Cashier selects QRIS and sees `/qris/static-qris.png`.
- [ ] Cashier opens the confirm-payment dialog and confirms paid.
- [ ] Smoke test verifies the paid detail shows the returned queue number.
- [ ] Smoke test verifies `Print Ticket` is unavailable before payment and available after payment.
- [ ] Smoke test avoids testing persisted reprint or Today's Orders lookup in this slice.

**Verification:**

- [ ] E2E passes locally when backend/frontend test setup is available: `npm run test:e2e`.
- [ ] If e2e setup is not available, document the skipped check and rely on Vitest plus manual browser verification.

**Dependencies:** Tasks 1 through 8.

**Files likely touched:**

- `tests/e2e/cashier-order-entry.spec.ts`
- Test fixtures/helpers if they already exist.

**Estimated scope:** Small to Medium: 1-3 files

## Task 10: Final Visual Verification Against UI Designs

**Description:** Perform a final visual verification pass after implementation, comparing the running frontend against the initial UI designs in `docs/screen-captures/03.order.png` and `docs/screen-captures/04.order-confirm-payment.png`. This task is not a pixel-perfect requirement, but the implemented screen should be clearly similar in layout, visual hierarchy, control placement, states, and overall cashier workflow.

**Acceptance criteria:**

- [ ] Captured implementation screenshot for the main order-entry screen is visually similar to `docs/screen-captures/03.order.png`.
- [ ] Captured implementation screenshot for the confirm-payment dialog is visually similar to `docs/screen-captures/04.order-confirm-payment.png`.
- [ ] Desktop layout preserves the intended structure: top header, left menu/selected-item column, center current-order column, and right QRIS/payment panel.
- [ ] Confirm-payment dialog appears centered over the dimmed order screen with total, payment method, explanatory copy, `Back`, and `Confirm Paid` controls.
- [ ] Key visual states match the design intent: selected modifiers, QRIS selected payment, not-paid status, disabled unpaid print action, green primary actions, and subtle glass/surface styling.
- [ ] Text, icons, buttons, QRIS image, and dialogs do not overlap or clip at the design viewport and at the responsive breakpoints already covered by Task 8.
- [ ] Any intentional visual differences from the screenshots are documented briefly in the implementation notes or PR description.

**Verification:**

- [ ] Tests pass: `npm --prefix frontend test`
- [ ] Type checks pass: `npm --prefix frontend run check`
- [ ] Build succeeds: `npm --prefix frontend run build`
- [ ] Browser screenshots are captured for the main order screen and confirm-payment dialog at the same desktop viewport used by the supplied designs.
- [ ] Manual visual comparison is completed against `docs/screen-captures/03.order.png` and `docs/screen-captures/04.order-confirm-payment.png`.

**Dependencies:** Tasks 1 through 9.

**Files likely touched:**

- `frontend/src/styles.css`
- Existing cashier component files as needed for final visual adjustments.
- `tests/e2e/cashier-order-entry.spec.ts` if screenshot capture is automated.

**Estimated scope:** Small: 1-3 files

## Risks And Mitigations

| Risk | Impact | Mitigation |
| --- | --- | --- |
| No `GET /api/pos/orders/{orderId}` endpoint exists. | Paid detail cannot be restored after refresh. | Use create/cancel responses for this slice and leave reload-safe lookup to Today's Orders. |
| QRIS asset copy is missed. | QRIS flow cannot match the spec. | Copy `docs/screen-captures/05.qris-payment.png` to `frontend/public/qris/static-qris.png` in Task 5 and test that the UI references `/qris/static-qris.png`. |
| Frontend totals could drift from backend totals. | Cashier sees one total before confirmation and another after persistence. | Calculate frontend totals from backend menu data only, and display backend-returned total after create. |
| Idempotency key handling is easy to get wrong. | Retrying confirm could create conflicts or duplicate orders. | Generate UUID at submit start and reuse it only for retries of the same confirmed draft. |
| Large all-in-one screen can become hard to test. | Fragile components and large diffs. | Land small components and helpers incrementally with tests after each task. |
| Cashier workflow is speed-sensitive. | UI may be visually correct but slow to operate. | Keep interactions direct, avoid unnecessary modals before final payment, and keyboard-test core controls. |
| Minimal ticket action is confused with full printing scope. | Reviewers may expect 80mm receipt polish in this slice. | This slice only enables paid printability from the new paid detail. Full ticket CSS, reprint lookup, and printer-target verification remain in the printing slice. |
| Final implementation drifts from the supplied UI designs. | The feature may work but fail the expected cashier experience. | Task 10 requires screenshot capture and manual comparison against `03.order.png` and `04.order-confirm-payment.png` before review. |

## Resolved Review Decisions

- Paid ticket scope: this slice includes the minimal paid `Print Ticket` action and printable ticket view needed after payment confirmation. Full 80mm receipt-printer CSS, reprint lookup, and persisted ticket recovery remain out of scope.
- QRIS asset policy: use the provided `docs/screen-captures/05.qris-payment.png` as the approved source image and copy it to `frontend/public/qris/static-qris.png`. Do not generate or substitute a different QRIS image.
- Header navigation scope: keep `Today's Orders` out of this slice. Retain the current protected shell's `Daily Summary` placeholder until the relevant later slices update navigation intentionally.
- Order note limit: use 120 characters in the frontend to match the supplied UI mockup. The backend's 500-character validation remains a broader safety limit.
- Playwright strategy: use mocked auth, menu, and order API routes for this frontend-only smoke test, following the current `tests/e2e/auth-login.spec.ts` style.

## Definition Of Done

- [ ] Every task has focused acceptance criteria and verification.
- [ ] Each implementation increment leaves the frontend compiling and tests passing.
- [ ] No backend files are changed for this frontend-only slice.
- [ ] No cashier PIN, PIN hash, session token, or private payment key is added to frontend code or static assets.
- [ ] Unpaid drafts remain frontend-only.
- [ ] Paid order persistence happens only after the confirmation dialog action.
- [ ] QRIS uses `/qris/static-qris.png`, sourced from `docs/screen-captures/05.qris-payment.png`.
- [ ] Frontend order notes are capped at 120 characters.
- [ ] `Today's Orders` navigation and lookup remain out of scope for this slice.
- [ ] Paid orders expose a minimal enabled `Print Ticket` action; full 80mm print polish remains out of scope.
- [ ] Final visual verification compares implementation screenshots against `docs/screen-captures/03.order.png` and `docs/screen-captures/04.order-confirm-payment.png`.
- [ ] `npm --prefix frontend test`, `npm --prefix frontend run check`, and `npm --prefix frontend run build` pass before review.
- [ ] Browser verification covers responsive layout and keyboard access for the main order-entry flow.
