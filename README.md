# Coffee POS

Foundation scaffold for a small coffee shop POS MVP. The backend currently includes cashier PIN
authentication, database-backed menu reads, paid cashier order creation, and same-day cancellation.

## Local Development

Install frontend dependencies:

```sh
npm --prefix frontend install
```

Apply database migrations and seed the menu before using cashier order entry:

```sh
export DATABASE_URL="postgres://coffee_pos:coffee_pos_dev@localhost:5432/coffee_pos?sslmode=disable"
go -C backend run ./cmd/coffee-pos db migrate
go -C backend run ./cmd/coffee-pos db seed
```

Run the backend API:

```sh
CASHIER_PIN_HASH="$(go -C backend run ./cmd/coffee-pos auth hash-pin 123456)"
export CASHIER_PIN_HASH
export DATABASE_URL="postgres://coffee_pos:coffee_pos_dev@localhost:5432/coffee_pos?sslmode=disable"
go -C backend run ./cmd/coffee-pos serve
```

Run the Vite frontend in another terminal:

```sh
npm --prefix frontend run dev
```

The frontend calls `/api/health` relative to its own origin. Vite proxies `/api` to the backend on
`http://localhost:8080`.

## Backend Configuration

Required for `serve`:

- `CASHIER_PIN_HASH`: bcrypt hash of the 6-digit cashier PIN. Generate a local development value with `go -C backend run ./cmd/coffee-pos auth hash-pin <6-digit-pin>`.

Required for `serve` and database commands:

- `DATABASE_URL`: PostgreSQL connection string, for example `postgres://coffee_pos:coffee_pos_dev@localhost:5432/coffee_pos?sslmode=disable`.

Do not commit real PINs, PIN hashes, database passwords, or local `.env` files.

## Database Migrations And Seeding

Apply migrations explicitly, then run the idempotent initial menu seeder:

```sh
DATABASE_URL="postgres://coffee_pos:coffee_pos_dev@localhost:5432/coffee_pos?sslmode=disable" \
  go -C backend run ./cmd/coffee-pos db migrate

DATABASE_URL="postgres://coffee_pos:coffee_pos_dev@localhost:5432/coffee_pos?sslmode=disable" \
  go -C backend run ./cmd/coffee-pos db seed
```

The seeder creates the Coffee category, Americano at Rp18.000, Latte at Rp25.000, required
Temperature options Hot and Iced, and required Sugar options Normal, Less sugar, and No sugar.
Rerunning `db seed` converges the same rows without duplicates.

## Backend Cashier APIs

Protected cashier endpoints require the HttpOnly session cookie from `POST /api/auth/login`:

- `GET /api/pos/menu`: returns active menu categories, items, required modifier groups, options, stable slugs, and rupiah prices.
- `POST /api/pos/orders`: persists a paid Cash or QRIS order after payment confirmation. The backend accepts `clientRequestId`, `paymentMethod`, optional `note`, and cart lines by menu/modifier slugs. It recalculates all prices and totals server-side and stores only paid orders.
- `POST /api/pos/orders/{orderId}/cancel`: marks a same-day paid order as `cancelled` without deleting it.

Create-order requests require a canonical lowercase UUID `clientRequestId` idempotency key. Retrying
the same request with the same key returns the original `PaidOrderDetail`; reusing the key with a
different request returns a conflict. `orderId` is exposed as a string even though the database ID is
numeric. Create and cancel responses share the same `PaidOrderDetail` shape.

Regenerate sqlc wrappers after changing SQL queries or migrations:

```sh
go -C backend run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.29.0 generate
```

## Tests And Checks

```sh
go -C backend test ./...
go -C backend test -tags=integration ./...
go -C backend vet ./...
npm --prefix frontend test
npm --prefix frontend run check
npm --prefix frontend run build
```

The integration test command uses Testcontainers with PostgreSQL and requires Podman or a compatible
Docker API socket. With rootless Podman, start or expose the user socket first if your environment
does not already do it. In one terminal:

```sh
mkdir -p "${XDG_RUNTIME_DIR}/podman"
podman system service --time=0 "unix://${XDG_RUNTIME_DIR}/podman/podman.sock"
```

Then run:

```sh
DOCKER_HOST="unix://${XDG_RUNTIME_DIR}/podman/podman.sock" \
TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE="${XDG_RUNTIME_DIR}/podman/podman.sock" \
  go -C backend test -tags=integration ./...
```

Run the browser smoke test after the local app or Compose stack is serving `http://localhost:8080`:

```sh
npm install
npx playwright test
```

## Containers

Build images:

```sh
podman build -f backend/Containerfile -t coffee-pos-backend:dev backend
podman build -f frontend/Containerfile -t coffee-pos-frontend:dev frontend
```

Start the production-style local stack:

```sh
CASHIER_PIN_HASH="$(go -C backend run ./cmd/coffee-pos auth hash-pin 123456)"
export CASHIER_PIN_HASH
podman compose up --build
```

The browser-facing service is the frontend/Caddy container on `http://localhost:8080`. Caddy proxies
`/api/health` to the backend service on the Compose network. PostgreSQL is configured for backend
database commands with local-only development credentials.

The spec currently recommends Go 1.26.x, while this repository still uses Go 1.25.0 in
`backend/go.mod` and `backend/Containerfile`; resolve that intentionally when Go tooling is updated.
