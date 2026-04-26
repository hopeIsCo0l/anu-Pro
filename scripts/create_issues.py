"""Create all backlog issues on GitHub. Run once."""
import subprocess, json, sys, time

REPO = "hopeIsCo0l/anu-Pro"

# milestone numbers from GitHub (1-indexed in order created)
M = {
    "M0": "M0 — Foundations",
    "M1": "M1 — Inventory",
    "M2": "M2 — Recipe / BOM",
    "M3": "M3 — Production",
    "M4": "M4 — Costing",
    "M5": "M5 — Dashboard",
    "M6": "M6 — Onboarding & Polish",
}

issues = [
    # ── M0-A Infrastructure ──────────────────────────────────────────────────
    dict(
        title="[M0-A1] Repo init, .gitignore, CLAUDE.md",
        labels=["P0", "M0-infra"],
        milestone=M["M0"],
        body="""## Summary
Initialize git repo, configure `.gitignore`, commit project context document.

## Acceptance Criteria
- [ ] `.gitignore` excludes `.idea/`, `.vscode/`, `.claude/`, `.env`, `*.exe`, `/bin/`, `/dist/`
- [ ] `CLAUDE.md` committed and readable
- [ ] Remote set to `github.com/hopeIsCo0l/anu-Pro`, default branch `main`

## Status
✅ Done""",
    ),
    dict(
        title="[M0-A2] Go project directory scaffold",
        labels=["P0", "M0-infra", "backend"],
        milestone=M["M0"],
        body="""## Summary
Create the full directory tree for the modular monolith with stub `.go` files so `go build ./...` passes.

## Directories
```
cmd/api, cmd/worker, cmd/migrate
internal/platform/{config,db,auth,tenant,storage,httpserver}
internal/{inventory,production,recipe,costing,audit}
migrations/public/, migrations/tenant/
deploy/
```

## Acceptance Criteria
- [ ] `go build ./...` exits 0
- [ ] `cmd/api/main.go` starts chi HTTP server, `GET /health` → `200 {"status":"ok"}`
- [ ] Server reads port from `$PORT`, defaults `8080`
- [ ] Graceful shutdown on `SIGTERM`/`SIGINT` with 30s timeout

## Status
✅ Done""",
    ),
    dict(
        title="[M0-A3] pgx/v5 connection pool with tenant search_path switching",
        labels=["P0", "M0-infra", "backend", "database", "security"],
        milestone=M["M0"],
        body="""## Summary
`internal/platform/db/db.go` — pgx pool that switches `search_path` on every connection acquisition based on tenant slug from context. Reset on release to prevent cross-tenant leaks.

## Implementation Notes
- `BeforeAcquire`: `SET search_path TO tenant_<slug>, public`
- `AfterRelease`: `SET search_path TO public`
- Tenant slug injected via `context.Context` with typed key
- `WithTenant(ctx, slug)` and `FromContext(ctx)` helpers
- Pool limits: MaxConns=25, MinConns=2, HealthCheckPeriod=30s

## Acceptance Criteria
- [ ] `db.New(cfg)` returns connected pool or error
- [ ] `BeforeAcquire` sets correct `search_path` per tenant context
- [ ] `AfterRelease` always resets to `public`
- [ ] `TestCrossTenantIsolation` passes (two goroutines, different tenants, no data bleed)
- [ ] Uses real Postgres via testcontainers (no mocks)

## Dependencies
M0-A7""",
    ),
    dict(
        title="[M0-A4] Goose migration scaffold + first public migration",
        labels=["P0", "M0-infra", "database"],
        milestone=M["M0"],
        body="""## Summary
Set up `goose` as the migration tool. Split into `migrations/public/` and `migrations/tenant/`. `cmd/migrate` handles both.

## First Public Migration (`0001_public_schema.sql`)
Tables: `tenants`, `users`, `refresh_tokens`, `events` (partitioned monthly).

## CLI Interface
```
go run ./cmd/migrate up public
go run ./cmd/migrate status public
go run ./cmd/migrate down public
```

## Acceptance Criteria
- [ ] All four tables created after `up`
- [ ] `events` table partitioned by month
- [ ] Migration idempotent (running twice does not error)
- [ ] Tests run migrations in testcontainer (no mocks)

## Dependencies
M0-A2, M0-A3""",
    ),
    dict(
        title="[M0-A5] Docker Compose local dev environment",
        labels=["P0", "M0-infra"],
        milestone=M["M0"],
        body="""## Summary
`docker-compose up` reproduces full local dev: Postgres 16, MinIO, API server, worker. `.env.example` documents all env vars.

## Services
- `postgres`: postgres:16-alpine, port 5432, healthcheck
- `minio`: port 9000 (S3) + 9001 (console), auto-create bucket
- `api`: built from Dockerfile
- `worker`: same binary, CMD=worker
- `migrate`: one-shot, runs migrations and exits

## Acceptance Criteria
- [ ] `docker compose up` starts all services with no manual steps
- [ ] `curl localhost:8080/health` returns `{"status":"ok"}`
- [ ] `.env.example` documents every env var in the codebase
- [ ] `docker compose down -v` tears down cleanly

## Dependencies
M0-A2""",
    ),
    dict(
        title="[M0-A6] GitHub Actions CI/CD pipeline",
        labels=["P0", "M0-infra"],
        milestone=M["M0"],
        body="""## Summary
Two workflows: `ci.yml` (runs on every PR + push to main) and `deploy.yml` (runs after CI on main).

## CI Steps
1. `go vet ./...`
2. `golangci-lint run`
3. `go test ./... -race -count=1` (testcontainers real Postgres)
4. `go build ./cmd/api ./cmd/worker ./cmd/migrate`
5. Frontend lint + type-check (once scaffold exists)

## Deploy Steps
1. Build multi-stage Docker image
2. Push to GHCR with git SHA + `latest` tags
3. SSH to Oracle VM, pull, restart systemd service

## Acceptance Criteria
- [ ] PR cannot merge if CI fails
- [ ] Race detector enabled (`-race`)
- [ ] golangci-lint with errcheck enabled (`.golangci.yml`)
- [ ] Deploy uses `secrets.ORACLE_SSH_KEY`, `secrets.ORACLE_HOST`
- [ ] Final Docker image ~30MB (multi-stage builder)

## Dependencies
M0-A5""",
    ),
    dict(
        title="[M0-A7] envconfig-based Config struct",
        labels=["P0", "M0-infra", "backend"],
        milestone=M["M0"],
        body="""## Summary
`internal/platform/config/config.go` — all config from env vars via `kelseyhightower/envconfig`. No platform-specific APIs, no config files.

## Fields
`PORT`, `ENV`, `DATABASE_URL` (required), `JWT_SECRET` (required), `JWT_ACCESS_EXPIRY_MIN`, `JWT_REFRESH_EXPIRY_DAYS`, `STORAGE_ENDPOINT/BUCKET/ACCESS_KEY/SECRET_KEY` (required), `STORAGE_REGION`, `LOG_LEVEL`

## Acceptance Criteria
- [ ] Required fields fail fast on missing
- [ ] `config.Load()` returns `(*Config, error)`
- [ ] Test: `Load()` errors when `DATABASE_URL` missing
- [ ] Test: `Load()` populates all fields from env

## Status
✅ Done""",
    ),
    dict(
        title="[M0-A8] Structured logging setup (slog)",
        labels=["P0", "M0-infra", "backend"],
        milestone=M["M0"],
        body="""## Summary
`internal/platform/logger/logger.go` — centralize slog config. JSON in production, text in development. Chi request logging middleware.

## Acceptance Criteria
- [ ] JSON output in `ENV=production`
- [ ] Human-readable in `ENV=development`
- [ ] Every HTTP request logged: method, path, status, latency_ms, request_id
- [ ] `tenant_id` included when present in context
- [ ] `log_level=debug` shows SQL queries (pgx trace handler)
- [ ] No `fmt.Println` or `log.Printf` outside tests

## Dependencies
M0-A7""",
    ),
    dict(
        title="[M0-A9] Oracle Cloud VM provisioning + Caddy + systemd",
        labels=["P1", "M0-infra"],
        milestone=M["M0"],
        body="""## Summary
Provision Oracle Cloud Always Free ARM VM. Ubuntu 22.04, Postgres 16, Caddy 2 (auto HTTPS), systemd units for api + worker. All scripts in `deploy/oracle/`.

## Deliverables
- `deploy/oracle/provision.sh` (idempotent)
- `deploy/oracle/Caddyfile`
- `deploy/oracle/anu-api.service` + `anu-worker.service`
- `deploy/oracle/deploy.sh` (zero-downtime restart)

## Acceptance Criteria
- [ ] `provision.sh` is idempotent (safe to re-run)
- [ ] TLS cert auto-renewed by Caddy
- [ ] Postgres WAL archiving enabled
- [ ] `systemctl status anu-api` shows `active (running)`

## Dependencies
M0-A6""",
    ),
    dict(
        title="[M0-A10] Automated daily backups to Cloudflare R2 + restore test",
        labels=["P0", "M0-infra"],
        milestone=M["M0"],
        body="""## Summary
Non-negotiable. Implement before any business logic. Daily `pg_dump` → R2. Weekly automated restore test.

## Scripts
- `deploy/oracle/backup.sh` — `pg_dump --format=custom`, upload to R2 `anu-backups/YYYY/MM/DD/`, 30-day retention, alert on failure
- `deploy/oracle/test-restore.sh` — download latest, restore to temp db, smoke queries, drop, log result

## Schedule
- Backup: daily 02:00 UTC (systemd timer)
- Restore test: weekly Sunday 03:00 UTC (systemd timer)

## Acceptance Criteria
- [ ] Daily backup runs without manual intervention
- [ ] Backup verifiable with `pg_restore --list`
- [ ] Weekly restore smoke queries pass
- [ ] Failure sends email alert within 5 minutes
- [ ] 30-day retention enforced

## Dependencies
M0-A9""",
    ),
    dict(
        title="[M0-A11] Cross-tenant isolation integration test",
        labels=["P0", "M0-infra", "backend", "security"],
        milestone=M["M0"],
        body="""## Summary
Permanent P0 gate. Runs on every PR. Never skip.

`internal/platform/db/isolation_test.go`:
1. Spin Postgres via testcontainers
2. Create two tenants with separate schemas
3. Insert row into `tenant_alpha.items`
4. Query with `tenant_beta` context → assert 0 rows
5. Query with `tenant_alpha` context → assert 1 row
6. Concurrent goroutines, interleaved queries

## Acceptance Criteria
- [ ] `TestCrossTenantIsolation` — do not rename
- [ ] Real Postgres, no mocks
- [ ] Covers concurrent access
- [ ] Fails loudly if `search_path` misconfigured

## Dependencies
M0-A3, M0-A4""",
    ),
    # ── M0-B Auth ────────────────────────────────────────────────────────────
    dict(
        title="[M0-B1] Argon2id password hashing",
        labels=["P0", "M0-auth", "backend", "security"],
        milestone=M["M0"],
        body="""## Summary
`internal/platform/auth/password.go` — hash and verify with Argon2id (not bcrypt). OWASP params: memory=64MB, iterations=3, parallelism=2, salt=16B, key=32B.

## Acceptance Criteria
- [ ] `HashPassword(plain) (string, error)` returns encoded hash
- [ ] `VerifyPassword(plain, encoded) (bool, error)` constant-time compare
- [ ] Different hash each call (random salt)
- [ ] Wrong password → `false`, no error
- [ ] Benchmark: hash < 500ms
- [ ] Round-trip test passes
- [ ] Tampered hash → false, not error

## Dependencies
M0-A2""",
    ),
    dict(
        title="[M0-B2] JWT issuance (access + refresh tokens)",
        labels=["P0", "M0-auth", "backend", "security"],
        milestone=M["M0"],
        body="""## Summary
`internal/platform/auth/jwt.go` — issue and validate JWTs using `golang-jwt/jwt/v5`.

## Access Token Claims
`sub`, `tid` (tenant_id), `tslug`, `role`, `exp`, `iat`, `jti` (unique UUID)

## Functions
- `IssueAccessToken(cfg, claims) (string, error)`
- `ValidateAccessToken(cfg, tokenStr) (*Claims, error)`
- `IssueRefreshToken() (plain, hash string, error)` — opaque token, stored as SHA-256 hash

## Acceptance Criteria
- [ ] Signed with HS256 using `JWT_SECRET`
- [ ] Expired token → `ErrTokenExpired`
- [ ] Tampered token → `ErrTokenInvalid`
- [ ] Refresh token = 32 random bytes, hex-encoded
- [ ] Refresh token hash = SHA-256 of plain

## Dependencies
M0-A7""",
    ),
    dict(
        title="[M0-B3] Refresh token rotation",
        labels=["P0", "M0-auth", "backend", "security"],
        milestone=M["M0"],
        body="""## Summary
`internal/platform/auth/refresh.go` — stateful rotation with reuse detection.

## Flow
1. Client sends refresh token (HttpOnly cookie)
2. Hash it, look up in DB (`not revoked, not expired`)
3. Found → revoke old, issue new access + refresh tokens (single transaction)
4. Not found → 401
5. Reuse of revoked token → revoke ALL user tokens (session hijack detection)

## Acceptance Criteria
- [ ] Successful refresh: old revoked, new tokens in one transaction
- [ ] Reuse detection triggers full revocation
- [ ] `HttpOnly; Secure; SameSite=Strict` cookie attributes
- [ ] Test: 3-round rotation, reuse detection

## Dependencies
M0-B2, M0-A4""",
    ),
    dict(
        title="[M0-B4] Signup endpoint — POST /api/v1/auth/signup",
        labels=["P0", "M0-auth", "backend"],
        milestone=M["M0"],
        body="""## Summary
Create new tenant + first owner user in a single transaction. Also provisions tenant schema.

## Request
`tenant_name`, `tenant_slug` (3-50 chars, lowercase alphanumeric+hyphens), `full_name`, `email`, `password` (min 12 chars), `timezone`

## Response
`201` with `access_token`, `user`, `tenant`. Refresh token in HttpOnly cookie.

## Acceptance Criteria
- [ ] Creates `public.tenants` row
- [ ] Creates schema `tenant_<slug>` and runs all tenant migrations
- [ ] Creates `public.users` row with role `owner`
- [ ] Argon2id password hash
- [ ] Emits `user.signed_up` event
- [ ] 409 on duplicate slug or email
- [ ] 422 with field errors for validation failures
- [ ] Entire operation in one DB transaction

## Dependencies
M0-B1, M0-B2, M0-A4, M0-C1""",
    ),
    dict(
        title="[M0-B5] Login endpoint — POST /api/v1/auth/login",
        labels=["P0", "M0-auth", "backend"],
        milestone=M["M0"],
        body="""## Summary
Authenticate user, return access token + set refresh cookie.

## Security
- Constant-time compare (Argon2id verify)
- Same error for unknown email and wrong password (no user enumeration)
- Rate limit: 5 failed attempts per email per 15 minutes → 429
- Update `last_login_at` on success

## Acceptance Criteria
- [ ] 200 + tokens on correct credentials
- [ ] 401 `invalid credentials` for wrong password OR unknown email
- [ ] 401 for inactive user
- [ ] 429 after 5 failed attempts
- [ ] Emits `user.logged_in` event

## Dependencies
M0-B1, M0-B2, M0-B3""",
    ),
    dict(
        title="[M0-B6] Logout + token revocation",
        labels=["P1", "M0-auth", "backend"],
        milestone=M["M0"],
        body="""## Summary
`POST /api/v1/auth/logout` — revoke current refresh token, clear cookie.
`POST /api/v1/auth/logout-all` — revoke ALL refresh tokens for user.

## Acceptance Criteria
- [ ] Sets `revoked_at = now()` on refresh token
- [ ] Clears HttpOnly cookie
- [ ] Logout-all revokes all non-revoked tokens
- [ ] Both idempotent (200 even if already revoked)
- [ ] Emits `user.logged_out` event

## Dependencies
M0-B3""",
    ),
    dict(
        title="[M0-B7] Auth middleware — JWT validation + tenant resolution",
        labels=["P0", "M0-auth", "backend", "security"],
        milestone=M["M0"],
        body="""## Summary
Chi middleware: extract Bearer token, validate JWT, load tenant (verify not suspended), inject `user_id`, `tenant_id`, `tenant_slug`, `role` into context. Sets `search_path` context key for pgx pool.

Also: `RequireRole(roles ...string)` middleware.

## Acceptance Criteria
- [ ] Missing/malformed token → 401
- [ ] Expired token → 401 `token_expired`
- [ ] Suspended tenant → 403 `tenant_suspended`
- [ ] `RequireRole("owner", "manager")` returns 403 for operator
- [ ] Public routes (`/health`, `/api/v1/auth/*`) bypass middleware

## Dependencies
M0-B2, M0-A3""",
    ),
    dict(
        title="[M0-B8] Role-based access control (RBAC)",
        labels=["P1", "M0-auth", "backend", "security"],
        milestone=M["M0"],
        body="""## Summary
Four roles: `owner`, `manager`, `operator`, `viewer`. Role stored in JWT claim — no per-request DB lookup.

## Permission Matrix
| Action | owner | manager | operator | viewer |
|---|---|---|---|---|
| Manage users | ✓ | | | |
| Manage items/recipes | ✓ | ✓ | | |
| Create/complete production jobs | ✓ | ✓ | ✓ | |
| View all data | ✓ | ✓ | ✓ | ✓ |
| Export/import | ✓ | ✓ | | |
| Tenant settings | ✓ | | | |

## Acceptance Criteria
- [ ] `RequireRole` middleware usable on any route
- [ ] Tests cover boundary cases (operator creates item → 403)

## Dependencies
M0-B7""",
    ),
    # ── M0-C Tenancy ─────────────────────────────────────────────────────────
    dict(
        title="[M0-C1] Tenant schema provisioning",
        labels=["P0", "M0-tenancy", "backend", "database"],
        milestone=M["M0"],
        body="""## Summary
`internal/platform/tenant/provisioner.go` — on new tenant creation: create schema `tenant_<slug>`, run all `migrations/tenant/` against it, record in `public.tenants`. Atomic: failed migration drops schema and rolls back.

## First Tenant Migration includes
`items`, `locations`, `lots`, `stock_movements` (partitioned, append-only), `current_stock` view, tenant-level `events` (partitioned).

## Acceptance Criteria
- [ ] `ProvisionTenant(ctx, db, slug, ...)` atomic
- [ ] Failed migration → schema dropped, error returned
- [ ] Idempotent: calling twice returns error `tenant already exists`
- [ ] `current_stock` view returns correct SUM
- [ ] Cross-tenant isolation test passes after provisioning two tenants

## Dependencies
M0-A4""",
    ),
    dict(
        title="[M0-C2] Per-tenant migration runner",
        labels=["P0", "M0-tenancy", "database"],
        milestone=M["M0"],
        body="""## Summary
Apply new tenant migrations to all existing tenant schemas.

## CLI
```
go run ./cmd/migrate up tenant:sunrise-dairy   # single tenant
go run ./cmd/migrate up tenant:*               # all tenants
go run ./cmd/migrate status tenant:*           # show status per tenant
```

## Acceptance Criteria
- [ ] `tenant:*` iterates all `public.tenants` rows
- [ ] Goose version table per schema: `tenant_<slug>.goose_db_version`
- [ ] Reports tenants behind latest version
- [ ] `--dry-run` prints SQL without executing
- [ ] Test: 3 tenants at different versions, upgrade all

## Dependencies
M0-C1""",
    ),
    dict(
        title="[M0-C3] Row-Level Security (defense in depth)",
        labels=["P1", "M0-tenancy", "database", "security"],
        milestone=M["M0"],
        body="""## Summary
RLS on all tenant tables as secondary guard behind schema-per-tenant. Policy uses `current_setting('app.tenant_slug')` set in `BeforeAcquire` hook.

## Acceptance Criteria
- [ ] RLS enabled on: `items`, `lots`, `locations`, `stock_movements`, `events` (tenant schema)
- [ ] Direct `psql` query with wrong `app.tenant_slug` → 0 rows
- [ ] Application queries unaffected
- [ ] RLS bypass test: misconfigured `search_path` still blocked by RLS

## Dependencies
M0-A3, M0-C1""",
    ),
    # ── M1 Inventory ─────────────────────────────────────────────────────────
    dict(
        title="[M1-A1] Items catalog — FTS indexes migration",
        labels=["P0", "M1-inventory", "database"],
        milestone=M["M1"],
        body="""## Summary
Add GIN full-text search index on `items.name`, category + active partial indexes. Migration `0002_items_indexes.sql`.

## Acceptance Criteria
- [ ] `CREATE INDEX items_name_idx ON items USING gin(to_tsvector('english', name))`
- [ ] `CREATE INDEX items_category_idx ON items(category) WHERE category IS NOT NULL`
- [ ] Migration applies cleanly

## Dependencies
M0-C1""",
    ),
    dict(
        title="[M1-A2] Item CRUD endpoints",
        labels=["P0", "M1-inventory", "backend"],
        milestone=M["M1"],
        body="""## Summary
```
POST   /api/v1/items        201 (manager+)
GET    /api/v1/items        200 (viewer+)
GET    /api/v1/items/:id    200 (viewer+)
PUT    /api/v1/items/:id    200 (manager+)
DELETE /api/v1/items/:id    204 soft-delete (manager+)
```
sqlc for all queries. `attributes` JSONB field.

## Acceptance Criteria
- [ ] `POST` returns 201 with full item body
- [ ] `DELETE` sets `is_active=false` (soft delete, preserves ledger history)
- [ ] SKU unique constraint → 409
- [ ] `updated_at` updated on every PUT
- [ ] All queries via sqlc generated code (no raw string SQL in handlers)

## Dependencies
M0-A3, M0-B7, M0-C1, M1-A1""",
    ),
    dict(
        title="[M1-A3] Item search and cursor-based pagination",
        labels=["P1", "M1-inventory", "backend"],
        milestone=M["M1"],
        body="""## Summary
`GET /api/v1/items` with: FTS `?q=`, category filter, active filter, cursor pagination (`?cursor=&limit=`), sort by name/sku/created_at.

Cursor encodes `(created_at, id)` as base64 JSON (stable sort, no offset).

## Acceptance Criteria
- [ ] `?q=milk` uses GIN/tsvector index
- [ ] Cursor is opaque to clients
- [ ] `next_cursor` null on last page
- [ ] Over-limit capped at 200
- [ ] Empty result: `{"items":[],"next_cursor":null}`

## Dependencies
M1-A2""",
    ),
    dict(
        title="[M1-A4] Units of measure (UoM) + item-specific conversions",
        labels=["P1", "M1-inventory", "backend", "database"],
        milestone=M["M1"],
        body="""## Summary
`units_of_measure` and `item_uom_conversions` tables. Standard units seeded (kg, g, L, mL, each, box, case). Pure `Convert()` function in `internal/inventory/uom/converter.go`.

## Acceptance Criteria
- [ ] `Convert(1, "kg", "g", nil)` = 1000
- [ ] Item-specific conversions override standard
- [ ] Returns error for incompatible unit types
- [ ] Never float64 — shopspring/decimal throughout
- [ ] Unit tests: kg→g, L→mL, case→each

## Dependencies
M1-A1""",
    ),
    dict(
        title="[M1-B1] Lot CRUD endpoints",
        labels=["P0", "M1-inventory", "backend"],
        milestone=M["M1"],
        body="""## Summary
```
POST   /api/v1/lots              201 (operator+)
GET    /api/v1/lots              200 (viewer+)
GET    /api/v1/lots/:id          200 (viewer+)
PUT    /api/v1/lots/:id          200 (manager+)
GET    /api/v1/items/:id/lots    200 (viewer+)
```

Lot number unique per item (not globally). `expires_at` optional. Status starts `active`.

## Acceptance Criteria
- [ ] Duplicate `(item_id, lot_number)` → 409
- [ ] List filter by `item_id`, `status`, `expiring_before`
- [ ] `PUT` allows updating `expires_at`, `attributes`, `status` only

## Dependencies
M0-B7, M0-C1""",
    ),
    dict(
        title="[M1-B2] Lot status transitions + expiry background job",
        labels=["P1", "M1-inventory", "backend"],
        milestone=M["M1"],
        body="""## Summary
State machine: `active → quarantine → active`, `active → consumed`, `active|quarantine → expired`.

`PATCH /api/v1/lots/:id/status` for explicit transitions.

River background job runs hourly: mark `expired` where `expires_at < now() AND status = 'active'`, emit `lot.expired` event.

## Acceptance Criteria
- [ ] Invalid transitions → 422
- [ ] Cannot move stock out of `quarantine` lot
- [ ] `lot.expired` event written in same transaction
- [ ] Test: state machine covers all transitions

## Dependencies
M1-B1""",
    ),
    dict(
        title="[M1-C1] Location CRUD endpoints",
        labels=["P1", "M1-inventory", "backend"],
        milestone=M["M1"],
        body="""## Summary
Hierarchical locations: `warehouse > zone > bin`. Self-referential `parent_id`. `?format=tree` returns nested JSON.

## Acceptance Criteria
- [ ] Cycle detection in application layer
- [ ] `?format=tree` returns properly nested structure
- [ ] Deactivating location deactivates all children
- [ ] Cannot deactivate if location has current stock

## Dependencies
M0-C1""",
    ),
    dict(
        title="[M1-D1] Receipt movement endpoint — POST /api/v1/stock/receive",
        labels=["P0", "M1-inventory", "backend"],
        milestone=M["M1"],
        body="""## Summary
Record incoming stock. Writes `stock_movements` row (`movement_type=receipt`, positive quantity). Emits `stock.received` event in same transaction.

## Key Fields
`lot_id`, `location_id`, `quantity` (positive, decimal), `unit_cost` (for WAC), `notes`, `idempotency_key`

## Acceptance Criteria
- [ ] Quantity must be positive (422 otherwise)
- [ ] Duplicate `idempotency_key` → 200 with original response (not 409)
- [ ] Quantity stored as `NUMERIC(18,6)` via shopspring/decimal
- [ ] `current_stock` view reflects balance immediately
- [ ] `stock.received` event in same transaction

## Dependencies
M0-B7, M0-C1""",
    ),
    dict(
        title="[M1-D2] Adjustment movement endpoint — POST /api/v1/stock/adjust",
        labels=["P1", "M1-inventory", "backend"],
        milestone=M["M1"],
        body="""## Summary
Corrections, waste, cycle count reconciliation. `movement_type` ∈ {adjustment, waste, cycle_count}. Blocks if resulting balance < 0.

## Acceptance Criteria
- [ ] Blocks if `(current_balance + quantity) < 0`
- [ ] Current balance in 422 error body
- [ ] `SELECT FOR UPDATE` on aggregated balance before insert
- [ ] `stock.adjusted` event in same transaction

## Dependencies
M1-D1""",
    ),
    dict(
        title="[M1-D3] Transfer movement endpoint — POST /api/v1/stock/transfer",
        labels=["P1", "M1-inventory", "backend"],
        milestone=M["M1"],
        body="""## Summary
Move stock between locations. Two movements atomically: `transfer_out` (negative) + `transfer_in` (positive), sharing the same `reference_id` UUID.

## Acceptance Criteria
- [ ] Both movements in one transaction
- [ ] `from_location == to_location` → 422
- [ ] Blocks if source would go negative
- [ ] `stock.transferred` event in same transaction

## Dependencies
M1-D2""",
    ),
    dict(
        title="[M1-D4] Stock query endpoints",
        labels=["P1", "M1-inventory", "backend"],
        milestone=M["M1"],
        body="""## Summary
```
GET /api/v1/stock                    current stock (queries current_stock view)
GET /api/v1/stock?item_id=...
GET /api/v1/stock?location_id=...
GET /api/v1/stock/movements          movement history (paginated, most-recent-first)
GET /api/v1/stock/movements?lot_id=...
GET /api/v1/stock/movements?item_id=...
```

## Acceptance Criteria
- [ ] Zero-balance lots excluded from `/stock` response
- [ ] Movements paginated with cursor
- [ ] `?from=&to=` date range filter on movements

## Dependencies
M0-C1, M1-D1""",
    ),
    dict(
        title="[M1-D5] Low-stock alert system",
        labels=["P2", "M1-inventory", "backend"],
        milestone=M["M1"],
        body="""## Summary
`items.min_stock_qty` column. River job every 15 minutes: compare current stock vs threshold, emit `stock.low` event (debounced 6 hours per item).

## Acceptance Criteria
- [ ] `items.min_stock_qty NUMERIC(18,6)` via migration
- [ ] River job `CheckLowStockJob` runs every 15 minutes
- [ ] `stock.low` payload: item_id, item_name, current_qty, threshold_qty
- [ ] No duplicate alerts within 6 hours per item

## Dependencies
M1-D4""",
    ),
    dict(
        title="[M1-E1] CSV import — items",
        labels=["P1", "M1-inventory", "backend"],
        milestone=M["M1"],
        body="""## Summary
`POST /api/v1/import/items` — multipart upload. Upserts items by SKU. Returns import report (created/updated/failed counts + row-level errors).

## CSV Format
`sku, name, unit, category, description, attributes_json`

## Acceptance Criteria
- [ ] Max 10MB, UTF-8 required
- [ ] Row-level errors collected (not fail-fast)
- [ ] Idempotent (upsert by SKU)
- [ ] Returns 200 with report even if some rows failed
- [ ] Test: 100-row happy path, partial failure, duplicate SKU upsert

## Dependencies
M1-A2""",
    ),
    dict(
        title="[M1-E2] CSV import — opening stock balances",
        labels=["P1", "M1-inventory", "backend"],
        milestone=M["M1"],
        body="""## Summary
`POST /api/v1/import/stock` — bulk receipt movements to establish opening balances. All movements in single transaction with shared `import_batch_id`. Dry-run mode: `?dry_run=true`.

## CSV Format
`sku, lot_number, location_code, quantity, expires_at, supplier_ref`

## Acceptance Criteria
- [ ] All movements in one transaction
- [ ] `import_batch_id` returned for audit trail
- [ ] Partial failure rolls back entire batch
- [ ] `?dry_run=true` validates without committing
- [ ] Test: 50-row import, dry run, rollback on error

## Dependencies
M1-D1, M1-E1""",
    ),
    dict(
        title="[M1-F1] Next.js frontend scaffold",
        labels=["P0", "M1-inventory", "frontend"],
        milestone=M["M1"],
        body="""## Summary
Initialize `frontend/` with Next.js 14 (App Router), TypeScript, Tailwind CSS, shadcn/ui, TanStack Query v5.

## Structure
```
frontend/app/(auth)/login, signup
frontend/app/(app)/dashboard, inventory, lots, stock
frontend/lib/api.ts (typed fetch wrapper)
frontend/components/ui/ (shadcn)
```

## Acceptance Criteria
- [ ] `pnpm dev` starts at localhost:3000
- [ ] `pnpm build` exits 0
- [ ] `pnpm tsc --noEmit` exits 0
- [ ] Auth guard: redirect to `/login` if no access token
- [ ] Access token in memory (not localStorage), refresh via HttpOnly cookie
- [ ] TanStack Query v5: retry=1, staleTime=30s

## Dependencies
M0-B5""",
    ),
    dict(
        title="[M1-F2] Items list + detail pages",
        labels=["P1", "M1-inventory", "frontend"],
        milestone=M["M1"],
        body="""## Summary
`/inventory`: searchable/filterable table with slide-over "New Item" form.
`/inventory/:id`: detail form (editable), Lots tab, Movement History tab.

## Acceptance Criteria
- [ ] Search debounced 300ms
- [ ] Cursor-based "Load more"
- [ ] Optimistic update on edit
- [ ] Toast on save success/failure
- [ ] Mobile responsive

## Dependencies
M1-F1, M1-A2, M1-A3""",
    ),
    dict(
        title="[M1-F3] Stock overview page",
        labels=["P1", "M1-inventory", "frontend"],
        milestone=M["M1"],
        body="""## Summary
`/stock`: summary cards, item table with lot drill-down, Receive/Adjust/Transfer modals.

## Acceptance Criteria
- [ ] Auto-refresh every 60 seconds
- [ ] Low stock highlighted amber, zero qty highlighted red
- [ ] Modals validate before submit
- [ ] Lot expiry badge with days-remaining
- [ ] Operator-friendly: min 44px tap targets

## Dependencies
M1-F1, M1-D1, M1-D2, M1-D3, M1-D4""",
    ),
    # ── M2 Recipe ─────────────────────────────────────────────────────────────
    dict(
        title="[M2-A1] Recipe + recipe_versions migration",
        labels=["P0", "M2-recipe", "database"],
        milestone=M["M2"],
        body="""## Summary
`migrations/tenant/0004_recipes.sql`: `recipes`, `recipe_versions`, `recipe_inputs`, `recipe_outputs` tables.

Key constraints: unique `(recipe_id, version_number)`, `cost_per_unit` nullable (computed on publish), `output_type` ∈ {primary, co-product, by-product}.

## Dependencies
M0-C1""",
    ),
    dict(
        title="[M2-A2] Recipe CRUD + versioning endpoints",
        labels=["P0", "M2-recipe", "backend"],
        milestone=M["M2"],
        body="""## Summary
Full recipe lifecycle:
```
POST   /api/v1/recipes
GET    /api/v1/recipes/:id/versions
POST   /api/v1/recipes/:id/versions/:v/publish
POST   /api/v1/recipes/:id/versions/:v/archive
```
Publishing computes `cost_per_unit`. At most one published version per recipe.

## Acceptance Criteria
- [ ] Cannot edit published or archived version (422)
- [ ] Cannot publish if no inputs or no primary output
- [ ] Publishing old version auto-archives current published
- [ ] `version_number` sequential per recipe
- [ ] Test: full draft→publish→new draft→publish lifecycle

## Dependencies
M2-A1, M0-B7""",
    ),
    dict(
        title="[M2-B1] BOM engine — pure domain logic",
        labels=["P0", "M2-recipe", "backend"],
        milestone=M["M2"],
        body="""## Summary
`internal/recipe/engine/bom.go` — pure function, zero I/O dependencies. Scales recipe to target qty with yield variance, waste %, co-products.

## Math
```
scale_factor = targetQty / batchSize
input_needed = input.qty × scale_factor × (1 + waste_pct/100)
output_expected = output.qty × scale_factor × (yield_pct/100)
cost = Σ (input_needed × itemCosts[input.item_id])
cost_per_unit = cost / primary_output_expected
```

## Acceptance Criteria
- [ ] Zero imports of `net/http`, DB packages, or any I/O
- [ ] Zero float64 — shopspring/decimal throughout
- [ ] Co-products scaled but excluded from cost_per_unit denominator
- [ ] Substitution uses cheaper item when available
- [ ] 10+ table-driven tests including edge cases

## Dependencies
M2-A1""",
    ),
    dict(
        title="[M2-B2] Recursive BOM explosion",
        labels=["P2", "M2-recipe", "backend"],
        milestone=M["M2"],
        body="""## Summary
`internal/recipe/engine/explode.go` — explode multi-level BOM to leaf raw materials with full cost rollup.

## Acceptance Criteria
- [ ] Handles 3-level deep BOM
- [ ] Cycle detection (returns error)
- [ ] Max depth 5 enforced
- [ ] All decimal arithmetic, no float64

## Dependencies
M2-B1""",
    ),
    dict(
        title="[M2-C1] Recipe cost computation on publish",
        labels=["P1", "M2-recipe", "backend"],
        milestone=M["M2"],
        body="""## Summary
At publish time: call BOM engine with current `item_costs` WAC values. Store result in `recipe_versions.cost_per_unit`.

`GET /api/v1/costs/recipes/:version_id` — theoretical cost at current WAC.

## Acceptance Criteria
- [ ] `cost_per_unit` populated on publish
- [ ] Uses shopspring/decimal
- [ ] Cost breakdown endpoint shows per-input contribution %

## Dependencies
M2-A2, M2-B1""",
    ),
    dict(
        title="[M2-D1] Recipe list + builder UI",
        labels=["P1", "M2-recipe", "frontend"],
        milestone=M["M2"],
        body="""## Summary
`/recipes`: list with cost/unit column.
`/recipes/:id`: BOM tab (ingredient list), Versions tab, Cost breakdown tab.

Builder: dynamic input rows, real-time cost preview (debounced), yield % slider, unit selector.

## Acceptance Criteria
- [ ] Add/remove ingredient rows without reload
- [ ] Cost preview updates on qty/unit change
- [ ] Publish button disabled when validation fails
- [ ] Version history shows which inputs changed between versions

## Dependencies
M1-F1, M2-A2""",
    ),
    # ── M3 Production ─────────────────────────────────────────────────────────
    dict(
        title="[M3-A1] Production jobs + consumption + outputs migration",
        labels=["P0", "M3-production", "database"],
        milestone=M["M3"],
        body="""## Summary
`migrations/tenant/0005_production.sql`: `production_jobs`, `production_consumption`, `production_outputs` tables with indexes on status, recipe_version_id, lot_id.

Auto-generated job numbers: `PRD-YYYY-NNNN`.

## Dependencies
M2-A1, M0-C1""",
    ),
    dict(
        title="[M3-A2] Production job lifecycle endpoints",
        labels=["P0", "M3-production", "backend"],
        milestone=M["M3"],
        body="""## Summary
Full state machine: `draft → scheduled → in_progress → completed → cancelled`

Key endpoints:
- `POST /jobs/:id/start` — validate stock, reserve input lots, write consumption rows (atomic)
- `POST /jobs/:id/complete` — accept actual qtys, reconcile variance, write output stock movements, update WAC
- `POST /jobs/:id/cancel` — reverse reserved stock movements

## Acceptance Criteria
- [ ] Insufficient stock on start → 422 with per-lot breakdown
- [ ] Start is fully atomic (all movements or none)
- [ ] Complete handles yield variance (actual ≠ planned)
- [ ] Cancel reverses reserved stock correctly
- [ ] `production_job.completed` and `production_job.cancelled` events emitted

## Dependencies
M3-A1, M1-D1, M0-B8""",
    ),
    dict(
        title="[M3-B1] Forward + backward traceability queries",
        labels=["P0", "M3-production", "backend"],
        milestone=M["M3"],
        body="""## Summary
Recursive CTE trace queries.

```
GET /api/v1/traceability/forward/:lot_id   → consumption chain
GET /api/v1/traceability/backward/:lot_id  → ingredient chain
GET /api/v1/traceability/recall/:lot_id    → all affected lots
```

## Acceptance Criteria
- [ ] Forward trace: lot → jobs → output lots (recursive, max depth 10)
- [ ] Backward trace: output lot → job → input lots → origins
- [ ] Recall query: all lots in forward chain, CSV export
- [ ] Handles cycles via depth limit
- [ ] Performance < 500ms for 5-level deep trace on 1M movement rows

## Dependencies
M3-A2""",
    ),
    dict(
        title="[M3-A3] Production jobs list + execution UI",
        labels=["P1", "M3-production", "frontend"],
        milestone=M["M3"],
        body="""## Summary
`/production`: Kanban columns (Draft|Scheduled|In Progress|Completed).
`/production/:id`: Inputs tab (lot picker with FEFO suggestion), Outputs tab (assign lot numbers), action buttons.

## Acceptance Criteria
- [ ] Kanban updates optimistically on status change
- [ ] Lot picker shows expiry + available qty
- [ ] FEFO suggestion highlighted (not enforced)
- [ ] Warning if actual qty >20% variance from planned
- [ ] Operator-mode: large buttons, simple layout

## Dependencies
M1-F1, M3-A2""",
    ),
    dict(
        title="[M3-B2] Traceability explorer UI",
        labels=["P1", "M3-production", "frontend"],
        milestone=M["M3"],
        body="""## Summary
`/traceability`: lot search, Forward/Backward trace tabs as indented table, recall impact button with CSV export.

## Acceptance Criteria
- [ ] Search by partial lot number or item name
- [ ] Indented table (not force-graph)
- [ ] Each row: lot number, item, qty, date, status
- [ ] Recall CSV: lot_id, lot_number, item_name, qty, location, status
- [ ] Empty state for raw materials with no production history

## Dependencies
M1-F1, M3-B1""",
    ),
    dict(
        title="[M3-C1] FEFO lot suggestion engine",
        labels=["P2", "M3-production", "backend"],
        milestone=M["M3"],
        body="""## Summary
`internal/production/fefo/fefo.go` — pure function. Given a list of available lots for an item, return sorted by `expires_at ASC NULLS LAST` with sufficient quantity to fulfill the required amount.

## Acceptance Criteria
- [ ] No I/O dependencies
- [ ] Returns ordered list: soonest-expiring first
- [ ] Skips quarantine lots
- [ ] Handles partial lots (multiple lots to fill required qty)
- [ ] Returns error if insufficient total stock

## Dependencies
M1-B1""",
    ),
    # ── M4 Costing ───────────────────────────────────────────────────────────
    dict(
        title="[M4-A1] Weighted average cost migration",
        labels=["P1", "M4-costing", "database"],
        milestone=M["M4"],
        body="""## Summary
`migrations/tenant/0006_costing.sql`: `item_costs` (current WAC per item) + `cost_history` (point-in-time snapshots, indexed by item_id + snapshot_at).

## Dependencies
M0-C1""",
    ),
    dict(
        title="[M4-A2] WAC recalculation on receipt",
        labels=["P1", "M4-costing", "backend"],
        milestone=M["M4"],
        body="""## Summary
`internal/costing/wac.go` — recalculate WAC on every receipt:
```
new_wac = (current_qty × current_wac + received_qty × unit_cost) / (current_qty + received_qty)
```
Upsert `item_costs` + append `cost_history` in same transaction as movement.

## Acceptance Criteria
- [ ] Receipt endpoint requires `unit_cost` > 0
- [ ] Decimal arithmetic throughout (no float64)
- [ ] First receipt: WAC = unit_cost
- [ ] `cost_history` row written with trigger=`receipt`

## Dependencies
M1-D1, M4-A1""",
    ),
    dict(
        title="[M4-A3] WAC recalculation on production completion",
        labels=["P1", "M4-costing", "backend"],
        milestone=M["M4"],
        body="""## Summary
On job completion: `production_cost = Σ(actual_input_qty × input_item_wac)`. Distribute across outputs. Update `item_costs` for all output items. By-products get WAC = 0.

## Acceptance Criteria
- [ ] WAC updated for all output items
- [ ] Co-product WAC proportional to output qty
- [ ] By-product WAC = 0
- [ ] `cost_history` written for each output item with trigger=`production`

## Dependencies
M3-A2, M4-A2""",
    ),
    dict(
        title="[M4-A4] Cost query endpoints",
        labels=["P2", "M4-costing", "backend"],
        milestone=M["M4"],
        body="""## Summary
```
GET /api/v1/costs/items                  current WAC for all items
GET /api/v1/costs/items/:id/history      WAC history (90 points for sparkline)
GET /api/v1/costs/jobs/:id               job cost breakdown (planned vs actual)
GET /api/v1/costs/recipes/:version_id    theoretical cost at current WAC
```

## Dependencies
M4-A2, M4-A3, M2-B1""",
    ),
    # ── M5 Dashboard ──────────────────────────────────────────────────────────
    dict(
        title="[M5-A1] Dashboard API endpoint",
        labels=["P1", "M5-dashboard", "backend"],
        milestone=M["M5"],
        body="""## Summary
`GET /api/v1/dashboard` — single request returns all dashboard data. Sub-queries parallelized via goroutines. In-memory cache per tenant (60s TTL).

## Response Shape
`low_stock_items`, `expiring_lots` (next 30 days), `jobs_in_progress`, `recent_movements` (last 20), `kpis` (total_skus, active_lots, jobs_completed_7d, movements_7d)

## Acceptance Criteria
- [ ] Single HTTP request
- [ ] Sub-queries parallelized (not sequential)
- [ ] Cached 60s per tenant
- [ ] Never returns data older than 2 minutes
- [ ] Response time < 500ms

## Dependencies
M1-D5, M3-A2""",
    ),
    dict(
        title="[M5-A2] Dashboard frontend",
        labels=["P1", "M5-dashboard", "frontend"],
        milestone=M["M5"],
        body="""## Summary
`/dashboard`: KPI cards row, Low Stock + Expiring Lots tables, Recent Activity feed, Jobs In Progress list.

## Acceptance Criteria
- [ ] Dashboard loads < 2 seconds
- [ ] Auto-refreshes every 60 seconds
- [ ] Low stock items clickable → `/inventory/:id`
- [ ] Activity feed: human-readable descriptions
- [ ] Numbers locale-formatted

## Dependencies
M1-F1, M5-A1""",
    ),
    # ── M6 Polish ─────────────────────────────────────────────────────────────
    dict(
        title="[M6-A1] Tenant onboarding wizard",
        labels=["P1", "M6-polish", "frontend"],
        milestone=M["M6"],
        body="""## Summary
4-step wizard shown after signup: company details → locations → categories → import/skip. State in `tenant.onboarding_step` JSONB.

## Acceptance Criteria
- [ ] Shown on first login, dismissed after step 4 or skip
- [ ] Each step validates before advancing
- [ ] Back button works
- [ ] Progress indicator (step N of 4)

## Dependencies
M1-F1, M1-E1""",
    ),
    dict(
        title="[M6-A2] User management — invite, role change, deactivate",
        labels=["P1", "M6-polish", "backend", "frontend"],
        milestone=M["M6"],
        body="""## Summary
`/settings/users` (owner only). Invite flow: email + role → invite token (48h expiry) → email link → set password.

```
POST /api/v1/users/invite
GET  /api/v1/users
PUT  /api/v1/users/:id/role
POST /api/v1/users/:id/deactivate
POST /api/v1/users/:id/reactivate
```

## Acceptance Criteria
- [ ] Invite token expires 48h
- [ ] Cannot deactivate own account
- [ ] Deactivated users → 401 on login
- [ ] Cannot change own role

## Dependencies
M0-B4, M0-B8""",
    ),
    dict(
        title="[M6-B1] RFC 7807 error handling standardization",
        labels=["P1", "M6-polish", "backend"],
        milestone=M["M6"],
        body="""## Summary
All 4xx/5xx responses use Problem+JSON format. Central error handler in chi middleware. Sentinel error types in `internal/platform/httpserver/errors.go`.

## Response Format
```json
{"type":"https://anupro.app/errors/...","title":"...","status":422,"detail":"...","extensions":{}}
```

## Acceptance Criteria
- [ ] `Content-Type: application/problem+json` on all error responses
- [ ] 500s log full stack trace but return generic message
- [ ] All existing endpoints updated
- [ ] Error types documented in `docs/errors.md`""",
    ),
    dict(
        title="[M6-B2] API rate limiting per tenant",
        labels=["P2", "M6-polish", "backend"],
        milestone=M["M6"],
        body="""## Summary
In-memory sliding window rate limiter (no Redis needed). Keyed by `tenant_id`.

Limits: default 500 req/min, import endpoints 10 req/min, auth endpoints 20 req/min per IP.

## Acceptance Criteria
- [ ] 501st request → 429 with `Retry-After` header
- [ ] Import endpoints have lower limit
- [ ] Auth endpoints rate-limited by IP

## Dependencies
M0-B7""",
    ),
    dict(
        title="[M6-B3] Request validation middleware",
        labels=["P1", "M6-polish", "backend"],
        milestone=M["M6"],
        body="""## Summary
`internal/platform/httpserver/validate.go` — central validation using `go-playground/validator/v10`. All field errors returned at once. Custom validators for `decimal.Decimal > 0` and lot_number format.

## Response Format
```json
{"fields":[{"field":"quantity","error":"must be greater than 0"}]}
```

## Acceptance Criteria
- [ ] All field errors returned at once (not fail-fast)
- [ ] Field names use `json` tag names
- [ ] Custom decimal validator""",
    ),
    dict(
        title="[M6-B4] OpenAPI spec + Swagger UI",
        labels=["P2", "M6-polish", "backend"],
        milestone=M["M6"],
        body="""## Summary
OpenAPI 3.1 spec for all M0–M5 endpoints. Swagger UI at `/api/docs` in non-production.

## Acceptance Criteria
- [ ] All endpoints documented with request/response schemas
- [ ] Auth security scheme documented
- [ ] Spec validates with `openapi-spec-validator`
- [ ] Only served in `ENV != production`""",
    ),
]


def create_issue(issue):
    cmd = [
        "gh", "issue", "create",
        "--repo", REPO,
        "--title", issue["title"],
        "--body", issue["body"],
        "--milestone", str(issue["milestone"]),
    ]
    for label in issue["labels"]:
        cmd += ["--label", label]
    result = subprocess.run(cmd, capture_output=True, text=True)
    if result.returncode != 0:
        print(f"FAIL: {issue['title']}\n{result.stderr}", file=sys.stderr)
        return False
    url = result.stdout.strip()
    print(f"OK  {url}  {issue['title']}")
    return True


if __name__ == "__main__":
    ok = fail = 0
    for i, issue in enumerate(issues):
        success = create_issue(issue)
        if success:
            ok += 1
        else:
            fail += 1
        # small delay to avoid secondary rate limit
        if (i + 1) % 10 == 0:
            time.sleep(2)
    print(f"\nDone: {ok} created, {fail} failed")
