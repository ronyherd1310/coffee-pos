# Spec: Small Coffee Shop POS MVP

Source idea: [Small Coffee Shop POS: Cashier + Printed Queue Number](../ideas/small-coffee-shop-pos-cashier-printed-queue-number.md)

Created: 2026-06-28

## Assumptions

1. This is a web application used by the cashier in a browser, not a native mobile app.
2. The MVP is intended for a single coffee shop, not multiple branches.
3. MVP authentication uses one predefined 6-digit cashier PIN, not individual user accounts.
4. The PIN is verified by the backend; it must not be embedded in the frontend bundle.
5. Menu data is seeded from the backend; there is no menu management UI in MVP.
6. The recommended receipt printer is Epson TM-T82III using 80mm thermal paper.
7. The receipt printer can print browser-generated receipts through the operating system print dialog or browser print flow.
8. The app uses Asia/Jakarta as the business timezone for timestamps, daily queue reset, and daily reports.
9. The default printed shop name is `Coffee Shop`, configurable with `SHOP_NAME`.
10. QRIS uses an owner-provided static image at `frontend/public/qris/static-qris.png`.
11. The frontend runs as a separate small cloud container with about 0.1 CPU and 512 MB memory.
12. The backend API runs as a separate small cloud container with about 0.1 CPU and 512 MB memory.
13. The database runs separately with about 1 CPU and 512 MB memory.
14. The frontend production container serves static files only; Node.js is used at build time, not runtime.
15. The tech stack has not been scaffolded yet, so the stack and commands below are the recommended implementation baseline for those resource limits.
16. The order-entry revamp design in `docs/screen-captures/06-order-revamp.png` is the target visual and interaction direction for the cashier ordering screen.
17. Product, QRIS, and cashier avatar imagery are static frontend assets for MVP and must be included in the production frontend container build.
18. Numeric totals, menu volume, and visible draft reference values in design screenshots are illustrative unless captured as explicit requirements below.

## Objective

Build a small coffee shop POS MVP focused on simple cashier PIN authentication, cashier order entry, manual payment confirmation, receipt-printer tickets with daily queue numbers, and basic daily sales reporting.

The product is for a shop serving around 100 cups per day. It should optimize for fast counter operation and low training effort instead of full POS breadth.

The cashier order-entry experience should follow the order revamp design: a visual menu catalog on the left, a persistent current-order panel on the right, fast search/filter controls, image-led item cards, compact quantity controls, and a modal payment confirmation flow.

### Primary Users

- Cashier: signs in with a 6-digit PIN, enters orders, confirms modifiers, records Cash or QRIS payment, and prints queue-number tickets.
- Barista: prepares drinks from printed paper tickets and does not use the app.
- Owner/operator: reviews daily totals for orders, items/cups, gross sales, and payment methods.

## Tech Stack

Recommended baseline for the target resource limits:

- Frontend app: Preact + Vite + TypeScript.
- Frontend runtime: static files served by Caddy in the frontend container, with `/api/*` reverse-proxied to the backend API where possible.
- Frontend styling: plain CSS or CSS modules. Avoid a large runtime component framework for MVP.
- Backend API service: Go 1.26.x, built as a single Linux binary.
- Backend HTTP server: Go standard library `net/http`, exposing JSON API endpoints.
- Database: PostgreSQL, using the provider-supported current stable version.
- Data access: `database/sql` with the PostgreSQL driver, generated query wrappers from sqlc, and explicit SQL migrations.
- Container runtime and image builds: Podman with service-specific `Containerfile`s.
- Printing: browser print flow from the Preact frontend with receipt-printer friendly CSS.
- Testing: Go `testing` package for backend unit tests; Testcontainers for Go with PostgreSQL module for database integration tests; frontend build/type checks; Playwright only for local/CI browser workflow tests, not as a production dependency.

### Stack Rationale

- Preact + Vite is the selected frontend because it keeps a React-like component model for a small team while still producing static production assets.
- The frontend container should serve built static files only. It should not run a persistent Node.js frontend server in production.
- Caddy should expose the browser-facing origin, serve static assets, and proxy `/api/*` to the backend API so session cookies can stay same-origin.
- Preact is appropriate for cashier order entry because the cart, modifiers, QRIS display, and ticket preview need fast client-side interaction.
- Caddy is the default static file server because its file server is straightforward and production-ready; Nginx is an acceptable substitute if the deployment platform standardizes on it.
- Go is a better fit than a Node.js API for the backend service on a 0.1 CPU / 512 MB container because the MVP can run as one compiled API process.
- PostgreSQL is a better fit than SQLite for the stated deployment because the database has its own cloud resource allocation. SQLite would be simpler locally but creates persistence, backup, and multi-container deployment concerns in the cloud.
- Podman is the default local container runtime because it can build and run OCI container images without requiring a long-running Docker daemon.
- Testcontainers for Go is the default integration test harness for database-backed backend tests so repository and migration behavior can be tested against a real PostgreSQL container instead of mocks.
- Testcontainers-backed tests should run against Podman locally and in CI when a compatible Podman socket is available.
- Avoid Next.js SSR, Remix, SvelteKit SSR, Nuxt, Prisma, large component frameworks, and extra background workers for MVP because they add memory, CPU, and operational surface area that the product does not need yet.

Reference docs:

- Go `go build` compiles packages into an executable: <https://go.dev/doc/tutorial/compile-install>
- Go `net/http`: <https://pkg.go.dev/net/http>
- Preact getting started with Vite: <https://preactjs.com/guide/v10/getting-started/>
- Vite production build: <https://vite.dev/guide/build>
- Vite getting started templates: <https://vite.dev/guide/>
- Caddy static file server: <https://caddyserver.com/docs/caddyfile/directives/file_server>
- Caddy reverse proxy: <https://caddyserver.com/docs/caddyfile/directives/reverse_proxy>
- Nginx static content docs: <https://docs.nginx.com/nginx/admin-guide/web-server/serving-static-content/>
- Podman documentation: <https://docs.podman.io/>
- Podman build: <https://docs.podman.io/en/stable/markdown/podman-build.1.html>
- Podman run: <https://docs.podman.io/en/latest/markdown/podman-run.1.html>
- Testcontainers for Go: <https://golang.testcontainers.org/>
- Testcontainers for Go PostgreSQL module: <https://golang.testcontainers.org/modules/postgres/>
- Testcontainers for Go with Podman: <https://golang.testcontainers.org/system_requirements/using_podman/>
- Epson TM-T82III product page: <https://www.epson.com.sg/For-Work/Printers/POS-Printers/Epson-TM-T82III-POS-Printer/p/C31CH51542>
- Epson TM-T82III technical reference: <https://download4.epson.biz/sec_pubs/bs/pdf/TM-T82III_trg_en_revF.pdf>
- sqlc documentation: <https://docs.sqlc.dev/>
- PostgreSQL resource settings: <https://www.postgresql.org/docs/current/runtime-config-resource.html>
- PostgreSQL connection settings: <https://www.postgresql.org/docs/current/runtime-config-connection.html>

### Resource Constraints

- Frontend container:
  - Target memory limit: 512 MB.
  - Target CPU limit: 0.1 CPU.
  - Run Caddy as the only production process.
  - Serve built Preact assets, CSS, and the static QRIS image.
  - Reverse-proxy `/api/*` to the backend API service if the deployment platform allows service-to-service networking.
  - Do not run Node.js, Vite dev server, Playwright, or file watchers in the production container.

- Backend API container:
  - Target memory limit: 512 MB.
  - Target CPU limit: 0.1 CPU.
  - Run one Go API process in production.
  - Expose only the API needed by the frontend and health checks.
  - Do not serve frontend build tools or browser test tooling.

- Database:
  - Target memory: 512 MB.
  - Target CPU: 1 CPU.
  - Keep application database pool small: start with `max_open_connections = 3` and `max_idle_connections = 1`.
  - Keep PostgreSQL connection count low; increasing `max_connections` increases shared resource allocation.
  - Start with conservative PostgreSQL memory settings such as `shared_buffers = 128MB` and `work_mem = 2MB` to `4MB`, if the provider allows tuning.

The selected stack should still be confirmed before implementation. If the project is scaffolded with a different stack, update this spec before coding.

## Commands

Target commands after project scaffold exists:

```bash
# Install backend dependencies
go -C backend mod download

# Run backend API locally
go -C backend run ./cmd/coffee-pos serve

# Run backend/database seeders
go -C backend run ./cmd/coffee-pos db seed

# Run database migrations
go -C backend run ./cmd/coffee-pos db migrate

# Run backend unit tests and non-container integration tests
go -C backend test ./...

# Run backend Testcontainers-backed integration tests
go -C backend test -tags=integration ./...

# Run backend static checks
go -C backend vet ./...

# Format backend code
gofmt -w backend/

# Build backend production binary
go -C backend build -o ../bin/coffee-pos ./cmd/coffee-pos

# Build backend container image with Podman
podman build -f backend/Containerfile -t coffee-pos-backend:dev backend

# Install frontend dependencies
npm --prefix frontend install

# Run frontend dev server locally
npm --prefix frontend run dev

# Build frontend static assets
npm --prefix frontend run build

# Build frontend container image with Podman
podman build -f frontend/Containerfile -t coffee-pos-frontend:dev frontend

# Run frontend type/lint checks
npm --prefix frontend run check

# Run browser workflow tests locally or in CI
npx playwright test
```

Current repository status: the app has been scaffolded, and command availability may differ as MVP slices land. Use this section as the target command contract and `AGENTS.md` as the current agent runbook.

## Architecture

Use hexagonal architecture for the backend so business rules are isolated from HTTP, PostgreSQL, session storage, and other infrastructure.

Backend dependency direction:

```text
HTTP / CLI adapters  ->  application use cases  ->  domain model and domain services
PostgreSQL adapters  ->  application ports      ->  domain model and domain services
```

Rules:

- Domain packages contain entities, value objects, domain services, validation rules, and pure calculations.
- Application packages contain use cases and port interfaces. Use cases orchestrate transactions, repositories, clocks, queue-number allocation, session operations, and domain services.
- Adapter packages implement ports for HTTP, PostgreSQL, password/PIN hashing, session cookies/storage, rate limiting, clocks, and configuration.
- Domain packages must not import `net/http`, `database/sql`, generated sqlc query packages, environment/config packages, or frontend/API DTO packages.
- HTTP handlers validate and translate requests, call application use cases, and translate use case results into JSON responses.
- PostgreSQL adapters use sqlc-generated queries and explicit transactions, but database row types do not leak into domain or application APIs.
- Queue-number allocation is exposed as an application port backed by PostgreSQL transaction-safe storage so duplicate same-day queue numbers cannot occur under concurrent paid-order creation.
- Time-dependent behavior uses a clock port so Asia/Jakarta business-date logic and session expiration can be tested deterministically.
- Frontend API contracts should be defined separately from backend domain types. Do not shape the domain model around wire-format convenience.

Backend package naming should stay practical and small. Prefer packages such as `internal/domain/orders`, `internal/app/orders`, and `internal/adapters/postgres` over deeply nested abstractions.

## Project Structure

Target structure after scaffold:

```text
backend/
  Containerfile           -> Backend production image build
  cmd/coffee-pos/         -> Backend API entrypoint and CLI commands
  internal/domain/auth/   -> PIN/session domain rules and value objects
  internal/domain/menu/   -> Menu, modifier groups, and modifier option rules
  internal/domain/orders/ -> Order, payment, queue-number, cancellation, and total rules
  internal/domain/reports/ -> Daily summary aggregation rules
  internal/domain/money/  -> Rupiah values and money helpers
  internal/app/auth/      -> Login, logout, session-check use cases and ports
  internal/app/menu/      -> Menu read and seeding use cases and ports
  internal/app/orders/    -> Create paid order, list/detail, reprint, cancel use cases and ports
  internal/app/reports/   -> Daily summary use cases and ports
  internal/adapters/http/ -> HTTP router, handlers, middleware, request/response DTOs
  internal/adapters/postgres/ -> Database connection, transactions, sqlc-backed port implementations
  internal/adapters/security/ -> PIN hash verification, session cookie/storage, rate limiting adapters
  internal/adapters/clock/ -> Real clock implementation for production
  internal/config/        -> Environment/config loading and validation
  internal/seed/          -> Seeder wiring that calls app/menu use cases
  migrations/             -> SQL migrations
  queries/                -> SQL files used by sqlc
frontend/
  Containerfile           -> Frontend static Caddy image build
  src/                    -> Preact application source
  src/features/auth/      -> PIN login UI and session checks
  src/features/cashier/   -> Cashier order-entry UI
  src/features/orders/    -> Today's orders list and paid order detail UI
  src/features/reports/   -> Daily summary UI
  src/features/printing/  -> Ticket preview and print UI
  src/lib/                -> Frontend API client, formatting, validation helpers
  public/avatar/          -> Static staff/avatar imagery used by the POS shell
  public/menu/            -> Static product images used by menu cards
  public/qris/static-qris.png -> Static QRIS image asset for MVP
  dist/                   -> Built static assets, generated by Vite and not edited manually
  Caddyfile               -> Static frontend server config and /api reverse proxy
tests/e2e/                -> Browser workflow tests
testcontainers/           -> Shared test container helpers if needed
docs/
  ideas/                  -> Idea refinement documents
  specs/                  -> Specifications
```

Existing structure:

```text
docs/ideas/               -> Idea document exists
docs/specs/               -> This spec lives here
docs/scripts/             -> Local helper scripts
```

## Functional Requirements

### Cashier Authentication

User Stories:

#### US-01: Sign In With Cashier PIN

As a cashier, I can sign in with a predefined 6-digit PIN so only staff can access the POS screens.

Acceptance:

- Cashier can enter exactly 6 digits.
- Correct PIN creates an authenticated session.
- Incorrect PIN shows a generic error without revealing whether any individual digit was correct.
- Login attempts are rate-limited to reduce brute-force attempts.
- Authenticated cashier can access the cashier order screen and daily summary.
- Unauthenticated browser sessions are redirected to the PIN login screen.

Requirements:

- Authentication uses one predefined 6-digit cashier PIN for MVP.
- PIN validation happens only on the backend.
- The frontend must never contain the real PIN or a PIN hash.
- Production configuration should provide a slow hash of the PIN, not the plaintext PIN.
- Backend must validate PIN format as exactly 6 numeric digits before verification.
- Backend must create a server-side session after successful PIN verification.
- Session cookie must be `HttpOnly`, `Secure` in production, and `SameSite=Lax` or stricter.
- Session expires at the end of the Asia/Jakarta business day or after 12 hours, whichever comes first.
- Cashier can log out, which invalidates the current session.
- Auth endpoints must use rate limiting, starting with 5 failed attempts per 5 minutes per client identifier.
- All POS API endpoints are protected except health checks, static assets, and the PIN login endpoint.
- PIN rotation is manual for MVP by changing the backend environment configuration and redeploying/restarting the service.
- No individual cashier identity, role management, password reset, or PIN management UI is included in MVP.

Wireframe:

```text
+------------------------------------------+
| Coffee POS                               |
|                                          |
| Cashier PIN                              |
| [ _  _  _  _  _  _ ]                     |
|                                          |
| [Sign In]                                |
|                                          |
| Error: Invalid PIN. Try again.           |
+------------------------------------------+
```

### Seeded Menu

User Stories:

#### US-02: Seed Initial Menu

As an operator/developer, I can run a backend seeder so the MVP starts with the approved menu items and modifier options without requiring a menu management screen.

Acceptance:

- Seeder creates the menu categories needed by the order revamp design.
- Seeder creates the approved visible catalog items with rupiah prices.
- Seeder creates required Temperature options for drinks where temperature applies: Hot and Iced.
- Seeder creates required Sugar options for drinks where sugar applies: Normal, Less sugar, and No sugar.
- Seeder can attach optional display metadata such as image path, popularity, promotional, iced, low-sugar, or new-arrival flags.
- Running the seeder multiple times does not create duplicate menu data.

Requirements:

- The backend seeder creates the initial menu data.
- MVP menu categories:
  - Coffee.
  - Tea.
  - Snacks.
  - Seasonal.
- MVP menu items:
  - Americano, Rp18.000.
  - Latte, Rp25.000.
  - Cappuccino, Rp25.000.
  - Mocha, Rp28.000.
  - Matcha Latte, Rp28.000.
  - Flat White, Rp24.000.
  - Caramel Latte, Rp28.000.
  - Espresso, Rp15.000.
  - Iced Tea, Rp15.000.
  - Chocolate, Rp25.000.
  - Croissant, Rp20.000.
  - Muffin, Rp20.000.
- MVP modifier groups:
  - Temperature: required single select with Hot and Iced for drink items where temperature applies.
  - Sugar: required single select with Normal, Less sugar, and No sugar for drink items where sugar applies.
- Modifier options have price delta Rp0 for MVP.
- Menu items may include static image paths served from the frontend public assets, for example `/menu/americano.png`.
- Menu metadata may include `bestSeller`, `promo`, `iced`, `lowSugar`, and `newArrival` flags for UI filtering and badges.
- The `All` menu tab is a derived frontend view, not a persisted category.
- No menu management screen is included in MVP.

Wireframe:

Not applicable as a user-facing screen for MVP. Seeded menu data appears in the Cashier Order Entry screen as menu buttons and modifier controls.

### Cashier Order Entry

User Stories:

#### US-03: Create Order From Cashier Screen

As a cashier, I can select drinks, quantities, modifiers, and notes from one screen so I can prepare an order quickly while the customer is waiting.

Acceptance:

- Cashier can add approved menu items from the seeded visual catalog.
- Cashier can search menu items by name from the top search input.
- Cashier can filter menu items by category tabs: All, Coffee, Tea, Snacks, and Seasonal.
- Cashier can use quick filters for Best Seller, Iced, Low Sugar, and New Arrival when matching item metadata exists.
- Cashier can sort the catalog by Popular by default.
- Cashier can view menu items as image cards; list view can be added if supported by the UI toggle.
- Cashier can adjust item quantity.
- Cashier must choose one Temperature option and one Sugar option per drink item when those modifier groups apply.
- Cashier can add the same drink more than once with different modifiers as separate order lines.
- Cashier can add an optional order note.
- Total updates from item price, quantity, modifier price deltas, and any confirmed tax/service charge configuration.

#### US-04: Confirm Cash Payment

As a cashier, I can choose Cash and manually confirm payment so the order can receive a queue number and be printed.

Acceptance:

- Cash is available as a payment method.
- The order cannot be printed until payment is confirmed.
- Cashier must confirm the paid action in a confirmation dialog before the order is persisted.
- Confirming payment persists the order and assigns the next Asia/Jakarta daily queue number.

#### US-05: Confirm QRIS Payment

As a cashier, I can choose QRIS, show a static QRIS image, and manually confirm payment after checking the customer's payment proof.

Acceptance:

- QRIS is available as a payment method.
- Choosing QRIS displays `/qris/static-qris.png`.
- The system does not integrate with a payment gateway.
- Cashier must confirm the paid action in a confirmation dialog before the order is persisted.
- Confirming payment persists the order and assigns the next Asia/Jakarta daily queue number.

#### US-06: Cancel Accidental Paid Order

As a cashier, I can cancel an accidental paid order on the same day so sales totals stay accurate without deleting audit history.

Acceptance:

- Cashier can cancel a paid order from the same Asia/Jakarta business day.
- Cashier must confirm cancellation in a confirmation dialog.
- Cancelled order remains stored with cancelled status and cancellation timestamp.
- Cancelled order is excluded from sales totals.
- Deleted paid orders are not supported in MVP.

Requirements:

- The cashier can create a new order from the cashier screen.
- The cashier can add approved seeded menu items from visual menu cards.
- The menu catalog should use a desktop/tablet layout similar to `docs/screen-captures/06-order-revamp.png`: top app bar, centered search, category tabs, quick filters, sort control, grid/list toggle, product-card grid, and persistent current-order panel.
- Menu cards show product image, item name, formatted rupiah price, optional badge or filter indicator, and an accessible add button.
- Search filters menu items by name without clearing the current order.
- Category tabs filter by item category; `All` shows all active menu items.
- Quick filters use seeded item metadata and can be combined with the active category where practical.
- The default catalog sort is Popular; unsupported sort options must not be shown.
- Pagination or incremental loading may be used when the catalog exceeds the visible grid, but it must not interrupt current order state.
- The cashier can set item quantity.
- Each drink order item requires one Temperature option and one Sugar option when those modifier groups apply.
- Modifier selection is per order line, not global for the whole order.
- Adding an item with required modifiers must open a customization step before the line is added, unless sensible defaults have been explicitly approved.
- Adding the same menu item with different modifiers creates separate order lines.
- The cashier can add an optional text note to the order.
- The order note field has a 120-character limit and shows a remaining or used character count.
- The app calculates subtotal and total from item price, quantity, and modifier deltas.
- The order panel displays subtotal, optional tax/service charge, and total as separate rows.
- The order revamp design shows `Tax (11%)`; implementation must confirm the shop's actual tax/service policy before hard-coding this rate. If enabled, tax is calculated with integer rupiah rounding rules and included in the persisted paid order total.
- Currency is displayed in Indonesian rupiah, with no decimal cents.
- The app supports Cash and QRIS payment methods only.
- If QRIS is selected, the cashier screen displays `/qris/static-qris.png`.
- The current-order panel displays one row per order line with quantity, thumbnail, item name, modifier summary, line total, quantity stepper, and remove action.
- The `Proceed to Payment` or `Confirm Paid` action is disabled until the cart has at least one valid order line and a payment method.
- The `Print Ticket` action is not available on unpaid draft orders.
- Payment confirmation is manual. The cashier marks an order as paid after checking Cash or QRIS payment.
- Confirming payment shows a confirmation dialog or modal with order total and payment method before persisting the paid order.
- The QRIS confirmation modal shows the QRIS image, total amount, payment instructions, `Confirm Paid`, and `Cancel` controls.
- Closing or cancelling the payment modal returns the cashier to the unchanged current order.
- Unpaid orders are frontend-only drafts and are not persisted.
- Confirming payment persists the order, assigns the queue number, opens the Paid Order Detail screen, and makes the ticket printable.
- Any visible draft reference such as `#ORD-0142` in the design is a client-side draft label only unless the backend explicitly creates a persisted order. Customer-facing and post-payment references continue to use queue number.
- Accidental paid orders are corrected by same-day cancellation, not deletion.
- Cancelling a paid order shows a confirmation dialog before changing status.
- Cancelled orders remain in the database for audit and are excluded from sales totals.
- The app must not require dine-in, takeaway, or delivery labels.

Design reference:

- Canonical screenshot: `docs/screen-captures/06-order-revamp.png`.
- The first desktop viewport should show the brand mark/title, menu search, `New Order`, `Today's Orders`, cashier/avatar affordance, catalog controls, menu grid, and current-order panel without requiring navigation to a separate page.
- The current-order panel stays visible on desktop and tablet widths where space allows.
- Payment confirmation appears as a focused modal over the ordering screen, not as a separate route.

Layout sketch:

```text
+--------------------------------------------------------------------------------+
| POS Coffee POS             Search menu item...     [New Order] [Today's Orders] |
+-----------------------------------------------------+--------------------------+
| [All] [Coffee] [Tea] [Snacks] [Seasonal]            | Current Order            |
| Quick Filters: [Best Seller] [Iced] [Low Sugar]     | 1x Latte        Rp25.000 |
| Sort: Popular                         [Grid] [List] |    Hot, Less sugar       |
|                                                     |    [-] [1] [+] [Remove] |
| [photo] Americano Rp18.000 [+]                      | 1x Croissant    Rp20.000 |
| [photo] Latte     Rp25.000 [+]                      |    [-] [1] [+] [Remove] |
| [photo] Iced Tea  Rp15.000 [+]                      |                          |
| [photo] Croissant Rp20.000 [+]                      | Order note [__________]  |
|                                                     | Subtotal       Rp75.000 |
| Pagination: [<] [1] [2] [3] [...] [>]               | Tax (if set)   Rp8.250  |
|                                                     | Total          Rp83.250 |
|                                                     | Payment [Cash] [QRIS]   |
|                                                     | [Proceed to Payment]    |
+-------------------------------+------------------------------------------------+
```

QRIS payment state:

```text
+------------------------------------------------+
| Payment: QRIS                              [x] |
|                                                |
| Total Amount                                   |
| Rp83.250                                       |
|                                                |
|              [ Static QRIS Image ]             |
|                                                |
| Scan the QR code using an e-wallet or banking. |
|                                                |
| [Confirm Paid]                                 |
| [Cancel]                                       |
+------------------------------------------------+
```

Confirm paid dialog:

```text
+------------------------------------------------+
| Confirm Payment                                |
|                                                |
| Total: Rp36.000                                |
| Payment: Cash                                  |
|                                                |
| This will create queue number 001 and cannot   |
| be edited after printing.                      |
|                                                |
| [Back]                         [Confirm Paid]  |
+------------------------------------------------+
```

### Today's Orders

User Stories:

#### US-07: Find Today's Paid Order

As a cashier, I can open a list of today's persisted orders so I can find a paid order for reprint or cancellation.

Acceptance:

- Today's Orders is reachable from the cashier screen header.
- List includes paid and cancelled orders from the current Asia/Jakarta business date.
- List shows queue number, time, item count, payment method, total, and status.
- Cashier can search or filter by queue number.
- Selecting an order opens the Paid Order Detail screen.

#### US-08: Reprint Paid Order Ticket

As a cashier, I can reprint a ticket for an existing paid order so service can continue after printer paper, jam, or handling mistakes.

Acceptance:

- Paid Order Detail has a `Print Ticket` or `Reprint Ticket` action.
- Reprinted ticket uses the original queue number and original paid timestamp.
- Reprinting does not create a new order or new queue number.

Requirements:

- Today's Orders is the canonical lookup flow for persisted orders.
- After payment confirmation, the app opens Paid Order Detail for the newly paid order.
- Paid Order Detail is the single canonical post-payment screen.
- Paid Order Detail shows queue number, status, paid timestamp, payment method, total, item lines, modifiers, and notes.
- Paid Order Detail actions:
  - `Print Ticket` or `Reprint Ticket` for paid orders.
  - `Cancel Order` for same-day paid orders.
  - `Start New Order` to clear the draft cart and return to an empty cashier screen.
- Cancelled orders are visible in Today's Orders with `Cancelled` status.
- Cancelled order detail does not allow another cancellation.
- Queue number display uses canonical format `Queue No. 001` in the UI and `QUEUE NO. 001` on printed tickets.
- Order ID is internal and must not be shown as `#001`; visible staff/customer references use queue number.

Wireframe:

```text
+--------------------------------------------------------------------------------+
| Coffee POS                                      [New Order] [Today's Orders]    |
+--------------------------------------------------------------------------------+
| Today's Orders                                  Search queue: [ 001 ]           |
+--------------------------------------------------------------------------------+
| Queue No. | Time     | Items | Payment | Total     | Status                    |
| 001       | 10:15    | 1     | Cash    | Rp18.000  | Paid                      |
| 002       | 10:19    | 2     | QRIS    | Rp43.000  | Paid                      |
| 003       | 10:22    | 1     | Cash    | Rp25.000  | Cancelled                 |
+--------------------------------------------------------------------------------+
| Select an order row to open details.                                            |
+--------------------------------------------------------------------------------+
```

Paid order detail:

```text
+------------------------------------------------+
| Queue No. 001                         Paid      |
| 2026-06-28 10:15 WIB                            |
|                                                |
| 1x Americano                         Rp18.000  |
|    Hot, Normal sugar                            |
|                                                |
| Total: Rp18.000                                |
| Payment: Cash                                  |
|                                                |
| [Reprint Ticket] [Cancel Order] [Start New]    |
+------------------------------------------------+
```

Cancel confirmation dialog:

```text
+------------------------------------------------+
| Cancel Queue No. 001?                          |
|                                                |
| This keeps the order for audit but removes it  |
| from sales totals.                             |
|                                                |
| [Back]                         [Cancel Order]  |
+------------------------------------------------+
```

### Queue Numbering

User Stories:

#### US-09: Reset Queue Numbers Daily

As a cashier, I get queue numbers that restart each business day so customers receive simple daily queue numbers.

Acceptance:

- First paid order of a new Asia/Jakarta business date receives queue number 1.
- Queue numbers increment by 1 during the same Asia/Jakarta business date.
- Queue numbers do not duplicate for the same Asia/Jakarta business date.

Requirements:

- Each paid order receives a queue number.
- Queue numbers reset every Asia/Jakarta business day.
- Queue numbers increment by 1 for each paid order on the same Asia/Jakarta business day.
- Queue numbers should be unique per Asia/Jakarta business date.
- Queue number generation must be safe against duplicate numbers if two orders are submitted close together.

Wireframe:

Queue numbering is not a separate screen in MVP. After payment confirmation, the generated queue number appears on Paid Order Detail and on the printable ticket.

### Ticket Printing

User Stories:

#### US-10: Print Queue-Number Ticket

As a cashier, I can print a receipt-printer friendly ticket so the barista can prepare the drink from paper.

Acceptance:

- Ticket includes queue number, timestamp, items, quantities, modifiers, notes, total, and payment method.
- Ticket fits 80mm receipt paper for the Epson TM-T82III target printer.
- Ticket does not include digital queue status.

#### US-11: Fulfill From Paper Ticket

As a barista, I can read the printed ticket and prepare the drinks without using the application.

Acceptance:

- Printed ticket clearly shows the queue number.
- Printed ticket clearly shows modifiers and notes.
- No barista screen is required for MVP.

Requirements:

- The cashier can print a ticket after the order is paid.
- The ticket is optimized for an Epson TM-T82III receipt printer using 80mm thermal paper.
- Browser print CSS targets 80mm paper with content width constrained to avoid clipped text.
- The ticket includes:
  - Shop name from `SHOP_NAME`, defaulting to `Coffee Shop`.
  - Queue number.
  - Asia/Jakarta timestamp.
  - Order items.
  - Quantity per item.
  - Temperature and sugar modifiers.
  - Order note if present.
  - Total.
  - Payment method.
- The ticket does not include a digital queue status.

Wireframe:

```text
+------------------------------+
|        Coffee Shop            |
|                              |
|        QUEUE NO. 001          |
|  2026-06-28 10:15 WIB         |
|                              |
|  1x Americano        18.000   |
|     Hot                      |
|     Normal sugar             |
|                              |
|  Note: less ice              |
|                              |
|  Total              18.000   |
|  Payment              CASH   |
|                              |
|  Please give this ticket     |
|  to the barista.             |
+------------------------------+
```

### Daily Summary

User Stories:

#### US-12: View Daily Summary

As an owner/operator, I can view daily totals so I can reconcile sales at the end of the day.

Acceptance:

- Summary shows total order count.
- Summary shows total item/cup count.
- Summary shows gross sales.
- Summary shows Cash total and QRIS total.
- Summary shows cancelled order count.

Requirements:

- The owner/operator can view a daily summary.
- The daily summary includes:
  - Total order count.
  - Total item/cup count.
  - Gross sales.
  - Cash total.
  - QRIS total.
  - Cancelled order count.
- Daily summary uses Asia/Jakarta business date boundaries.
- Daily summary excludes cancelled orders from order count, item/cup count, gross sales, Cash total, and QRIS total.

Wireframe:

```text
+----------------------------------------------------------------+
| Daily Summary                                      2026-06-28   |
+----------------------------------------------------------------+
| Orders                 | 42                                     |
| Items / cups           | 57                                     |
| Gross sales            | Rp1.125.000                            |
| Cancelled orders       | 1                                      |
+----------------------------------------------------------------+
| Payment Method         | Total                                  |
| Cash                   | Rp525.000                              |
| QRIS                   | Rp600.000                              |
+----------------------------------------------------------------+
```

## Non-Functional Requirements

- Cashier order entry for a common drink should take 15-20 seconds after the customer decides.
- The cashier screen must be usable on a laptop or tablet-sized browser.
- Text on buttons and tickets must remain readable on common cashier displays.
- The ticket print layout must fit Epson TM-T82III 80mm receipt paper without clipped text.
- The app should avoid unnecessary network dependencies for the checkout path.
- Monetary values must be stored as integer rupiah values, not floating-point amounts.
- The app should keep the workflow simple enough for new staff to learn in under 30 minutes.
- Authentication must not add noticeable friction after the cashier is signed in.
- Authenticated sessions must use secure cookie settings and must not rely on localStorage tokens.
- Daily boundaries must consistently use Asia/Jakarta, not the database server timezone.

## Code Style

Use explicit domain names and small pure functions for core business rules. Keep calculations and queue-number rules outside HTTP handlers and frontend components so they can be tested directly.

Backend example:

```go
package orders

type PaymentMethod string

const (
	PaymentMethodCash PaymentMethod = "cash"
	PaymentMethodQRIS PaymentMethod = "qris"
)

type Modifier struct {
	GroupName    string
	OptionName   string
	PriceDeltaRp int
}

type OrderLine struct {
	MenuItemID  string
	Name        string
	UnitPriceRp int
	Quantity    int
	Modifiers   []Modifier
}

func CalculateOrderTotalRp(lines []OrderLine) int {
	totalRp := 0

	for _, line := range lines {
		modifierTotalRp := 0
		for _, modifier := range line.Modifiers {
			modifierTotalRp += modifier.PriceDeltaRp
		}

		totalRp += (line.UnitPriceRp + modifierTotalRp) * line.Quantity
	}

	return totalRp
}
```

Frontend example:

```tsx
import { useMemo, useState } from "preact/hooks";

type PaymentMethod = "cash" | "qris";

type CartLine = {
  menuItemId: string;
  name: string;
  unitPriceRp: number;
  quantity: number;
  modifiers: Array<{
    groupName: string;
    optionName: string;
    priceDeltaRp: number;
  }>;
};

type CashierCartProps = {
  initialLines: CartLine[];
};

export function CashierCart({ initialLines }: CashierCartProps) {
  const [lines, setLines] = useState(initialLines);
  const [paymentMethod, setPaymentMethod] = useState<PaymentMethod>("cash");

  const totalRp = useMemo(() => calculateCartTotalRp(lines), [lines]);
  const canConfirmPayment = lines.length > 0 && lines.every(hasRequiredModifiers);

  return (
    <section aria-label="Current order">
      <output>{formatRupiah(totalRp)}</output>
      <button
        type="button"
        disabled={!canConfirmPayment}
        onClick={() => openConfirmPaymentDialog(lines, paymentMethod)}
      >
        Confirm Paid
      </button>
    </section>
  );
}
```

Conventions:

- Use Go for backend API code.
- Use TypeScript and Preact components for frontend code.
- Use production UI patterns from `docs/screen-captures/06-order-revamp.png` for the cashier order-entry screen: restrained green action color, high-contrast dark text, image-led product cards, compact controls, and a persistent current-order panel.
- Use real icons or simple accessible symbols for search, add, quantity, remove, grid/list view, payment method, lock/proceed, and modal close controls.
- Avoid visible instructional copy that explains obvious UI mechanics; use concise labels, placeholders, and accessible names.
- Keep cards at modest radius and avoid nested card-on-card layouts except for individual repeated items and modals.
- Use hexagonal architecture for the backend: domain and application packages define behavior and ports; adapters implement HTTP, PostgreSQL, sessions, hashing, rate limiting, clocks, and config.
- Keep dependency direction inward. Domain packages must not import application or adapter packages; application packages must not import adapter packages.
- Define repository, transaction, clock, queue-number, session, and rate-limit interfaces in the application layer at the point of use.
- Keep generated sqlc query types inside PostgreSQL adapters. Convert between sqlc rows and domain/application types at the adapter boundary.
- Keep HTTP request/response DTOs inside HTTP adapters. Convert between DTOs and application command/result types at the adapter boundary.
- Use standard Go naming conventions: short local names where obvious, explicit exported names where they cross package boundaries.
- Use `PascalCase` for exported Go identifiers and `camelCase` for unexported identifiers.
- Use `PascalCase` for Preact components and TypeScript types.
- Keep Preact components focused on UI state and API calls; keep calculation helpers in `frontend/src/lib/`.
- Use `Rp` suffix for integer rupiah fields, for example `totalRp` and `unitPriceRp`.
- Keep presentation formatting separate from stored values.
- Validate backend inputs even when the frontend controls the options.
- Keep HTTP handlers thin: parse request, call application use cases, choose response.
- Keep backend SQL in `backend/queries/` and migrations in `backend/migrations/`; do not hide business rules inside SQL unless the rule is inherently a database constraint.
- Prefer unit tests against domain services and application use cases using in-memory/fake port implementations before adding database-backed integration tests.

## Testing Strategy

### Unit Tests

Use `go -C backend test ./...` for:

- PIN format validation.
- PIN verification failure and success paths.
- Session expiration logic.
- Order total calculation.
- Multiple order lines for the same menu item with different modifiers.
- Required modifier validation.
- Cash/QRIS payment method validation.
- Asia/Jakarta daily queue number reset logic.
- Daily summary aggregation.
- Cancelled order exclusion from sales totals.
- Rupiah formatting.
- Application use cases with fake ports for login, create paid order, list today's orders, cancel order, and daily summary.
- Port contract behavior for queue-number allocation, session expiration, and repository error handling.

### Integration Tests

Use Go integration tests with Testcontainers for Go for database-backed cases:

- Login endpoint accepts a valid PIN and creates a session.
- Login endpoint rejects invalid PINs with a generic error.
- Protected POS endpoints reject unauthenticated requests.
- Logout invalidates the current session.
- Auth rate limiting blocks repeated failed login attempts.
- Backend seeder creates the expected menu, items, and modifiers.
- Creating a paid Cash order persists the order and assigns a queue number.
- Creating a paid QRIS order persists the order and assigns a queue number.
- Listing today's persisted orders returns paid and cancelled orders for the Asia/Jakarta business date.
- Opening paid order detail returns item lines, per-line modifiers, payment method, total, status, and queue number.
- Reprinting a paid order does not create a new order or queue number.
- Unpaid drafts are not persisted.
- Cancelling a same-day paid order marks it cancelled without deleting it.
- Daily summary groups totals by Cash and QRIS.
- Daily summary excludes cancelled orders from financial totals and item/cup counts.
- HTTP handlers validate invalid requests and return useful errors.

Testcontainers requirements:

- Testcontainers-backed tests require a working Podman runtime or compatible Docker API socket.
- PostgreSQL integration tests should use the Testcontainers PostgreSQL module.
- Testcontainers-backed tests should be gated behind an `integration` build tag so fast unit tests can run without containers.
- Testcontainers must not be required inside production containers.

### Frontend Checks

Use frontend checks for:

- TypeScript type safety.
- Production frontend build.
- PIN entry form validation.
- Authenticated route guard behavior.
- Static QRIS image path `/qris/static-qris.png` is used when QRIS payment is selected.
- Static menu and avatar assets are included in the production frontend container build and served with image content types.
- Menu search filters visible catalog cards without clearing the current order.
- Category tabs and quick filters update the visible catalog from seeded item metadata.
- Menu cards render image, name, price, optional badge, and accessible add control.
- Per-line modifier selection and cart rendering.
- Current-order panel renders thumbnails, quantity badge, modifier summary, line total, quantity stepper, and remove action for each line.
- Order note enforces the 120-character limit and renders a character count.
- Subtotal, optional tax/service charge, and total rows render consistently with integer rupiah formatting.
- Disabled state for `Confirm Paid` on empty or invalid carts.
- Disabled or unavailable `Print Ticket` state before payment.
- Confirmation dialog or modal behavior for `Proceed to Payment`, `Confirm Paid`, `Cancel`, close, and `Cancel Order`.
- Today's Orders list filtering by queue number.
- API client request and response shapes.
- Rupiah formatting used by the cashier UI.
- Ticket preview rendering helpers.

### End-to-End Tests

Use Playwright in local development or CI for the few workflows that require a real browser:

- Cashier cannot access the cashier screen before PIN login.
- Cashier signs in with a valid 6-digit PIN and reaches the cashier screen.
- Cashier can search and filter the visual menu catalog while preserving the draft order.
- Cashier creates one Americano Hot and one Americano Iced in the same order, and both lines keep their own modifiers.
- Cashier cannot print an unpaid draft order.
- Cashier opens the payment modal, cancels it, and returns to the unchanged current order.
- Cashier confirms payment through the confirmation dialog or modal and lands on Paid Order Detail.
- Cashier creates an Americano order, marks Cash paid, and reaches printable ticket view.
- Cashier creates a Latte order, selects QRIS, sees static QRIS image, marks paid, and reaches printable ticket view.
- Cashier finds a paid order from Today's Orders by queue number and reprints its ticket without changing the queue number.
- Cashier cancels a same-day paid order and daily summary excludes it from sales totals.
- Daily summary reflects orders created during the test day.
- Ticket print view renders at the Epson TM-T82III / 80mm target width without clipped essential fields.

Coverage target:

- Core business rules should have unit coverage.
- Frontend must pass type/build checks before deployment.
- Main cashier workflow should have at least one end-to-end test before MVP is considered complete.
- Browser test tooling must not be installed or executed inside production containers.

## Boundaries

- Always:
  - Keep MVP scope limited to cashier order entry, static QRIS display, printed queue-number ticket, seeded menu data, and daily summary.
  - Treat `docs/screen-captures/06-order-revamp.png` as the target for cashier order-entry layout, hierarchy, and interaction flow unless a newer approved design replaces it.
  - Require an authenticated session for cashier and reporting workflows.
  - Keep PIN verification on the backend; never expose the PIN or PIN hash to the frontend.
  - Store production PIN material and session secrets outside source control.
  - Use `HttpOnly`, `Secure` in production, and `SameSite=Lax` or stricter session cookies.
  - Rate-limit PIN login attempts.
  - Use Asia/Jakarta for queue resets, tickets, and daily summary date boundaries.
  - Persist only paid orders; keep unpaid drafts in frontend state.
  - Keep current-order state intact when search, category filters, quick filters, sort controls, payment modal cancel, or modal close actions occur.
  - Cancel accidental paid orders instead of deleting them.
  - Exclude cancelled orders from sales totals and item/cup counts.
  - Use hexagonal architecture for backend implementation boundaries.
  - Keep domain and application packages independent from HTTP, PostgreSQL, sqlc, cookies, and environment/config adapters.
  - Keep production split into a static Preact frontend container and a Go backend API container unless this spec is updated.
  - Use Podman for local production-image builds unless the deployment platform provides its own OCI image builder.
  - Use Testcontainers for Go for PostgreSQL-backed integration tests instead of hand-maintained shared test databases.
  - Keep frontend production runtime static-only; do not run Node.js in the frontend production container.
  - Keep production containers free of dev-only build tools, Testcontainers, and browser test tooling.
  - Keep the database connection pool small for the 512 MB database instance.
  - Store money as integer rupiah values.
  - Store any tax/service charge amount as integer rupiah and keep the configured rate auditable.
  - Validate required modifiers and payment method before creating a paid order.
  - Run relevant tests before marking implementation tasks complete.
  - Update this spec when scope or stack decisions change.

- Ask first:
  - Changing the selected app stack.
  - Adding a persistent Node.js production server, replacing Preact with another frontend framework, adding Next.js, adding Prisma, or adding a large UI framework.
  - Changing the authentication mechanism, adding user accounts, adding roles, or adding PIN management UI.
  - Replacing Podman as the local container runtime or replacing Testcontainers as the backend integration test harness.
  - Adding external payment integration.
  - Enabling, changing, or hard-coding tax/service charge policy or rate.
  - Adding a digital barista queue or customer queue display.
  - Adding menu management UI.
  - Adding refunds, partial cancellations, or editing paid orders after printing.
  - Adding inventory, loyalty, online ordering, or multi-branch support.
  - Changing receipt-printer assumptions or requiring direct printer drivers.

- Never:
  - Commit QRIS production credentials, PINs, session secrets, or private payment keys.
  - Store monetary amounts as floating-point values.
  - Hard-code tax or service charge behavior based only on a design mock without business approval.
  - Store auth tokens in localStorage or other client-accessible browser storage.
  - Embed the cashier PIN or PIN hash in frontend code, frontend config, static files, logs, or API responses.
  - Delete paid orders as the correction mechanism.
  - Run Playwright, Testcontainers, Podman, Vite dev server, frontend file watchers, or code generators in production containers.
  - Add unsupported payment methods to MVP without updating the spec.
  - Require the barista to use the app for MVP.
  - Add dine-in, takeaway, or delivery labels to MVP without approval.
  - Remove failing tests instead of fixing the behavior or updating the spec.

## Success Criteria

- Unauthenticated users must sign in with the predefined 6-digit cashier PIN before accessing POS workflows.
- Valid PIN creates a secure session; invalid PINs fail with a generic error and repeated failures are rate-limited.
- Cashier can create a paid Americano or Latte order with required modifiers after signing in, and the UI can present the broader approved seeded catalog.
- Cashier sees the order revamp layout on desktop/tablet: visual menu catalog, search, category tabs, quick filters, sort control, grid/list affordance, and persistent current-order panel.
- Cashier can add seeded catalog items from image-led product cards.
- Menu search, category tabs, quick filters, sort, and pagination do not clear or mutate the current order.
- Cashier can create multiple lines for the same drink with different per-line modifiers in one order.
- Cashier cannot confirm payment for an empty or invalid cart.
- Cashier cannot print a ticket before payment confirmation.
- Cashier must confirm payment in a dialog or modal before the paid order is persisted.
- Cashier can select Cash or QRIS only.
- QRIS payment flow displays `/qris/static-qris.png`, total amount, manual scan instructions, confirm, and cancel controls.
- Cancelling or closing the payment modal leaves the current order unchanged.
- Each paid order receives a queue number.
- Queue numbers reset each Asia/Jakarta business day and do not duplicate within the same day.
- After payment confirmation, the cashier lands on the Paid Order Detail screen for the new queue number.
- Cashier can find today's paid orders by queue number from Today's Orders.
- Cashier can reprint an existing paid order ticket without creating a new order or queue number.
- Cashier can cancel accidental same-day paid orders without deleting audit history.
- Cashier must confirm cancellation in a dialog before the order status changes.
- Cancelled orders are excluded from sales totals.
- Cashier can print an Epson TM-T82III / 80mm receipt-printer friendly ticket with queue number, Asia/Jakarta timestamp, items, modifiers, notes, total, and payment method.
- Barista can fulfill the order from the printed ticket without using the app.
- Daily summary shows order count, item/cup count, gross sales, Cash total, QRIS total, and cancelled order count.
- No user account management, PIN management UI, menu management screen, digital queue module, payment gateway integration, or service type labels are present in MVP.

## Open Questions

- Should the design's `Tax (11%)` row be implemented for MVP, and if so is it tax, service charge, or another shop-specific charge?
- Should the expanded design catalog items beyond Americano and Latte be seeded in the next backend slice, or should some remain visual placeholders until menu data is finalized?
- Should a visible draft reference such as `#ORD-0142` appear in the current-order panel, or should the MVP avoid draft identifiers and show only post-payment queue numbers?
- Is list view required for MVP, or is the grid/list toggle visual-only until a larger menu makes list view useful?
- Should catalog pagination be implemented immediately, or should the MVP use scrolling until the menu size requires pagination?

## Resolved Implementation Decisions

- Receipt printer: Epson TM-T82III, targeting 80mm thermal paper.
- Printed shop name: default to `Coffee Shop`, configurable via `SHOP_NAME`.
- Ticket timestamps: always display Asia/Jakarta local time with `WIB` label.
- Queue reset and daily summary boundaries: Asia/Jakarta business date.
- Queue display format: `Queue No. 001` in app UI and `QUEUE NO. 001` on printed tickets.
- Post-payment destination: Paid Order Detail is the single canonical post-payment screen.
- Modifier behavior: modifiers belong to individual order lines; same item with different modifiers creates separate lines.
- Backend architecture: hexagonal architecture with domain, application/use-case, and adapter packages.
- Container runtime: Podman for local image builds and local container runs.
- Backend integration tests: Testcontainers for Go with PostgreSQL module, using Podman as the local compatible container runtime.
- Session lifetime: expires at the end of the Asia/Jakarta business day or after 12 hours, whichever comes first.
- PIN rotation: manual for MVP through backend environment configuration and service restart/redeploy.
- Order persistence: unpaid drafts remain frontend-only; backend persists paid orders only.
- Order correction: same-day accidental paid orders are cancelled, never deleted.
- Cancelled orders: kept for audit and excluded from sales totals and item/cup counts.
- QRIS image: owner-provided static asset at `frontend/public/qris/static-qris.png`, served as `/qris/static-qris.png`.
- Cashier order-entry design reference: `docs/screen-captures/06-order-revamp.png`.

## Review Gate

This spec is Phase 1 of the spec-driven workflow. Implementation should not start until the assumptions, tech stack, and resolved implementation decisions are reviewed or accepted.
