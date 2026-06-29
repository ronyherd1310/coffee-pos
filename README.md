# Coffee POS

Foundation scaffold for a small coffee shop POS MVP. This repository currently proves the frontend,
backend, test, and container wiring only; POS application logic is intentionally not implemented yet.

## Local Development

Install frontend dependencies:

```sh
npm --prefix frontend install
```

Run the backend API:

```sh
go -C backend run ./cmd/coffee-pos serve
```

Run the Vite frontend in another terminal:

```sh
npm --prefix frontend run dev
```

The frontend calls `/api/health` relative to its own origin. Vite proxies `/api` to the backend on
`http://localhost:8080`.

## Tests And Checks

```sh
go -C backend test ./...
go -C backend test -tags=integration ./...
go -C backend vet ./...
npm --prefix frontend test
npm --prefix frontend run check
npm --prefix frontend run build
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
podman compose up --build
```

The browser-facing service is the frontend/Caddy container on `http://localhost:8080`. Caddy proxies
`/api/health` to the backend service on the Compose network. PostgreSQL is included for future
database-backed work, but the scaffold health endpoint does not depend on database connectivity.
