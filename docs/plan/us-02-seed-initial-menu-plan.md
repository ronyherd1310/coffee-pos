# Implementation Plan: US-02 Seed Initial Menu

**Status:** Ready for code review

## Overview

Implement `US-02: Seed Initial Menu` from `docs/specs/small-coffee-shop-pos-mvp-spec.md`. This plan covers the first database-backed menu slice: database migrations, sqlc query setup, menu domain/application rules, PostgreSQL-backed idempotent seeding, and a backend CLI command that creates the approved MVP menu data. No menu management UI or cashier order-entry UI is included in this slice.

## Scope

In scope:

- Database schema for menu categories, menu items, modifier groups, and modifier options.
- Backend migration command foundation needed to create the menu schema.
- sqlc configuration and generated query wrappers for menu seeding and menu reads needed to verify seeded data.
- Menu domain/application types for the approved seeded menu and required modifier rules.
- PostgreSQL adapter implementation for idempotent menu seeding.
- Backend CLI command `coffee-pos db seed`.
- Integration tests proving the seeder creates the expected menu data and can be run repeatedly without duplicates.
- Documentation updates for commands or runtime configuration that intentionally change.

Out of scope:

- Menu management screens or CRUD APIs.
- Cashier order-entry UI and frontend menu rendering.
- Order creation, queue numbers, payment confirmation, ticket printing, and reports.
- Branch-specific or tenant-specific menus.
- Modifier price deltas other than Rp0.
- Runtime editing, archiving, or reordering of seeded menu data.

## Architecture Decisions

- Keep menu invariants in `internal/domain/menu`; do not embed seeded menu business rules in HTTP, CLI, SQL, or generated sqlc types.
- Define seeding use cases and ports in `internal/app/menu`, with PostgreSQL-specific implementation under `internal/adapters/postgres`.
- Use explicit SQL migrations under `backend/migrations/` and SQL query files under `backend/queries/`, matching the spec's sqlc direction.
- Make seed data idempotent with stable natural keys or slugs plus database uniqueness constraints. The same seeder can safely run multiple times without creating duplicate categories, items, groups, or options.
- Store prices as integer rupiah values using `price_rp` and `price_delta_rp` columns.
- Model `Temperature` and `Sugar` as required single-select modifier groups. Attach both groups to each MVP menu item unless the implementation records groups globally and exposes equivalent item applicability for every seeded item.
- Add database configuration only as needed for this slice, likely `DATABASE_URL`, while keeping existing auth configuration intact.
- Keep Testcontainers-backed database tests behind the `integration` build tag so normal `go -C backend test ./...` stays fast and container-free.

## Dependency Graph

```text
Database config and connection pool settings
  -> migration runner and menu schema
      -> sqlc config and menu seed/read queries
          -> menu domain seed definition and validation
              -> app/menu seed use case and repository ports
                  -> adapters/postgres menu repository
                      -> CLI db seed wiring
                          -> integration tests for migration + idempotent seeding
```

## Task List

### Phase 1: Database Foundation

## Task 1: Add Database Configuration

**Description:** Extend backend configuration so database-backed commands can connect to PostgreSQL without affecting the existing auth-only server path more than necessary. Keep connection pool defaults conservative for the small deployment target.

**Acceptance criteria:**

- [ ] Backend config supports a PostgreSQL connection string such as `DATABASE_URL`.
- [ ] Missing database config fails clearly for database commands that require it.
- [ ] Existing auth tests and server startup configuration behavior remain compatible.
- [ ] Database pool settings default to low resource usage, aligned with the spec's starting point of 3 open and 1 idle connection.

**Verification:**

- [ ] Tests pass: `go -C backend test ./...`
- [ ] Static checks pass: `go -C backend vet ./...`
- [ ] Manual check: `go -C backend run ./cmd/coffee-pos db seed` without database config returns a clear configuration error after the command exists.

**Dependencies:** None

**Files likely touched:**

- `backend/internal/config/config.go`
- `backend/internal/config/config_test.go`
- `backend/go.mod`

**Estimated scope:** Medium: 3 files

## Task 2: Add Migration Runner And Initial Menu Schema

**Description:** Add a minimal migration system and the first SQL migration for menu data. The schema should support seeded categories, items, required single-select modifier groups, item-to-group applicability, and option price deltas while enforcing uniqueness needed for idempotency.

**Acceptance criteria:**

- [ ] `backend/migrations/` contains an initial menu migration.
- [ ] The schema stores the Coffee category, Americano and Latte items, Temperature and Sugar groups, and their options without duplicate rows.
- [ ] Database constraints prevent duplicate category names, duplicate item identity within a category, duplicate modifier group names, duplicate options within a group, and duplicate item/group links.
- [ ] Required single-select modifier groups can be represented explicitly.
- [ ] Integer rupiah prices are used for item prices and option price deltas.
- [ ] A backend migration command such as `coffee-pos db migrate` can apply migrations once and then no-op cleanly when rerun.

**Verification:**

- [ ] Tests pass: `go -C backend test ./...`
- [ ] Integration tests pass: `go -C backend test -tags=integration ./...`
- [ ] Manual check: running the migration command twice against a local PostgreSQL database does not fail or duplicate schema state.

**Dependencies:** Task 1

**Files likely touched:**

- `backend/cmd/coffee-pos/main.go`
- `backend/internal/adapters/postgres/db.go`
- `backend/internal/adapters/postgres/migrations.go`
- `backend/internal/adapters/postgres/migrations_test.go`
- `backend/migrations/000001_create_menu_schema.sql`

**Estimated scope:** Medium: 5 files

### Checkpoint: Database Foundation

- [ ] Database config is isolated from non-database commands.
- [ ] Menu schema can be created from scratch.
- [ ] Migration command is idempotent.
- [ ] `go -C backend test ./...`, `go -C backend vet ./...`, and relevant integration tests pass.

### Phase 2: Menu Contracts

## Task 3: Add sqlc Configuration And Menu Queries

**Description:** Introduce sqlc for generated query wrappers and add the minimal menu queries needed by seeding and verification. Keep generated types inside the PostgreSQL adapter boundary.

**Acceptance criteria:**

- [ ] `backend/sqlc.yaml` configures Go generation for menu queries.
- [ ] `backend/queries/` contains upsert/read queries for categories, items, modifier groups, item/group links, and modifier options.
- [ ] Generated sqlc code is placed under an adapter-owned package and does not leak into domain or application packages.
- [ ] Query names and parameters use rupiah suffixes for money fields where applicable.
- [ ] The generated code can be refreshed with a documented command.

**Verification:**

- [ ] sqlc generation succeeds with the chosen local command.
- [ ] Tests pass: `go -C backend test ./...`
- [ ] Static checks pass: `go -C backend vet ./...`

**Dependencies:** Task 2

**Files likely touched:**

- `backend/sqlc.yaml`
- `backend/queries/menu.sql`
- `backend/internal/adapters/postgres/sqlc/*`
- `backend/go.mod`
- `README.md`

**Estimated scope:** Medium: 5 files

## Task 4: Implement Menu Domain Seed Definition

**Description:** Add pure menu domain types and validation for the approved MVP menu seed data. This gives the seeder a single source of truth for the Coffee category, two items, two required modifier groups, and five Rp0 modifier options.

**Acceptance criteria:**

- [ ] Domain code defines the Coffee category.
- [ ] Domain code defines Americano at `18000` rupiah and Latte at `25000` rupiah.
- [ ] Domain code defines required single-select Temperature options: Hot and Iced.
- [ ] Domain code defines required single-select Sugar options: Normal, Less sugar, and No sugar.
- [ ] Domain validation rejects empty names, negative prices, missing required options, and duplicate names within a logical group.
- [ ] Domain package does not import `database/sql`, sqlc packages, HTTP packages, config packages, or CLI packages.

**Verification:**

- [ ] Tests pass: `go -C backend test ./...`
- [ ] Targeted unit tests cover the approved seed definition and invalid seed definitions.

**Dependencies:** None

**Files likely touched:**

- `backend/internal/domain/menu/menu.go`
- `backend/internal/domain/menu/seed.go`
- `backend/internal/domain/menu/seed_test.go`

**Estimated scope:** Medium: 3 files

## Task 5: Add Menu Seeding Use Case

**Description:** Add the application-layer use case that validates the seed definition and asks a repository port to persist it. This task establishes the adapter contract without depending on PostgreSQL directly.

**Acceptance criteria:**

- [ ] `internal/app/menu` exposes a seed use case that accepts or constructs the approved seed definition.
- [ ] The use case validates seed data before repository writes.
- [ ] Repository and transaction boundaries are expressed as application-layer ports.
- [ ] The use case returns clear application errors for validation and persistence failures.
- [ ] Unit tests use fake ports to verify successful seeding, validation failure, and repository failure behavior.

**Verification:**

- [ ] Tests pass: `go -C backend test ./...`
- [ ] Static checks pass: `go -C backend vet ./...`

**Dependencies:** Task 4

**Files likely touched:**

- `backend/internal/app/menu/ports.go`
- `backend/internal/app/menu/usecases.go`
- `backend/internal/app/menu/usecases_test.go`

**Estimated scope:** Medium: 3 files

### Checkpoint: Menu Contracts

- [ ] Menu domain and application packages compile without adapter imports.
- [ ] Approved seed data is represented once in backend code.
- [ ] sqlc-generated database types remain inside PostgreSQL adapters.
- [ ] `go -C backend test ./...` and `go -C backend vet ./...` pass.

### Phase 3: Seeder Implementation

## Task 6: Implement PostgreSQL Menu Seeder Repository

**Description:** Implement the `internal/app/menu` repository port with PostgreSQL and sqlc. The repository should persist the whole seed definition transactionally and use upsert-style behavior so repeated runs converge on one copy of the approved menu.

**Acceptance criteria:**

- [ ] The repository creates or updates the Coffee category.
- [ ] The repository creates or updates Americano at `18000` rupiah and Latte at `25000` rupiah.
- [ ] The repository creates or updates required single-select Temperature and Sugar modifier groups.
- [ ] The repository creates or updates Hot, Iced, Normal, Less sugar, and No sugar options with `0` rupiah deltas.
- [ ] The repository links both modifier groups to both menu items or otherwise represents equivalent applicability for both items.
- [ ] All seed writes happen inside one transaction.
- [ ] Repeated repository calls do not create duplicate rows.

**Verification:**

- [ ] Tests pass: `go -C backend test ./...`
- [ ] Integration tests pass: `go -C backend test -tags=integration ./...`
- [ ] Targeted integration test runs the repository seeder twice and asserts exact counts and values.

**Dependencies:** Tasks 3 and 5

**Files likely touched:**

- `backend/internal/adapters/postgres/menu_repository.go`
- `backend/internal/adapters/postgres/menu_repository_integration_test.go`
- `backend/internal/adapters/postgres/db.go`
- `backend/queries/menu.sql`

**Estimated scope:** Medium: 4 files

## Task 7: Wire `coffee-pos db seed`

**Description:** Add CLI wiring for the backend seeder command using the existing command style in `backend/cmd/coffee-pos/main.go`. The command should load database config, apply required wiring, run the menu seeding use case, and report a concise success or failure result.

**Acceptance criteria:**

- [ ] `go -C backend run ./cmd/coffee-pos db seed` runs the menu seeder.
- [ ] Existing commands `serve` and `auth hash-pin <pin>` keep their current behavior.
- [ ] Command usage output includes the database command without becoming misleading.
- [ ] Seeder failures return a non-zero exit code and a clear stderr message.
- [ ] Successful seeding does not print secrets or database connection strings.

**Verification:**

- [ ] Tests pass: `go -C backend test ./...`
- [ ] Static checks pass: `go -C backend vet ./...`
- [ ] Manual check: run `db seed` twice against a migrated local PostgreSQL database and confirm both runs succeed.

**Dependencies:** Task 6

**Files likely touched:**

- `backend/cmd/coffee-pos/main.go`
- `backend/cmd/coffee-pos/main_test.go`
- `backend/internal/seed/menu.go`

**Estimated scope:** Medium: 3 files

## Task 8: Add End-To-End Seeder Integration Coverage

**Description:** Add integration tests that exercise the same path a developer/operator uses: migrate an empty PostgreSQL database, run the seeder, run it again, and read back the exact category, items, groups, and options.

**Acceptance criteria:**

- [ ] Testcontainers-backed integration test starts a PostgreSQL database behind the `integration` build tag.
- [ ] The test applies migrations to an empty database.
- [ ] The test runs the seeder twice through the application or CLI-equivalent wiring.
- [ ] The test asserts exactly one Coffee category.
- [ ] The test asserts exactly Americano at `18000` rupiah and Latte at `25000` rupiah.
- [ ] The test asserts required single-select Temperature options Hot and Iced with `0` deltas.
- [ ] The test asserts required single-select Sugar options Normal, Less sugar, and No sugar with `0` deltas.
- [ ] The test asserts there are no duplicate seeded rows after the second run.

**Verification:**

- [ ] Integration tests pass: `go -C backend test -tags=integration ./...`
- [ ] Fast tests still pass without containers: `go -C backend test ./...`

**Dependencies:** Task 7

**Files likely touched:**

- `backend/internal/seed/menu_integration_test.go`
- `backend/internal/adapters/postgres/test_helpers_integration_test.go`
- `backend/cmd/coffee-pos/main_test.go`

**Estimated scope:** Medium: 3 files

### Checkpoint: Seeder Complete

- [ ] `coffee-pos db migrate` and `coffee-pos db seed` work against a local PostgreSQL database.
- [ ] Seeder can be run multiple times without duplicate menu data.
- [ ] `go -C backend test ./...` passes.
- [ ] `go -C backend test -tags=integration ./...` passes when Podman/Testcontainers is available.
- [ ] `go -C backend vet ./...` passes.

### Phase 4: Documentation And Local Runtime

## Task 9: Update Runtime Documentation And Compose Wiring

**Description:** Update developer documentation and local runtime wiring so database-backed seeding is discoverable. Keep secrets and local `.env` files out of source control.

**Acceptance criteria:**

- [ ] README documents `DATABASE_URL`, `db migrate`, and `db seed` usage.
- [ ] Compose backend service receives database connection configuration for the local PostgreSQL service.
- [ ] Documentation still notes `CASHIER_PIN_HASH` is required and must not be committed.
- [ ] Documentation describes integration tests requiring Podman or a compatible Docker API socket.
- [ ] Any intentional mismatch with the spec, such as Go version, is surfaced rather than copied silently.

**Verification:**

- [ ] Manual check: README commands are accurate for the implemented command names.
- [ ] `podman compose config` succeeds when required environment values are provided.
- [ ] Tests pass: `go -C backend test ./...`

**Dependencies:** Task 7

**Files likely touched:**

- `README.md`
- `compose.yaml`
- `docs/specs/small-coffee-shop-pos-mvp-spec.md` if behavior or commands intentionally differ from the spec
- `AGENTS.md` only if repository guidance changes

**Estimated scope:** Small: 2 files

## Task 10: Run Podman Compose Runtime Smoke

**Description:** Run the full local production-style stack with Podman Compose after US-02 is implemented. This verifies that the backend, frontend, and PostgreSQL services build, start, connect, and serve without runtime errors after the database-backed menu seeding slice is added.

**Acceptance criteria:**

- [ ] A local development `CASHIER_PIN_HASH` is generated for the smoke run and is not committed.
- [ ] `podman compose up --build` or `podman-compose up --build` starts PostgreSQL, backend, and frontend services successfully.
- [ ] Backend service reaches a healthy state and `/api/health` responds through the frontend/Caddy origin.
- [ ] PostgreSQL service reaches a healthy state and the backend can connect with the configured Compose database URL.
- [ ] Runtime logs show no repeated startup failures, migration/seed configuration errors, panics, or unhealthy service loops.
- [ ] The stack shuts down cleanly after verification.

**Verification:**

- [ ] Generate a local test PIN hash: `go -C backend run ./cmd/coffee-pos auth hash-pin 123456`
- [ ] Start stack with the generated hash in the shell environment: `CASHIER_PIN_HASH=<generated-hash> podman compose up --build`
- [ ] Confirm health through the browser-facing service: `curl -f http://localhost:8080/api/health`
- [ ] Inspect service status/logs: `podman compose ps` and `podman compose logs --tail=100`
- [ ] Stop the stack cleanly: `podman compose down`

**Dependencies:** Task 9

**Files likely touched:**

- None expected unless the smoke run exposes a runtime bug in `compose.yaml`, container config, or backend startup wiring.

**Estimated scope:** Small: runtime verification only

### Checkpoint: Ready For Review

- [ ] Every US-02 acceptance criterion is covered by automated tests or a documented manual check.
- [ ] No menu management UI was added.
- [ ] No frontend code was changed unless needed only for documentation or build config.
- [ ] Database-backed tests remain isolated behind the `integration` build tag.
- [ ] Changed commands and environment variables are documented.
- [ ] Podman Compose runtime smoke passes without service errors.

## Parallelization Opportunities

- Task 4 can be implemented in parallel with Tasks 1 and 2 because pure domain seed validation does not depend on the database.
- Task 5 can start after Task 4 while Task 3 is still being refined, as long as the repository port contract is stable.
- Task 9 documentation can start after command names and environment variables are agreed, but should be finalized after Task 7.
- Task 10 should run after Task 9 because it validates the final Compose/runtime wiring.
- Tasks 2, 3, 6, 7, and 8 should stay sequential because they share schema, query, adapter, and command contracts.

## Risks And Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Schema choices for modifiers conflict with later order-entry needs | High | Include item-to-group applicability and required single-select fields now, matching US-03 modifier validation needs. |
| Seeder idempotency relies only on application logic | High | Enforce uniqueness in the database and use transactional upsert behavior. |
| sqlc setup expands the slice too much | Medium | Add only queries needed for seed and verification; avoid broad menu CRUD. |
| Integration tests become required for normal development | Medium | Gate Testcontainers tests behind `integration` build tags and keep unit tests fast. |
| Database config breaks current auth-only local startup | Medium | Load/require database config only for database commands or database-backed server features. |
| Compose database URL contains local-only credentials | Low | Use only development credentials already present in Compose and do not commit real secrets or `.env` files. |
| Podman Compose is unavailable or uses the legacy `podman-compose` command locally | Low | Document both command forms and treat unavailable local container tooling as an environment blocker, not an application failure. |

## Resolved Decisions

- `db seed` will not automatically apply pending migrations. Developers/operators should run `db migrate` explicitly first, matching the separate commands listed in the spec. If the schema is missing or outdated, `db seed` should fail with a clear migration-related error.
- The seeder will converge existing seeded rows to the approved MVP seed data. If Americano, Latte, modifier groups, or modifier options already exist under the seeded identities with different approved values, the seeder should update them instead of treating that mismatch as fatal.
- Modifier groups will be modeled as global reusable groups with explicit item/group applicability links. For US-02, both Temperature and Sugar are linked to both Americano and Latte, preserving a straightforward path for US-03 order-entry validation.

## Planning Verification

- [x] Every task has acceptance criteria.
- [x] Every task has a verification step.
- [x] Task dependencies are identified and ordered.
- [x] No task is expected to touch more than about 5 files.
- [x] Checkpoints exist between major phases.
- [x] Human has resolved the initial planning questions.
- [ ] Human has reviewed and approved the full plan.
