# Small Coffee Shop POS: Cashier + Printed Queue Number

Created: 2026-06-28

## Context

The shop serves around 100 cups per day, so the product should not start as a full restaurant POS. The highest-value problem is helping staff take orders quickly, calculate totals consistently, and print a clear paper ticket with a queue number for the barista.

Primary users:

- Cashier: enters orders, confirms modifiers, records payment status, and prints the queue-number ticket.
- Barista: manually manages the drink queue from printed paper tickets.
- Owner/operator: needs a simple daily view of order count and sales without managing a complex back office.

## Problem Statement

How might we help a small coffee shop take orders quickly and print reliable queue-number tickets without introducing heavyweight POS complexity?

## Recommended Direction

Build a counter-first POS that behaves like a cashier order-entry and ticket-printing system, not an enterprise sales platform. The core loop should be:

Customer order -> cashier enters items and modifiers -> order gets a queue number -> ticket is printed -> barista prepares drinks from the paper ticket.

This is the right first direction because a 100-cup/day shop wins more from speed, accuracy, and low training effort than from advanced POS features. The app should replace manual price calculation and inconsistent handwritten order notes first. The barista workflow can stay paper-based until there is clear evidence that a digital queue is needed.

## Success Criteria

- A cashier can submit a common drink order in 15-20 seconds after the customer decides.
- New staff can learn the cashier flow in under 30 minutes.
- Every paid order produces a readable ticket with queue number, drink details, modifiers, notes, and timestamp.
- The barista can work from printed tickets without needing to touch the app.
- The owner can see basic daily totals: orders, cups/items sold, and gross sales.

## Resolved MVP Decisions

- Printer: receipt printer.
- Payment methods: Cash and QRIS only.
- QRIS handling: show a static QRIS image; cashier manually checks and confirms payment.
- Service type labels: not needed for MVP.
- Queue numbering: queue numbers reset every day.
- Initial seeded menu:
  - Americano: Rp18.000.
  - Latte: Rp25.000.
  - Temperature modifier: hot or iced.
  - Sugar modifier: normal, less sugar, or no sugar.

## Idea Variations Considered

1. Full cafe POS with inventory, accounting, loyalty, and payments.
   - Too broad for the first version. It increases setup work before proving the daily workflow.

2. Cashier order entry plus printed queue-number ticket.
   - Recommended. This directly targets the shop's highest-frequency operational pain without adding a second digital workflow for the barista.

3. Ticket-only system without sales capture.
   - Simpler, but too limited. The cashier still needs to calculate orders somewhere else.

4. Customer self-ordering through QR code.
   - Useful later, but not first. It adds menu publishing, customer UX, and payment questions before the staff workflow is proven.

5. Digital barista queue and customer-facing queue display.
   - Useful later, but not first. Printed tickets are enough for the current operating model.

## Key Assumptions to Validate

- [ ] The menu is small and stable enough for fast button-based order entry.
  - Test by mapping the current menu, sizes, add-ons, milk options, ice levels, sugar levels, and common notes.

- [ ] Menu changes are infrequent enough to handle through backend seeders during MVP.
  - Test by reviewing how often prices, items, and modifier options changed in the last 1-2 months.

- [ ] Staff can use a tablet, laptop, or cashier terminal comfortably at the counter.
  - Test by timing order entry with the intended device during a real rush window.

- [ ] Payment can be handled outside the app at first, while the app records payment method and paid/unpaid status.
  - Test whether Cash and QRIS labels, plus manual cashier confirmation, are enough for daily operations.

- [ ] Printed tickets are enough for the barista to manage drink preparation accurately.
  - Test by using printed tickets for one normal day and one rush period, then count missed items, unclear modifiers, and remake incidents.

- [ ] Printer reliability matters more than cloud dashboards in the first version.
  - Test whether the app needs offline/local-network support because internet quality at the shop may be inconsistent.

## MVP Scope

### In Scope

- Seeded menu data:
  - Categories, items, prices, and availability.
  - Initial items: Americano at Rp18.000 and Latte at Rp25.000.
  - Initial modifier groups: temperature with hot/iced options, and sugar with normal/less/no sugar options.
  - Data is created from backend seeders for MVP instead of an in-app management screen.

- Cashier order screen:
  - Fast item selection.
  - Quantity changes.
  - Item modifiers.
  - Generated queue number as the main customer reference.
  - Order notes.
  - Subtotal and total.
  - Cash or QRIS payment method label.
  - Static QRIS image display for QRIS payments.
  - Paid/unpaid status confirmed manually by the cashier.

- Queue-number ticket printing:
  - Daily queue number generation.
  - Queue numbers reset every day.
  - Receipt-printer friendly browser ticket or receipt.
  - Ticket includes queue number, order details, modifiers, notes, total, payment method, and timestamp.

- Basic daily reporting:
  - Total orders.
  - Total items/cups.
  - Gross sales.
  - Sales grouped by payment method.

### Out of Scope for MVP

- Digital barista queue module.
- Customer-facing queue display.
- Menu management screen.
- Inventory and ingredient stock tracking.
- Accounting exports and tax filing.
- Payment gateway integration.
- Loyalty program.
- Customer accounts.
- Online ordering.
- Delivery app integration.
- Staff scheduling and payroll.
- Multi-branch management.
- Advanced table management.

## Core Screens

1. Cashier Order Screen
   - Main working screen.
   - Optimized for speed and repeated use.
   - Shows menu buttons, current order, modifiers, total, payment status, and submit button.

2. Daily Summary Screen
   - Lightweight end-of-day view for order count, cups sold, gross sales, and payment method totals.

3. Optional Later Queue Screen
   - Digital barista or customer queue display.
   - Should be added only if paper tickets create measurable problems during service.

## Suggested MVP Data Model

- MenuCategory: name, sort order, active status.
- MenuItem: category, name, price, active status.
- ModifierGroup: name, selection type, required/optional.
- ModifierOption: group, name, price delta.
- Order: queue number, customer label, payment method, payment status, total, created time.
- OrderItem: order, menu item, quantity, modifiers, notes, line total.
- DailySummary: date, order count, item count, gross sales, payment method totals.

## Initial Seeder Data

- Category: Coffee.
- MenuItem: Americano, price Rp18.000, active.
- MenuItem: Latte, price Rp25.000, active.
- ModifierGroup: Temperature, required single select.
- ModifierOption: Hot, price delta Rp0.
- ModifierOption: Iced, price delta Rp0.
- ModifierGroup: Sugar, required single select.
- ModifierOption: Normal, price delta Rp0.
- ModifierOption: Less sugar, price delta Rp0.
- ModifierOption: No sugar, price delta Rp0.

## First Build Sequence

1. Build static menu order entry with hardcoded sample items.
2. Replace hardcoded sample items with backend seeders for menu items and modifiers.
3. Add order saving and generated queue numbers.
4. Add browser-printable queue-number tickets.
5. Add daily summary.
6. Test during one normal day and one rush period before adding advanced features.

## Not Doing and Why

- Inventory tracking: likely high setup cost and not required to prove the cashier-to-ticket workflow.
- Digital queue module: not needed while barista can reliably work from printed tickets.
- Customer queue display: not needed until customers or staff repeatedly ask for visible order status.
- Menu management UI: not needed while menu data can be seeded from the backend for MVP.
- Payment integration: useful later, but static QRIS plus manual cashier confirmation is enough for MVP.
- Dine-in, takeaway, and delivery labels: not needed for current operations.
- Loyalty and promotions: not necessary for serving the next 100 cups more reliably.
- Online ordering: changes the product into a customer-facing ordering system before the staff workflow is solid.
- Multi-branch support: premature for a single small shop.
- Complex restaurant table management: coffee orders are usually faster and lighter than full-service restaurant orders.
