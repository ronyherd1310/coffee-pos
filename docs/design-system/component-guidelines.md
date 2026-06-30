# Component Guidelines

These components adapt the supplied styling to the Coffee POS MVP. Each component should be implemented with semantic HTML first, then enhanced with Preact state where needed.

## Background

Use the app gradient for authenticated workflows and login screens.

Structure:

- `body` uses `--gradient-app`.
- Main workflow uses a translucent or solid app shell depending on density.
- Dense screens such as reports and order tables may use solid `--color-surface` panels on top of the gradient.

Implementation notes:

- Keep background fixed to the viewport.
- Do not add decorative blobs, stock illustrations, or heavy SVG background art.
- Use `backdrop-filter: blur(18px)` only where text contrast remains strong; provide a solid fallback.

## Buttons

The reference image relies on pill checkout buttons and circular icon controls. Coffee POS needs both plus compact secondary buttons.

Variants:

- Primary: green pill, white text, used for checkout, confirm paid, print, and save.
- Secondary: white surface, navy text, green border or subtle shadow.
- Ghost: transparent, navy or muted icon/text, used for low-risk navigation.
- Danger: red text or red filled button only for destructive confirmation.
- Icon: circular `44px` button with an accessible label.

States:

- Focus: `--shadow-focus`.
- Disabled: muted text, muted background, no shadow, plus nearby reason when recovery is not obvious.
- Loading: keep button width stable and change label to the current action, such as `Confirming...`.

## Text Field

Use pill fields for search and standard rounded fields for forms.

Anatomy:

- Visible label above the field for forms.
- Optional leading icon for search, PIN, or date.
- Input surface `--color-surface`.
- Helper or error text below the field.

Sizing:

- Height: `44px` minimum.
- Padding: `12px 16px`.
- Radius: `--radius-control`.

States:

- Default: white surface, subtle border.
- Focus: green border and `--shadow-focus`.
- Error: red border, red helper text, and `aria-invalid="true"`.
- Disabled: muted background and no pointer interaction.

POS examples:

- Cashier PIN entry.
- Menu search.
- Order note.
- Manual amount or quantity input where needed.

## Dropdown

Use native `select` for simple lists. Use a custom popover only when search, icons, or multi-column options are required.

Anatomy:

- Label.
- Trigger styled like a text field.
- Chevron icon.
- Popover with white surface, `8px` card radius, border, and shadow.

Behavior:

- Opens with Enter, Space, ArrowDown, or pointer.
- Closes on Escape and outside click.
- Supports ArrowUp and ArrowDown movement.
- Selected option is marked with green text, check icon, or selected indicator.

POS examples:

- Payment method when not presented as radio options.
- Report period.
- Order status filter.

## Modal Dialog

Use modal dialogs for decisions that interrupt the cashier flow. Use bottom sheets on small screens when the action benefits from thumb reach.

Anatomy:

- Scrim: `rgb(17 20 58 / 36%)`.
- Dialog surface: `--color-surface`, `--radius-dialog`, `--shadow-shell`.
- Header with title and icon-only close button.
- Body with the decision content.
- Footer with primary action on the right and secondary action on the left.

Behavior:

- Trap focus while open.
- Return focus to the trigger on close.
- Close with Escape unless the dialog is in a blocking network state.
- Make destructive actions explicit, such as `Cancel paid order`.

POS examples:

- Confirm paid order.
- Confirm cancellation.
- QRIS payment proof reminder.
- Session expired notice.

## Error State

The reference image does not show errors, so Coffee POS adds an explicit error language.

Types:

- Field error: directly below the field, short and specific.
- Inline panel error: inside the affected card or section.
- Page error: for failed page-level data load.
- Toast error: for transient failures after an action.

Visual treatment:

- Use `--color-danger-soft` background, `--color-danger` text/icon, and a red border.
- Pair every error with a text label and recovery action when possible.
- Avoid generic text such as `Something went wrong` when the system knows the failed action.

POS examples:

- `PIN must be 6 digits.`
- `Cannot confirm payment with an empty cart.`
- `Order was paid but the ticket did not print. Reprint from Today's Orders.`

## Card

Cards are repeated content units, not page sections.

Visual treatment:

- Surface: `--color-surface-raised`.
- Radius: `--radius-card`.
- Shadow: `--shadow-card` only when the card sits directly on the gradient.
- Border: `1px solid var(--color-border)` for dense lists and tables.

Patterns:

- Menu item card: drink name, price, short modifier defaults, add button.
- Cart item card: drink name, modifiers, quantity stepper, line total.
- Order card: queue number, status, item count, total, time.
- Report card: metric label, value, comparison note if available.

Rules:

- Keep cards compact enough for cashier scanning.
- Use images only when they help identify menu items; do not require images for the MVP.
- Do not nest cards inside cards.

## Options

Use option controls for mutually exclusive choices and fast filtering.

Patterns:

- Category chips: horizontal pill group with one selected item.
- Segmented control: equal-width choices for Cash or QRIS.
- Radio tiles: larger choices with title, description, and icon.
- Modifier chips: Temperature, sugar, and size options per item.

Visual treatment:

- Selected: green fill with white text, or green border plus soft green fill for larger tiles.
- Unselected: white surface, muted border, navy text.
- Focus: visible green ring.
- Disabled: muted text and short helper reason.

Behavior:

- Use radio semantics for mutually exclusive groups.
- Use checkbox semantics for multi-select options.
- Selected state must be available to assistive tech.

## Toast

Use toasts for action feedback that does not require a decision.

Placement:

- Mobile: bottom center above primary navigation.
- Tablet and desktop: top right.

Visual treatment:

- Surface: white or status-soft background.
- Radius: `--radius-card`.
- Shadow: `--shadow-float`.
- Include status icon, short message, and optional action.

Behavior:

- Auto-dismiss success/info toasts after 4 seconds.
- Keep error toasts visible until dismissed or replaced by a more specific inline error.
- Pause dismissal on hover or focus.

POS examples:

- `Order paid. Queue No. 001 created.`
- `Ticket sent to print.`
- `Backend unavailable. Retrying...`

## Table

Use tables for Today's Orders, reports, and audit-style lists.

Visual treatment:

- Solid `--color-surface` panel.
- Header text muted, uppercase optional only for short labels.
- Row border `1px solid var(--color-border)`.
- Hover/focus row background `--color-surface-muted`.
- Numeric columns right-aligned.

Behavior:

- Keep column labels visible on desktop.
- On mobile, either reduce columns to the most important fields or use stacked rows with the same source data.
- Rows that open details must be keyboard focusable and have an explicit accessible name.

POS columns:

- Today's Orders: Queue, Time, Items, Payment, Status, Total, Actions.
- Reports: Date, Orders, Cups, Cash, QRIS, Gross sales, Cancelled.

## Pagination

Use pagination for order history and reports beyond the current day. Today's Orders should prefer filtering and short lists before pagination.

Anatomy:

- Previous icon button.
- Page number buttons or compact `Page 1 of 4` label.
- Next icon button.
- Optional page-size dropdown for desktop reports.

Visual treatment:

- Buttons are `36px` to `40px` square.
- Current page uses green fill or green border with soft fill.
- Disabled previous/next buttons are muted and non-interactive.

Behavior:

- Use URL state for shareable report pages when history screens exist.
- Announce page changes with `aria-live="polite"` if content updates without navigation.

## Date Picker

Use a date picker for reports and historical order lookup. For MVP daily reports, default to the Asia/Jakarta business date.

Anatomy:

- Text field trigger with calendar icon.
- Popover calendar with month navigation.
- Day grid using button elements.
- Optional quick actions: Today, Yesterday, This week.

Visual treatment:

- Popover uses `--color-surface`, `--radius-card`, border, and `--shadow-card`.
- Selected day uses green fill and white text.
- Today's date uses green border if not selected.
- Disabled dates use muted text.

Behavior:

- Accept keyboard input in `YYYY-MM-DD` format when practical.
- Calendar supports Arrow keys, Home, End, PageUp, PageDown, Enter, and Escape.
- Store and display dates in Asia/Jakarta for business reports and queue resets.
- Validate invalid dates inline, not with a toast.

## Empty And Loading States

Use skeleton blocks for content loading, not spinners for entire cashier screens.

Examples:

- Empty cart: show a compact prompt near the cart total area.
- Empty order list: show `No paid orders for today yet.` with a New Order action.
- Loading report: skeleton rows that match the table shape.

## Accessibility Baseline

- Interactive controls use native elements wherever possible.
- Icon-only buttons have `aria-label`.
- Form fields have visible labels.
- Dialogs have `aria-modal="true"` and labelled titles.
- Error text is associated with the failed control through `aria-describedby`.
- Color state is always paired with text, icon, or both.
