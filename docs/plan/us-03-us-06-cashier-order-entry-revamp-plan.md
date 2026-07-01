# Implementation Plan: US-03 to US-06 Cashier Order Entry Revamp

**Status:** Ready for code review

## Overview

Update the existing cashier order-entry flow to match the revamp target in `docs/screen-captures/06-order-revamp.png` and the `### Cashier Order Entry` section of `docs/specs/small-coffee-shop-pos-mvp-spec.md`.

This is not implementation-ready as a frontend-only slice if it is expected to satisfy the current spec. The revamp UI can be built mostly in the frontend, but full US-03 acceptance depends on backend menu seed and menu contract work for the expanded catalog, categories, image paths, and filter metadata. This plan therefore includes backend prerequisite tasks before the frontend revamp tasks. Tax/service-charge remains out of scope until the business policy is approved.

## Review Findings Resolved

This plan incorporates the required findings from `docs/reviews/us-03-us-06-cashier-order-entry-revamp-plan-review-2026-07-01.md`:

- Backend seed and menu display metadata are explicit Phase 0 prerequisites for full US-03 acceptance.
- The non-functional grid/list toggle is excluded from scope and must not render in the accepted revamp UI.
- US-06 paid-order cancellation has dedicated regression coverage in Task 11 and the e2e acceptance path in Task 14.

## Source Inputs

- Spec section: `docs/specs/small-coffee-shop-pos-mvp-spec.md` `### Cashier Order Entry`.
- Design reference: `docs/screen-captures/06-order-revamp.png`.
- Existing frontend implementation:
  - `frontend/src/features/cashier/CashierOrderScreen.tsx`
  - `frontend/src/features/cashier/ConfirmPaymentDialog.tsx`
  - `frontend/src/features/cashier/CancelOrderDialog.tsx`
  - `frontend/src/features/cashier/PaidOrderDetail.tsx`
  - `frontend/src/features/cashier/orderDraft.ts`
  - `frontend/src/features/cashier/types.ts`
  - `frontend/src/lib/pos.ts`
  - `frontend/src/styles.css`
- Existing frontend tests:
  - `frontend/src/features/cashier/CashierOrderScreen.test.tsx`
  - `frontend/src/features/cashier/orderDraft.test.ts`
  - `frontend/src/lib/pos.test.ts`
  - `tests/e2e/cashier-order-entry.spec.ts`
- Existing backend contracts:
  - `GET /api/pos/menu`
  - `POST /api/pos/orders`
  - `POST /api/pos/orders/{orderId}/cancel`

## Current Implementation Snapshot

### Frontend

Already implemented:

- Authenticated `ProtectedPosShell` renders `CashierOrderScreen`.
- `CashierOrderScreen` loads `GET /api/pos/menu`.
- Cashier can select a menu item, choose required modifiers, set quantity, and add a cart line.
- Draft cart supports multiple lines, quantity changes, line removal, 120-character note limit, payment method selection, QRIS preview, disabled unpaid print action, confirm-payment dialog, create-order API call, paid-order detail, printable ticket action, and cancellation dialog.
- `orderDraft.ts` calculates line totals and draft total from backend-provided prices/modifier deltas.
- `pos.ts` parses current backend menu/order/cancel response shapes.
- Static assets already exist for current images:
  - `frontend/public/avatar/cashier.png`
  - `frontend/public/menu/americano.png`
  - `frontend/public/menu/latte.png`
  - `frontend/public/qris/static-qris.png`

Gaps versus the revamp:

- Layout is a selected-item/sidebar workflow, not the design's top search + catalog controls + menu card grid + persistent right order panel.
- No menu search state.
- No `All` tab or category tab filtering beyond rendering backend categories as sections.
- No quick filters for Best Seller, Iced, Low Sugar, or New Arrival.
- No sort control or grid/list affordance.
- Menu item images are hard-coded by slug heuristic in `menuItemImageSrc`.
- Current-order rows do not render item thumbnails.
- QRIS image appears in a side preview panel before confirmation, not inside the focused payment modal shown in the revamp.
- Confirm action text is `Confirm Paid`; the revamp uses `Proceed to Payment` before the modal.
- No draft order reference display. The spec leaves `#ORD-0142` as an open question.
- No tax row. The spec explicitly says the `Tax (11%)` design element needs business confirmation before implementation.

### Backend

Already implemented:

- `GET /api/pos/menu` returns categories, items, prices, modifier groups, and modifier options.
- `POST /api/pos/orders` accepts only `clientRequestId`, `paymentMethod`, `note`, and `lines`; it rejects client-supplied server-owned values such as prices, totals, queue number, status, timestamps, and business date.
- Backend resolves prices/modifiers from menu data and calculates totals server-side.
- `POST /api/pos/orders/{orderId}/cancel` cancels same-day paid orders.

Backend gaps relevant to the new design:

- Menu schema and API do not expose item image paths, popularity, promotion, iced, low-sugar, or new-arrival metadata.
- Current approved seed is Coffee only with Americano and Latte.
- No tax/service-charge schema or total breakdown exists.
- No Today's Orders lookup endpoint is part of this revamp plan.

## Architecture Decisions

- Full US-03 acceptance requires backend seed and menu response work before the frontend can truthfully render the approved category/filter behavior from real data.
- Keep order creation/cancellation contracts unchanged unless tax/service charge is explicitly approved in a later backend plan.
- Preserve backend authority for totals. The frontend may display subtotal and total from local draft state, but it must not send totals, line prices, tax, queue number, status, or timestamps to `POST /api/pos/orders`.
- Treat tax/service charge as not implemented in this plan. The UI may reserve space for an optional row only if it renders as absent or `Rp0`; do not hard-code `11%`.
- Treat image paths and filter metadata as backend-provided menu display metadata for the full revamp. Frontend parsers may remain backward-compatible, but the full acceptance path requires the backend to provide the metadata for seeded items.
- Derive safe fallback images from known slugs only as a compatibility fallback, not as the primary data source for the revamp.
- Keep unpaid draft state in `CashierOrderScreen` state; do not persist drafts to backend or browser storage.
- Keep paid-order detail and cancellation behavior from the current implementation.
- Use plain CSS in `frontend/src/styles.css`; do not add a UI framework or routing library.
- Use accessible native controls: search input, tab-like category buttons, radio inputs for payment method, buttons for steppers and remove actions, and modal focus management.
- Do not render a grid/list view toggle in this revamp. The default and only planned catalog view is the image-card grid.

## Dependency Graph

```text
Backend seed and menu metadata contract
  -> current backend menu/order/cancel API
      -> frontend menu type extensions and display metadata helpers
      -> catalog view model: search, category, quick filters, sort, pagination/windowing
          -> catalog header controls and menu card grid
              -> item customization entry point
                  -> current-order panel row rendering and totals
                      -> payment method + Proceed to Payment
                          -> QRIS/cash payment modal
                              -> paid-order detail/cancel/print regression checks
                                  -> responsive visual polish and e2e coverage
```

## Backend Impact Assessment

Expected for this revamp: frontend plus backend menu seed/metadata prerequisites.

Backend work is required for full US-03 acceptance because the current spec requires real seeded categories, catalog items, and filter metadata:

- Expanded seeded menu categories/items beyond the current Coffee/Americano/Latte seed.
- Persisted or API-exposed image paths and badge/filter metadata for seeded items.

Backend work is still not included for:

- Tax/service-charge calculation and persisted total breakdown.
- Draft order references such as `#ORD-0142`.

If the team intentionally wants a visual-only prototype instead, rename/narrow this plan before implementation and do not mark US-03 complete.

## Task List

### Phase 0: Backend Menu Prerequisites

## Task 0A: Expand Approved Menu Seed Data

**Description:** Update the backend approved menu seed so `GET /api/pos/menu` can return the categories and starter catalog required by the spec: Coffee, Tea, Snacks, Seasonal, and the approved item list.

**Acceptance criteria:**

- [ ] Backend seed supports multiple categories instead of only one Coffee category.
- [ ] Seeder creates the approved items from the spec with stable slugs and prices.
- [ ] Existing Americano and Latte order creation remains compatible with current order tests.
- [ ] Running the seeder repeatedly remains idempotent.

**Verification:**

- [ ] Tests pass: `go -C backend test ./internal/domain/menu ./internal/app/menu ./internal/adapters/postgres`
- [ ] Integration tests pass if Podman/Testcontainers is available: `go -C backend test -tags=integration ./internal/adapters/postgres ./internal/seed`

**Dependencies:** None.

**Files likely touched:**

- `backend/internal/domain/menu/menu.go`
- `backend/internal/domain/menu/seed.go`
- `backend/internal/domain/menu/seed_test.go`
- `backend/internal/adapters/postgres/menu_repository.go`
- `backend/internal/adapters/postgres/menu_repository_integration_test.go`

**Estimated scope:** Medium: 5 files

## Task 0B: Persist Menu Display Metadata

**Description:** Add backend-supported storage and repository mapping for menu display metadata needed by the revamp: image path and filter/badge flags such as best seller, promo, iced, low sugar, and new arrival.

**Acceptance criteria:**

- [ ] Menu schema can store optional `imagePath` and display/filter flags for menu items.
- [ ] Menu seed persists display metadata idempotently with approved menu data.
- [ ] PostgreSQL menu repository reads display metadata into application menu types.

**Verification:**

- [ ] Tests pass: `go -C backend test ./internal/domain/menu ./internal/app/menu ./internal/adapters/postgres`
- [ ] Integration tests pass if Podman/Testcontainers is available: `go -C backend test -tags=integration ./internal/adapters/postgres ./internal/seed`

**Dependencies:** Task 0A.

**Files likely touched:**

- `backend/migrations/<next>_add_menu_display_metadata.sql`
- `backend/queries/menu.sql`
- `backend/internal/app/menu/ports.go`
- `backend/internal/adapters/postgres/menu_repository.go`
- `backend/internal/adapters/postgres/menu_repository_integration_test.go`

**Estimated scope:** Medium: 5 files

## Task 0C: Expose Menu Display Metadata In The API

**Description:** Extend `GET /api/pos/menu` so frontend clients can consume the display metadata from Task 0B while older clients that ignore the fields remain compatible.

**Acceptance criteria:**

- [ ] `GET /api/pos/menu` includes display metadata for seeded items.
- [ ] Metadata field names are documented by tests and match the frontend parser expectations.
- [ ] Existing menu response fields remain unchanged.
- [ ] Handler tests cover a menu item with image path and filter/badge metadata.

**Verification:**

- [ ] Tests pass: `go -C backend test ./internal/adapters/http ./internal/app/menu`
- [ ] Manual/API check confirms metadata appears in `GET /api/pos/menu`.

**Dependencies:** Task 0B.

**Files likely touched:**

- `backend/internal/adapters/http/menu_handlers.go`
- `backend/internal/adapters/http/menu_handlers_test.go`
- `backend/internal/app/menu/ports.go`
- `backend/internal/app/menu/usecases_test.go`

**Estimated scope:** Medium: 4 files

### Checkpoint: Backend Menu Contract

- [ ] `go -C backend test ./...`
- [ ] `go -C backend test -tags=integration ./...` if Podman/Testcontainers is available.
- [ ] Manual/API check confirms `GET /api/pos/menu` returns Coffee, Tea, Snacks, Seasonal, seeded item prices, image paths, and filter metadata.

### Phase 1: Frontend Data And View Model Foundation

## Task 1: Extend Frontend Menu Types For Optional Display Metadata

**Description:** Make the frontend menu model consume the backend display metadata added by Task 0C while remaining tolerant of older responses during local development.

**Acceptance criteria:**

- [ ] `MenuItem` includes optional `imagePath`, `badges`, `tags`, or equivalent display metadata used by the revamp UI.
- [ ] `parseMenuItem` accepts current backend responses with no display metadata.
- [ ] `parseMenuItem` validates optional metadata when present and rejects malformed metadata safely.
- [ ] Known slugs still get deterministic fallback images.

**Verification:**

- [ ] Tests pass: `npm --prefix frontend test -- src/lib/pos.test.ts`
- [ ] Type checks pass: `npm --prefix frontend run check`

**Dependencies:** Task 0C for full spec acceptance; none for parser compatibility work.

**Files likely touched:**

- `frontend/src/features/cashier/types.ts`
- `frontend/src/lib/pos.ts`
- `frontend/src/lib/pos.test.ts`

**Estimated scope:** Medium: 3 files

## Task 2: Extract Catalog View Model Helpers

**Description:** Add pure helpers that flatten backend categories into catalog items and apply search, category, quick-filter, and sort state. This keeps UI rendering thin and makes the filter behavior testable.

**Acceptance criteria:**

- [ ] Helper returns an `All` category plus backend categories in display order.
- [ ] Search matches item names case-insensitively and does not mutate the draft order.
- [ ] Category filtering supports `All`, Coffee, Tea, Snacks, Seasonal, and any backend category returned later.
- [ ] Quick filters use backend item metadata from Task 0C.
- [ ] Popular sort is deterministic and uses backend popularity/display metadata where present, falling back to existing backend/category order only for compatibility.

**Verification:**

- [ ] Tests pass: `npm --prefix frontend test -- src/features/cashier/catalogView.test.ts`
- [ ] Type checks pass: `npm --prefix frontend run check`

**Dependencies:** Task 1.

**Files likely touched:**

- `frontend/src/features/cashier/catalogView.ts`
- `frontend/src/features/cashier/catalogView.test.ts`
- `frontend/src/features/cashier/types.ts`

**Estimated scope:** Medium: 3 files

## Task 3: Preserve Draft Calculations Without Tax

**Description:** Confirm draft totals remain subtotal-only until tax/service policy is approved, and expose a small total-breakdown helper for the order panel.

**Acceptance criteria:**

- [ ] Total breakdown returns `subtotalRp`, optional `taxRp` as absent/zero, and `totalRp`.
- [ ] Existing payload builder still omits all client-calculated totals.
- [ ] Tests document that tax is not hard-coded from the screenshot.

**Verification:**

- [ ] Tests pass: `npm --prefix frontend test -- src/features/cashier/orderDraft.test.ts`
- [ ] Type checks pass: `npm --prefix frontend run check`

**Dependencies:** None.

**Files likely touched:**

- `frontend/src/features/cashier/orderDraft.ts`
- `frontend/src/features/cashier/orderDraft.test.ts`

**Estimated scope:** Small: 2 files

### Checkpoint: Data Foundation

- [ ] `npm --prefix frontend test -- src/lib/pos.test.ts src/features/cashier/catalogView.test.ts src/features/cashier/orderDraft.test.ts`
- [ ] `npm --prefix frontend run check`
- [ ] Confirm frontend request payloads still match the backend order contract and do not include display metadata or totals.

### Phase 2: Catalog Revamp UI

## Task 4: Add Catalog Header Controls

**Description:** Replace the current simple menu section header with revamp controls: top search input, category tabs, quick filters, and sort control. The revamp will ship the image-card grid only; do not render a list-view toggle in this slice.

**Acceptance criteria:**

- [ ] Search input has an accessible label or placeholder matching the revamp intent.
- [ ] Category controls include `All` and backend categories.
- [ ] Quick filters render Best Seller, Iced, Low Sugar, and New Arrival controls.
- [ ] Sort control renders Popular as the default option.
- [ ] No grid/list toggle is rendered.
- [ ] Changing any control does not clear selected item, modifiers, cart lines, note, or payment method.

**Verification:**

- [ ] Tests pass: `npm --prefix frontend test -- src/features/cashier/CashierOrderScreen.test.tsx`
- [ ] Manual check: add an item, type in search, change filters, and verify the current order is unchanged.

**Dependencies:** Task 2.

**Files likely touched:**

- `frontend/src/features/cashier/CashierOrderScreen.tsx`
- `frontend/src/features/cashier/CashierOrderScreen.test.tsx`
- `frontend/src/styles.css`

**Estimated scope:** Medium: 3 files

## Task 5: Convert Menu Rendering To Image Card Grid

**Description:** Render filtered catalog results as image-led product cards similar to the screenshot. Keep the existing add/customize behavior so required modifiers are still selected before adding a line.

**Acceptance criteria:**

- [ ] Cards show image, name, formatted price, optional badge/filter indicator, and accessible add/select control.
- [ ] Empty search/filter results show a useful empty state.
- [ ] Fallback images are used when an item has no explicit image path.
- [ ] Existing required-modifier behavior still prevents invalid lines.

**Verification:**

- [ ] Tests pass: `npm --prefix frontend test -- src/features/cashier/CashierOrderScreen.test.tsx`
- [ ] Manual check: cards render without broken image icons for current Americano/Latte data.

**Dependencies:** Tasks 1, 2, and 4.

**Files likely touched:**

- `frontend/src/features/cashier/CashierOrderScreen.tsx`
- `frontend/src/features/cashier/CashierOrderScreen.test.tsx`
- `frontend/src/styles.css`

**Estimated scope:** Medium: 3 files

## Task 6: Reposition Item Customization For The Revamp

**Description:** Move the selected-item modifier controls out of the old sidebar mental model. Use either an inline panel below the selected card or a focused modal/popover, while keeping keyboard access and existing validation.

**Acceptance criteria:**

- [ ] Selecting an item with required modifiers exposes Temperature and Sugar choices before the line can be added.
- [ ] Selecting an item without required modifiers can be added quickly without unnecessary steps.
- [ ] Quantity stepper remains clamped from 1 to 99.
- [ ] Closing/cancelling customization does not mutate existing cart lines.

**Verification:**

- [ ] Tests pass: `npm --prefix frontend test -- src/features/cashier/CashierOrderScreen.test.tsx`
- [ ] Manual check: add two Americano lines with different modifiers and verify separate cart rows.

**Dependencies:** Task 5.

**Files likely touched:**

- `frontend/src/features/cashier/CashierOrderScreen.tsx`
- `frontend/src/features/cashier/CashierOrderScreen.test.tsx`
- `frontend/src/styles.css`

**Estimated scope:** Medium: 3 files

### Checkpoint: Catalog Revamp

- [ ] `npm --prefix frontend test -- src/features/cashier/CashierOrderScreen.test.tsx`
- [ ] `npm --prefix frontend run check`
- [ ] Manual browser check confirms search/category/filter controls preserve current order state.

### Phase 3: Current Order Panel Revamp

## Task 7: Restyle Current Order Rows With Thumbnails

**Description:** Update current-order rows to match the right panel in the design: quantity badge, thumbnail, item name, modifier summary, line total, quantity controls, and remove icon/button.

**Acceptance criteria:**

- [ ] Each cart line shows quantity badge, image thumbnail, name, modifier summary, and line total.
- [ ] Quantity increment/decrement and remove controls keep existing behavior.
- [ ] Long item names and modifier summaries wrap without overlapping adjacent controls.

**Verification:**

- [ ] Tests pass: `npm --prefix frontend test -- src/features/cashier/CashierOrderScreen.test.tsx`
- [ ] Manual check: add at least two cart lines and resize desktop/tablet widths.

**Dependencies:** Task 5.

**Files likely touched:**

- `frontend/src/features/cashier/CashierOrderScreen.tsx`
- `frontend/src/features/cashier/CashierOrderScreen.test.tsx`
- `frontend/src/styles.css`

**Estimated scope:** Medium: 3 files

## Task 8: Align Order Summary And Payment Controls

**Description:** Update the right panel summary to show order note, note count, subtotal, optional tax/service row, total, payment method cards, and the primary `Proceed to Payment` action.

**Acceptance criteria:**

- [ ] Note field remains limited to 120 characters and displays `current / 120`.
- [ ] Summary displays subtotal and total; tax row is hidden or zero until policy is approved.
- [ ] Cash and QRIS controls are accessible radio choices styled as payment cards.
- [ ] `Proceed to Payment` is disabled until at least one valid line and payment method exist.
- [ ] Unpaid print remains disabled/unavailable.

**Verification:**

- [ ] Tests pass: `npm --prefix frontend test -- src/features/cashier/CashierOrderScreen.test.tsx`
- [ ] Manual check: verify disabled/enabled states for empty, missing payment, and valid order states.

**Dependencies:** Tasks 3 and 7.

**Files likely touched:**

- `frontend/src/features/cashier/CashierOrderScreen.tsx`
- `frontend/src/features/cashier/CashierOrderScreen.test.tsx`
- `frontend/src/styles.css`

**Estimated scope:** Medium: 3 files

### Checkpoint: Order Panel

- [ ] `npm --prefix frontend test -- src/features/cashier/CashierOrderScreen.test.tsx src/features/cashier/orderDraft.test.ts`
- [ ] `npm --prefix frontend run check`
- [ ] Manual browser check confirms the order panel matches the screenshot hierarchy at desktop width.

### Phase 4: Payment Modal Revamp

## Task 9: Replace QRIS Side Preview With Payment Modal Content

**Description:** Move QRIS display into the payment confirmation modal. Cash payment can use the same modal shell without the QR image.

**Acceptance criteria:**

- [ ] `Proceed to Payment` opens a modal labeled by payment method.
- [ ] QRIS modal shows total amount, `/qris/static-qris.png`, scan instruction copy, `Confirm Paid`, `Cancel`, and close control.
- [ ] Cash modal shows total amount, payment method, `Confirm Paid`, `Cancel`, and close control without QRIS image.
- [ ] Cancel/close returns to the unchanged current order and restores focus to the trigger.

**Verification:**

- [ ] Tests pass: `npm --prefix frontend test -- src/features/cashier/CashierOrderScreen.test.tsx`
- [ ] Manual check: open QRIS modal, cancel it, and verify cart/note/payment state remain unchanged.

**Dependencies:** Task 8.

**Files likely touched:**

- `frontend/src/features/cashier/ConfirmPaymentDialog.tsx`
- `frontend/src/features/cashier/CashierOrderScreen.tsx`
- `frontend/src/features/cashier/CashierOrderScreen.test.tsx`
- `frontend/src/styles.css`

**Estimated scope:** Medium: 4 files

## Task 10: Preserve Paid Order Creation And Retry Semantics

**Description:** Verify the revamp still submits the same backend-safe payload and preserves idempotent retry behavior from the existing implementation.

**Acceptance criteria:**

- [ ] Create-order request still contains only `clientRequestId`, `paymentMethod`, optional `note`, and `lines`.
- [ ] No UI-only fields such as image path, tags, subtotal, tax, total, status, or queue number are sent.
- [ ] Recoverable create failure keeps the modal open and reuses the same `clientRequestId` on retry.
- [ ] Successful create still opens paid-order detail and clears draft state.

**Verification:**

- [ ] Tests pass: `npm --prefix frontend test -- src/features/cashier/CashierOrderScreen.test.tsx`
- [ ] Tests pass: `npm --prefix frontend test -- src/features/cashier/orderDraft.test.ts`

**Dependencies:** Task 9.

**Files likely touched:**

- `frontend/src/features/cashier/CashierOrderScreen.tsx`
- `frontend/src/features/cashier/CashierOrderScreen.test.tsx`
- `frontend/src/features/cashier/orderDraft.ts`
- `frontend/src/features/cashier/orderDraft.test.ts`

**Estimated scope:** Medium: 4 files

### Checkpoint: Payment Flow

- [ ] `npm --prefix frontend test`
- [ ] `npm --prefix frontend run check`
- [ ] Manual browser check confirms QRIS modal matches the design intent and payment succeeds against mocked or local backend data.

### Phase 5: Cancellation Regression, Shell Layout, Responsiveness, And E2E

## Task 11: Preserve Paid-Order Cancellation Behavior

**Description:** Update and preserve US-06 cancellation coverage after the paid-detail and modal refactor. Cancellation is not visually central to the revamp screenshot, but it remains part of US-06 and must not regress while changing the payment/detail flow.

**Acceptance criteria:**

- [ ] Paid-order detail still exposes `Cancel Order` for cancellable paid orders.
- [ ] Cancellation opens a confirmation dialog and only calls the backend after confirmation.
- [ ] Successful cancellation replaces local paid-order detail with the cancelled response and disables paid-only actions that no longer apply.
- [ ] `409 order_not_cancellable` shows a non-destructive error and leaves the paid order in paid state.

**Verification:**

- [ ] Tests pass: `npm --prefix frontend test -- src/features/cashier/CashierOrderScreen.test.tsx`
- [ ] Manual check: create a paid order in the browser, cancel from paid detail, and verify the cancelled status.

**Dependencies:** Tasks 9 and 10.

**Files likely touched:**

- `frontend/src/features/cashier/CashierOrderScreen.tsx`
- `frontend/src/features/cashier/CashierOrderScreen.test.tsx`
- `frontend/src/features/cashier/CancelOrderDialog.tsx`
- `frontend/src/features/cashier/PaidOrderDetail.tsx`

**Estimated scope:** Medium: 4 files

## Task 12: Update Protected Shell Header Layout

**Description:** Adjust the authenticated shell to match the revamp's first-viewport structure: POS mark, Coffee POS title, centered search area if search is owned by the shell, New Order and Today's Orders actions, and cashier/avatar affordance.

**Acceptance criteria:**

- [ ] Header visually aligns with the revamp while preserving logout accessibility.
- [ ] Search ownership is clear: either the search input remains inside `CashierOrderScreen` or shell passes the search state down explicitly.
- [ ] `New Order` and `Today's Orders` remain present without implementing a Today's Orders page in this slice.
- [ ] Logout remains reachable by keyboard and screen reader.

**Verification:**

- [ ] Tests pass: `npm --prefix frontend test -- src/features/pos/ProtectedPosShell.test.tsx src/App.test.tsx`
- [ ] Manual check: authenticated shell still logs out correctly.

**Dependencies:** Task 4.

**Files likely touched:**

- `frontend/src/features/pos/ProtectedPosShell.tsx`
- `frontend/src/features/pos/ProtectedPosShell.test.tsx`
- `frontend/src/App.test.tsx`
- `frontend/src/styles.css`

**Estimated scope:** Medium: 4 files

## Task 13: Responsive Styling And Asset Verification

**Description:** Complete CSS for desktop/tablet/mobile layouts, ensure static assets are served in the container build, and avoid layout overlap.

**Acceptance criteria:**

- [ ] Desktop layout shows catalog and current-order panel side by side.
- [ ] Tablet/mobile layout stacks controls without horizontal scrolling or overlapping text.
- [ ] Product images, cashier avatar, and QRIS image render from `frontend/public`.
- [ ] Buttons, cards, and text fit their containers at common viewport widths.

**Verification:**

- [ ] Build succeeds: `npm --prefix frontend run build`
- [ ] Type checks pass: `npm --prefix frontend run check`
- [ ] Container spot check after rebuild: `curl -I http://localhost:8080/menu/americano.png` returns `Content-Type: image/png`.
- [ ] Manual browser checks at desktop and mobile widths.

**Dependencies:** Tasks 5, 7, 8, 9, and 12.

**Files likely touched:**

- `frontend/src/styles.css`
- `frontend/Containerfile`
- `frontend/public/menu/*`
- `frontend/public/avatar/*`

**Estimated scope:** Medium: 4 files/directories

## Task 14: Update End-To-End Coverage For Revamp Flow

**Description:** Update Playwright coverage to use the new visible labels and prove the key revamp promises: filtering preserves the draft, QRIS modal cancel preserves the draft, confirm paid still produces paid detail, and same-day cancellation still works.

**Acceptance criteria:**

- [ ] E2E test searches or filters the catalog after adding a draft line and verifies the draft remains.
- [ ] E2E test opens QRIS payment modal, cancels it, and verifies the current order remains unchanged.
- [ ] E2E test confirms QRIS payment and lands on paid detail with printable action enabled.
- [ ] E2E test cancels a paid order from paid detail and verifies cancelled status.
- [ ] E2E or component coverage verifies non-cancellable cancellation errors remain non-destructive.
- [ ] Existing auth login smoke test still passes with updated shell labels.

**Verification:**

- [ ] Tests pass: `npm run test:e2e`

**Dependencies:** Tasks 9, 10, 11, 12, and 13.

**Files likely touched:**

- `tests/e2e/cashier-order-entry.spec.ts`
- `tests/e2e/auth-login.spec.ts`

**Estimated scope:** Small: 2 files

## Task 15: Capture And Compare Final Revamp UI

**Description:** Run a browser visual verification pass after implementation is complete. Capture the current implementation in the same payment-modal state as `docs/screen-captures/06-order-revamp.png` and compare the screenshot against the initial UI design. The implementation should be visually very similar to the design in layout, hierarchy, spacing, colors, and visible order/payment state.

**Acceptance criteria:**

- [ ] A deterministic browser state is prepared with menu cards, current-order lines, QRIS selected, and the payment modal open.
- [ ] A desktop screenshot is captured and saved under `docs/screen-captures/`, for example `docs/screen-captures/06-order-revamp-implementation.png`.
- [ ] The captured implementation has the same major structure as the design: top POS header, centered search, category and quick-filter controls, image-card grid, right current-order panel, subtotal/tax-or-zero/total area, payment method controls, and QRIS modal.
- [ ] Differences from `docs/screen-captures/06-order-revamp.png` are documented with either fixes or explicit approval if intentionally different.

**Verification:**

- [ ] Browser screenshot captured with Playwright or an equivalent real-browser workflow at a desktop viewport matching the design reference as closely as practical.
- [ ] Manual visual comparison completed against `docs/screen-captures/06-order-revamp.png`.
- [ ] Optional: add or run a focused visual smoke test, for example `npx playwright test tests/e2e/cashier-order-entry-visual.spec.ts --project=chromium`, if a stable test is added.

**Dependencies:** Tasks 0A through 14.

**Files likely touched:**

- `docs/screen-captures/06-order-revamp-implementation.png`
- `tests/e2e/cashier-order-entry-visual.spec.ts` if a repeatable visual smoke test is added
- `docs/reviews/` if visual differences are documented as a review note

**Estimated scope:** Small: 1-3 files

## Task 16: Verify Podman Compose Runtime

**Description:** Build and run the complete application with Podman Compose after implementation, then verify the frontend, backend, static assets, and auth flow work through the containerized Caddy entrypoint. This catches production-container issues that local Vite or unit tests can miss.

**Acceptance criteria:**

- [ ] `podman compose up --build -d` starts backend, frontend, and PostgreSQL containers successfully with a valid local `CASHIER_PIN_HASH`.
- [ ] Backend health is reachable through the frontend container at `http://localhost:8080/api/health`.
- [ ] The app shell is reachable at `http://localhost:8080/`.
- [ ] Static assets used by the revamp, including avatar, menu images, and QRIS image, return image content types from the running frontend container.
- [ ] A local PIN login smoke test succeeds through `http://localhost:8080/api/auth/login`.

**Verification:**

- [ ] Generate local PIN hash: `export CASHIER_PIN_HASH="$(go -C backend run ./cmd/coffee-pos auth hash-pin 123456)"`
- [ ] Start containers: `podman compose up --build -d`
- [ ] Check containers: `podman ps --filter label=io.podman.compose.project=coffee-pos`
- [ ] Check backend health: `curl -i http://localhost:8080/api/health`
- [ ] Check frontend shell: `curl -I http://localhost:8080/`
- [ ] Check assets: `curl -I http://localhost:8080/avatar/cashier.png`, `curl -I http://localhost:8080/menu/americano.png`, and `curl -I http://localhost:8080/qris/static-qris.png`
- [ ] Check login: `curl -i -c /tmp/coffee-pos-cookies.txt -H 'Content-Type: application/json' -d '{"pin":"123456"}' http://localhost:8080/api/auth/login`
- [ ] Stop containers when done if the user does not want them left running: `podman compose down`

**Dependencies:** Tasks 0A through 15.

**Files likely touched:**

- `frontend/Containerfile` if static assets are not included in the production image
- `backend/Containerfile` if backend runtime build fails
- `compose.yaml` if service wiring or health checks fail
- `AGENTS.md` if the documented container runbook needs correction

**Estimated scope:** Small: 0-4 files, depending on issues found

### Checkpoint: Complete Revamp

- [ ] `go -C backend test ./...`
- [ ] `go -C backend test -tags=integration ./...` if Podman/Testcontainers is available.
- [ ] `npm --prefix frontend test`
- [ ] `npm --prefix frontend run check`
- [ ] `npm --prefix frontend run build`
- [ ] `npm run test:e2e`
- [ ] Task 15 visual screenshot comparison is complete and any differences are fixed or explicitly approved.
- [ ] Task 16 Podman Compose runtime verification is complete.

## Risks And Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Backend seed/metadata work expands the revamp beyond frontend-only scope | High | Complete Phase 0 before claiming US-03 acceptance; keep order create/cancel contracts unchanged. |
| Frontend gets ahead of backend metadata | Medium | Keep parser compatibility and fallbacks, but gate full acceptance on `GET /api/pos/menu` returning real categories, images, and filter flags. |
| Tax row in screenshot gets implemented without business approval | High | Keep tax/service charge absent or zero; do not send tax fields to backend. |
| Payment modal refactor breaks idempotent retry | High | Preserve existing pending `clientRequestId` tests and payload-shape tests. |
| Paid-order cancellation regresses during modal/detail refactor | High | Keep Task 11 and Task 14 cancellation coverage for happy path and non-cancellable error path. |
| Layout becomes desktop-only | Medium | Add responsive CSS checkpoint and browser checks at mobile/tablet widths. |
| Search/filter state accidentally clears cart | Medium | Test state preservation after search, category, quick filter, sort, modal cancel, and modal close actions. |

## Parallelization Opportunities

- Tasks 0A and 3 can be done in parallel because backend seed expansion and frontend draft total breakdown are independent.
- Tasks 1 and 3 can be done in parallel after Task 0C is specified because API metadata parsing and draft total breakdown are independent.
- Task 12 can start after Task 4 defines search ownership; it does not need payment modal work.
- Task 14 should wait until labels, modal flow, and cancellation behavior stabilize.
- Task 15 must run last because it verifies the integrated visual result.
- Task 16 must run after Task 15 because it verifies the final build in production-like containers.
- Styling in Task 13 should trail core markup tasks to avoid rework.

## Open Questions

- Should `Tax (11%)` be omitted until confirmed, or should a disabled/zero tax row be visible to match the design shape?
- Should the current-order draft reference like `#ORD-0142` be shown, or should the UI avoid draft identifiers and show only post-payment queue numbers?

## Implementation Verification Notes

- Final implementation screenshot captured at `docs/screen-captures/06-order-revamp-implementation.png`.
- Intentional differences from `docs/screen-captures/06-order-revamp.png`: tax displays as `Rp0` because tax/service policy remains unapproved; no draft order reference is shown before payment; search remains owned by the order catalog instead of the shell; no grid/list toggle is rendered.
- Product image paths are served from `frontend/public/menu/`; some non-original starter images use placeholder copies until final item photography/assets are available.
- Podman Compose runtime verification passed for backend health, frontend shell, avatar/menu/QRIS static assets, and local PIN login through Caddy.

## Planning Verification

- [x] Current backend menu/order/cancel contracts reviewed.
- [x] Current frontend cashier components, helpers, tests, and assets reviewed.
- [x] Tasks have dependencies and verification steps.
- [x] Tasks are scoped to roughly five files or fewer.
- [x] Backend seed/metadata prerequisites are explicit for full spec acceptance.
- [x] Non-functional grid/list toggle is excluded from accepted scope.
- [x] US-06 cancellation regression coverage is explicitly planned.
- [ ] Human has reviewed and approved this plan before implementation.
