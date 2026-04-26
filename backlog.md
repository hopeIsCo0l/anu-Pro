# Anu Pro — Engineering Backlog

> Last updated: 2026-04-26
> Stack: Go 1.23 · chi · pgx/v5 · sqlc · goose · river · shopspring/decimal · Next.js · shadcn/ui · TanStack Query
> Module: `github.com/hopeIsCo0l/anu-pro`

---

## Legend

| Field | Values |
|---|---|
| Priority | P0 = blocking, P1 = core MVP, P2 = important, P3 = nice-to-have |
| Status | `todo` · `in-progress` · `done` · `blocked` |
| Estimate | Hours (1 engineer) |

---

## M0 — Foundations (Weeks 1–3)

### M0-A: Project Scaffold & Infrastructure

---

#### M0-A1 — Repo init, .gitignore, CLAUDE.md
**Status:** `done` | **Priority:** P0 | **Est:** 1h

**Description:**
Initialize git repo, set up .gitignore to exclude IDE and local Claude files, commit CLAUDE.md with full project context document.

**Acceptance Criteria:**
- [ ] `.gitignore` excludes `.idea/`, `.vscode/`, `.claude/`, `.env`, `*.exe`, `/bin/`, `/dist/`
- [ ] `CLAUDE.md` committed and readable
- [ ] Remote set to `github.com/hopeIsCo0l/anu-Pro`, default branch `main`
- [ ] `git push origin main` succeeds

**Dependencies:** none

---

#### M0-A2 — Go project directory scaffold
**Status:** `done` | **Priority:** P0 | **Est:** 2h

**Description:**
Create the full directory tree for the modular monolith. Each `internal/` sub-package gets a stub `.go` file with package declaration so the tree is navigable and `go build ./...` passes clean.

**Directory tree:**
```
cmd/api/main.go
cmd/worker/main.go
cmd/migrate/main.go
internal/platform/config/
internal/platform/db/
internal/platform/auth/
internal/platform/tenant/
internal/platform/storage/
internal/platform/httpserver/
internal/inventory/
internal/production/
internal/recipe/
internal/costing/
internal/audit/
migrations/public/
migrations/tenant/
deploy/
```

**Acceptance Criteria:**
- [ ] `go build ./...` exits 0 with no errors
- [ ] All directories exist as listed above
- [ ] `cmd/api/main.go` starts chi HTTP server, serves `GET /health` → `200 {"status":"ok"}`
- [ ] Server reads port from `$PORT` env var, defaults to `8080`
- [ ] Graceful shutdown on `SIGTERM`/`SIGINT` with 30s timeout

**Dependencies:** M0-A1

---

#### M0-A3 — pgx/v5 connection pool with tenant search_path switching
**Status:** `todo` | **Priority:** P0 | **Est:** 4h

**Description:**
Implement `internal/platform/db/db.go`. Create a `pgxpool.Pool` that switches `search_path` on every connection acquisition based on the current tenant slug. Reset `search_path` to `public` on connection release to prevent cross-tenant leaks.

This is the single most critical infrastructure piece. Get it exactly right.

**Implementation notes:**
- Use `pgxpool.Config.BeforeAcquire` to call `SET search_path TO tenant_<slug>, public` before handing the connection to the caller
- Use `pgxpool.Config.AfterRelease` to call `SET search_path TO public` when connection is returned to pool
- Tenant slug is passed via `context.Context` — define a `type ctxKey string` and `const tenantKey ctxKey = "tenant"`
- Provide `WithTenant(ctx, slug) context.Context` helper
- Provide `FromContext(ctx) (*pgxpool.Pool, error)` or accept pool + ctx in every query
- `Pool(cfg *config.Config) (*pgxpool.Pool, error)` constructor — parse `DATABASE_URL`, set sane pool limits (MaxConns=25, MinConns=2, HealthCheckPeriod=30s)
- Log pool events via `slog` at debug level

**Acceptance Criteria:**
- [ ] `db.New(cfg)` returns a connected pool or error
- [ ] `BeforeAcquire` sets `search_path TO tenant_<slug>, public` when tenant in context
- [ ] `BeforeAcquire` sets `search_path TO public` when no tenant in context
- [ ] `AfterRelease` always resets `search_path TO public`
- [ ] Unit test: two goroutines with different tenant contexts run concurrent queries; each sees its own schema rows and never the other's (requires real Postgres via testcontainers)
- [ ] Cross-tenant isolation test must always pass — this test is a P0 gate for every PR

**Dependencies:** M0-A7

---

#### M0-A4 — Goose migration scaffold + first public migration
**Status:** `todo` | **Priority:** P0 | **Est:** 4h

**Description:**
Set up `goose` as the migration tool. Migrations are split into two directories:
- `migrations/public/` — runs once, mutates the `public` schema (tenants, users, roles, refresh_tokens)
- `migrations/tenant/` — runs per-tenant schema on tenant creation

`cmd/migrate/main.go` reads `MIGRATION_DIR` and `MIGRATION_TARGET` env vars (`public` or `tenant:<slug>`) and runs goose up.

**First public migration (`0001_public_schema.sql`):**
```sql
-- tenants
CREATE TABLE tenants (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug        TEXT UNIQUE NOT NULL,
  name        TEXT NOT NULL,
  timezone    TEXT NOT NULL DEFAULT 'UTC',
  plan        TEXT NOT NULL DEFAULT 'pilot',
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  suspended_at TIMESTAMPTZ
);

-- users
CREATE TABLE users (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id    UUID NOT NULL REFERENCES tenants(id),
  email        TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  full_name    TEXT NOT NULL,
  role         TEXT NOT NULL DEFAULT 'operator',  -- owner|manager|operator|viewer
  is_active    BOOLEAN NOT NULL DEFAULT true,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_login_at TIMESTAMPTZ
);
CREATE INDEX users_tenant_id_idx ON users(tenant_id);
CREATE INDEX users_email_idx ON users(email);

-- refresh_tokens
CREATE TABLE refresh_tokens (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID NOT NULL REFERENCES users(id),
  token_hash  TEXT UNIQUE NOT NULL,
  expires_at  TIMESTAMPTZ NOT NULL,
  revoked_at  TIMESTAMPTZ,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  user_agent  TEXT,
  ip_addr     TEXT
);
CREATE INDEX refresh_tokens_user_id_idx ON refresh_tokens(user_id);

-- events (public-level audit, e.g. login, tenant creation)
CREATE TABLE events (
  id          BIGSERIAL,
  tenant_id   UUID REFERENCES tenants(id),
  actor_id    UUID,
  event_type  TEXT NOT NULL,
  payload     JSONB NOT NULL DEFAULT '{}',
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
) PARTITION BY RANGE (created_at);
CREATE TABLE events_2026_04 PARTITION OF events
  FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');
-- NOTE: future partitions created monthly by maintenance job
CREATE INDEX events_tenant_idx ON events(tenant_id, created_at DESC);
CREATE INDEX events_type_idx   ON events(event_type, created_at DESC);
```

**Acceptance Criteria:**
- [ ] `go run ./cmd/migrate up public` applies public migrations cleanly
- [ ] `go run ./cmd/migrate status public` shows applied migrations
- [ ] `go run ./cmd/migrate down public` rolls back correctly
- [ ] All four tables exist after up: `tenants`, `users`, `refresh_tokens`, `events`
- [ ] Events table is partitioned by month
- [ ] Migration idempotent: running `up` twice does not error
- [ ] Tests run migrations in testcontainer (no mocks)

**Dependencies:** M0-A2, M0-A3

---

#### M0-A5 — Docker Compose local dev environment
**Status:** `todo` | **Priority:** P0 | **Est:** 3h

**Description:**
`docker-compose up` must reproduce a full local dev environment: Postgres 16, MinIO (S3-compatible), the API server, and the worker. All services must be reachable from the host. Use `.env.example` to document all required env vars.

**`docker-compose.yml` services:**
- `postgres`: postgres:16-alpine, port 5432, persistent volume, healthcheck
- `minio`: minio/minio:latest, port 9000 (S3) + 9001 (console), persistent volume, auto-create bucket via `mc` init container
- `api`: build from Dockerfile, depends on postgres + minio, env_file: .env
- `worker`: same binary, `CMD=worker`, depends on postgres + minio
- `migrate`: one-shot service, runs migrations and exits

**`.env.example`:**
```
PORT=8080
ENV=development
DATABASE_URL=postgres://anu:anu@localhost:5432/anu?sslmode=disable
JWT_SECRET=dev-secret-change-in-prod
JWT_ACCESS_EXPIRY_MIN=15
JWT_REFRESH_EXPIRY_DAYS=30
STORAGE_ENDPOINT=http://localhost:9000
STORAGE_BUCKET=anu-dev
STORAGE_ACCESS_KEY=minioadmin
STORAGE_SECRET_KEY=minioadmin
STORAGE_REGION=us-east-1
LOG_LEVEL=debug
```

**Acceptance Criteria:**
- [ ] `docker compose up` starts all services with no manual steps
- [ ] `curl localhost:8080/health` returns `{"status":"ok"}`
- [ ] MinIO console accessible at `localhost:9001`
- [ ] Postgres accessible at `localhost:5432` with user `anu`, db `anu`
- [ ] `.env.example` documents every env var used anywhere in the codebase
- [ ] `docker compose down -v` tears down cleanly including volumes

**Dependencies:** M0-A2

---

#### M0-A6 — GitHub Actions CI/CD pipeline
**Status:** `todo` | **Priority:** P0 | **Est:** 5h

**Description:**
Two workflows:

**`ci.yml`** — runs on every PR and push to `main`:
1. `go vet ./...`
2. `golangci-lint run` (errcheck, staticcheck, gosimple, ineffassign, unused)
3. `go test ./... -race -count=1` (testcontainers spins real Postgres)
4. `go build ./cmd/api ./cmd/worker ./cmd/migrate`
5. Frontend lint + type-check (`pnpm lint && pnpm tsc --noEmit`) — once frontend scaffold exists

**`deploy.yml`** — runs on push to `main` after CI passes:
1. Build Docker image
2. Tag with `git SHA` + `latest`
3. Push to GitHub Container Registry (`ghcr.io`)
4. SSH to Oracle VM, pull new image, restart systemd service

**Acceptance Criteria:**
- [ ] PR cannot merge if CI fails
- [ ] `go test` runs with real Postgres via testcontainers (not mocked)
- [ ] Race detector enabled (`-race`)
- [ ] golangci-lint config at `.golangci.yml` with errcheck enabled
- [ ] Deploy workflow uses `secrets.ORACLE_SSH_KEY`, `secrets.ORACLE_HOST`
- [ ] Failed deploy sends a GitHub notification (built-in)
- [ ] Docker image is multi-stage: builder + minimal runtime (~30MB final image)

**Dependencies:** M0-A5

---

#### M0-A7 — envconfig-based Config struct
**Status:** `done` | **Priority:** P0 | **Est:** 1h

**Description:**
`internal/platform/config/config.go` — all configuration from env vars via `github.com/kelseyhightower/envconfig`. No platform-specific config APIs. No config files.

**Acceptance Criteria:**
- [ ] All fields have explicit `envconfig` tags
- [ ] Required fields (DATABASE_URL, JWT_SECRET, storage keys) fail fast on missing
- [ ] `config.Load()` returns `(*Config, error)`
- [ ] Defaults are set for PORT, ENV, LOG_LEVEL, expiry values
- [ ] Test: `Load()` returns error when `DATABASE_URL` missing
- [ ] Test: `Load()` populates all fields correctly from env

**Dependencies:** M0-A2

---

#### M0-A8 — Structured logging setup (slog)
**Status:** `todo` | **Priority:** P0 | **Est:** 2h

**Description:**
Centralize slog setup in `internal/platform/logger/logger.go`. Logger is configured once at startup from `cfg.LogLevel` and `cfg.Env`. In production: JSON handler to stdout. In development: text handler for readability. Pass logger via context or inject at handler construction — do not use global `slog.SetDefault` beyond `main.go`.

Add a chi middleware that logs every request: method, path, status, latency, request_id, tenant_id (if present).

**Acceptance Criteria:**
- [ ] JSON output in `ENV=production`
- [ ] Human-readable output in `ENV=development`
- [ ] Every HTTP request logged with: method, path, status, latency_ms, request_id
- [ ] `tenant_id` included in log line when present in context
- [ ] `log_level=debug` shows SQL queries (pgx trace handler)
- [ ] No `fmt.Println` or `log.Printf` anywhere outside tests

**Dependencies:** M0-A7

---

#### M0-A9 — Oracle Cloud VM provisioning + Caddy + systemd
**Status:** `todo` | **Priority:** P1 | **Est:** 6h

**Description:**
Provision Oracle Cloud Always Free VM (ARM, 4 OCPU, 24GB RAM). Set up:
- Ubuntu 22.04, ufw rules (22, 80, 443 only)
- Postgres 16 (self-hosted, data dir on block volume)
- Caddy 2 as reverse proxy (auto HTTPS via Let's Encrypt)
- systemd unit files for `anu-api` and `anu-worker`
- Automated deploy script: pull image from GHCR, stop, start, health-check

Store provisioning scripts in `deploy/oracle/`. Document in `deploy/oracle/README.md`.

**Acceptance Criteria:**
- [ ] `deploy/oracle/provision.sh` idempotent — safe to re-run
- [ ] `deploy/oracle/Caddyfile` routes `api.anupro.app` → `localhost:8080`
- [ ] `deploy/oracle/anu-api.service` and `anu-worker.service` systemd units
- [ ] `deploy/oracle/deploy.sh` pulls new image, zero-downtime restart
- [ ] TLS cert auto-renewed by Caddy
- [ ] Postgres WAL archiving enabled, ready for backup (M0-A10)
- [ ] `systemctl status anu-api` shows `active (running)`

**Dependencies:** M0-A6

---

#### M0-A10 — Automated daily backups to Cloudflare R2 + restore test
**Status:** `todo` | **Priority:** P0 | **Est:** 5h

**Description:**
Backups are non-negotiable. Implement before any business logic.

**Backup script (`deploy/oracle/backup.sh`):**
- `pg_dump` with `--format=custom` (compressed, supports partial restore)
- Upload to R2 bucket `anu-backups/YYYY/MM/DD/anu-TIMESTAMP.dump`
- Retain last 30 daily backups, delete older
- Write backup manifest JSON (timestamp, size, checksum)
- Alert via email on failure (use `msmtp` or curl to a webhook)

**Restore test script (`deploy/oracle/test-restore.sh`):**
- Download latest backup from R2
- Restore to a temp Postgres db (`anu_restore_test`)
- Run smoke queries (count rows in tenants, users, events)
- Drop temp db
- Log result with timestamp

**Schedule:**
- Backup: daily 02:00 UTC via systemd timer
- Restore test: weekly Sunday 03:00 UTC via systemd timer

**Acceptance Criteria:**
- [ ] Daily backup runs and uploads to R2 without manual intervention
- [ ] Backup file verifiable with `pg_restore --list`
- [ ] Weekly restore test passes (smoke queries return > 0 rows after seeding)
- [ ] Backup failure sends email alert within 5 minutes
- [ ] Restore test failure sends email alert
- [ ] 30-day retention enforced (older dumps auto-deleted)
- [ ] `deploy/oracle/README.md` documents how to manually trigger a restore

**Dependencies:** M0-A9

---

#### M0-A11 — Cross-tenant isolation integration test
**Status:** `todo` | **Priority:** P0 | **Est:** 3h

**Description:**
This test is a permanent P0 gate. It must run on every PR and never be skipped.

Create `internal/platform/db/isolation_test.go`:
1. Spin up Postgres via testcontainers
2. Create two tenants (`tenant_alpha`, `tenant_beta`) with separate schemas
3. Run first public migration
4. Run first tenant migration on both schemas
5. Insert a row into `tenant_alpha.items` (any table from tenant schema)
6. With `tenant_beta` context, query `items` — assert zero rows returned
7. With `tenant_alpha` context, query `items` — assert one row returned
8. Assert no errors, no panics, no cross-contamination

**Acceptance Criteria:**
- [ ] Test is in `package db_test` (black-box)
- [ ] Uses real Postgres (testcontainers), no mocks
- [ ] Covers concurrent access (two goroutines, interleaved queries)
- [ ] Test name: `TestCrossTenantIsolation` — do not rename
- [ ] Fails loudly if `search_path` is not being set correctly

**Dependencies:** M0-A3, M0-A4

---

### M0-B: Authentication

---

#### M0-B1 — Argon2id password hashing
**Status:** `todo` | **Priority:** P0 | **Est:** 2h

**Description:**
`internal/platform/auth/password.go` — hash and verify passwords using Argon2id (not bcrypt). Use `golang.org/x/crypto/argon2`.

Parameters (OWASP recommended minimums):
- Memory: 64MB (`64 * 1024` KiB)
- Iterations: 3
- Parallelism: 2
- Salt: 16 bytes random
- Key length: 32 bytes
- Encode as: `$argon2id$v=19$m=65536,t=3,p=2$<salt_b64>$<hash_b64>`

**Acceptance Criteria:**
- [ ] `HashPassword(plain string) (string, error)` — returns encoded hash string
- [ ] `VerifyPassword(plain, encoded string) (bool, error)` — constant-time compare
- [ ] Hash output is different each call (random salt)
- [ ] Verify returns `false` for wrong password, no error
- [ ] Verify returns `false` for empty password, no error
- [ ] Benchmark: hash < 500ms on development hardware
- [ ] Test: hash then verify round-trip passes
- [ ] Test: tampered hash returns false, not error

**Dependencies:** M0-A2

---

#### M0-B2 — JWT issuance (access + refresh tokens)
**Status:** `todo` | **Priority:** P0 | **Est:** 3h

**Description:**
`internal/platform/auth/jwt.go` — issue and validate JWTs using `github.com/golang-jwt/jwt/v5`.

**Access token claims:**
```json
{
  "sub": "<user_id>",
  "tid": "<tenant_id>",
  "tslug": "<tenant_slug>",
  "role": "operator",
  "exp": <unix>,
  "iat": <unix>,
  "jti": "<uuid>"
}
```

**Refresh token:**
- Random 32-byte token, stored as SHA-256 hash in `refresh_tokens` table
- Never put refresh token in JWT — it's an opaque token in a `HttpOnly` cookie

**Functions:**
- `IssueAccessToken(cfg, claims) (string, error)`
- `ValidateAccessToken(cfg, tokenStr) (*Claims, error)`
- `IssueRefreshToken() (plain string, hash string, error)` — returns both for cookie + DB

**Acceptance Criteria:**
- [ ] Access token signed with HS256 using `JWT_SECRET`
- [ ] Expired token returns `ErrTokenExpired` (not a generic error)
- [ ] Tampered token returns `ErrTokenInvalid`
- [ ] `jti` claim is unique UUID per token
- [ ] Refresh token is 32 random bytes, hex-encoded
- [ ] Refresh token hash is SHA-256 of plain token
- [ ] Tests cover: valid token, expired, tampered signature, wrong algorithm

**Dependencies:** M0-A7

---

#### M0-B3 — Refresh token rotation
**Status:** `todo` | **Priority:** P0 | **Est:** 3h

**Description:**
`internal/platform/auth/refresh.go` — stateful refresh token rotation.

**Flow:**
1. Client sends refresh token (HttpOnly cookie)
2. Server hashes it, looks up `refresh_tokens` where `token_hash = ? AND revoked_at IS NULL AND expires_at > now()`
3. If found: revoke old token, issue new access token + new refresh token, return both
4. If not found / expired: return 401
5. Detect reuse: if token already revoked, revoke ALL tokens for that user (session hijack detection)

**Acceptance Criteria:**
- [ ] Successful refresh: old token revoked, new tokens issued in single transaction
- [ ] Expired token: 401, no new tokens issued
- [ ] Reuse of revoked token: all user tokens revoked, 401
- [ ] Refresh token stored as hash, never plaintext
- [ ] `HttpOnly; Secure; SameSite=Strict` cookie attributes on refresh token
- [ ] Test: full rotation flow (3 rounds)
- [ ] Test: reuse detection triggers full revocation

**Dependencies:** M0-B2, M0-A4

---

#### M0-B4 — Signup endpoint
**Status:** `todo` | **Priority:** P0 | **Est:** 4h

**Description:**
`POST /api/v1/auth/signup`

Creates a new tenant + first owner user in a single transaction. Also runs tenant schema migrations.

**Request:**
```json
{
  "tenant_name": "Sunrise Dairy",
  "tenant_slug": "sunrise-dairy",
  "full_name": "Priya Sharma",
  "email": "priya@sunrise.co",
  "password": "correcthorsebatterystaple",
  "timezone": "Asia/Kolkata"
}
```

**Response:** `201 Created`
```json
{
  "access_token": "eyJ...",
  "user": { "id": "...", "email": "...", "role": "owner" },
  "tenant": { "id": "...", "slug": "sunrise-dairy", "name": "Sunrise Dairy" }
}
```
Refresh token in `HttpOnly` cookie.

**Validation:**
- `tenant_slug`: lowercase, alphanumeric + hyphens, 3-50 chars, unique
- `email`: valid format, unique across all tenants
- `password`: min 12 chars
- `full_name`: 1-100 chars

**Acceptance Criteria:**
- [ ] Creates tenant row in `public.tenants`
- [ ] Creates schema `tenant_<slug>`
- [ ] Runs all tenant migrations on new schema
- [ ] Creates user row in `public.users` with role `owner`
- [ ] Hashes password with Argon2id before storing
- [ ] Emits `user.signed_up` event in `public.events`
- [ ] Returns 409 if slug already taken
- [ ] Returns 409 if email already registered
- [ ] Returns 422 with field errors for validation failures
- [ ] Entire operation is one DB transaction (rollback on any failure)
- [ ] Test: happy path end-to-end
- [ ] Test: duplicate slug returns 409
- [ ] Test: weak password returns 422

**Dependencies:** M0-B1, M0-B2, M0-A4, M0-C1

---

#### M0-B5 — Login endpoint
**Status:** `todo` | **Priority:** P0 | **Est:** 3h

**Description:**
`POST /api/v1/auth/login`

**Request:**
```json
{
  "email": "priya@sunrise.co",
  "password": "correcthorsebatterystaple"
}
```

**Response:** `200 OK`
```json
{
  "access_token": "eyJ...",
  "user": { "id": "...", "email": "...", "role": "owner", "full_name": "..." },
  "tenant": { "id": "...", "slug": "sunrise-dairy", "name": "Sunrise Dairy" }
}
```
Refresh token in `HttpOnly` cookie.

**Security:**
- Constant-time compare for password (via Argon2id verify)
- Same error message for "user not found" and "wrong password" (`invalid credentials`)
- Update `last_login_at` on success
- Rate limit: max 5 failed attempts per email per 15 minutes (in-memory or Postgres-backed counter)

**Acceptance Criteria:**
- [ ] Returns 200 + tokens on correct credentials
- [ ] Returns 401 `invalid credentials` for wrong password
- [ ] Returns 401 `invalid credentials` for unknown email (no user enumeration)
- [ ] Returns 401 for inactive user
- [ ] Updates `last_login_at` on success
- [ ] Emits `user.logged_in` event
- [ ] Rate limiting returns 429 after 5 failures
- [ ] Test: correct credentials
- [ ] Test: wrong password
- [ ] Test: unknown email
- [ ] Test: inactive user

**Dependencies:** M0-B1, M0-B2, M0-B3

---

#### M0-B6 — Logout + token revocation
**Status:** `todo` | **Priority:** P1 | **Est:** 2h

**Description:**
`POST /api/v1/auth/logout` — requires valid access token.

Revokes the refresh token sent in cookie. Clears the cookie. Does not invalidate the access token (short-lived by design — 15 min).

`POST /api/v1/auth/logout-all` — revokes ALL refresh tokens for the current user.

**Acceptance Criteria:**
- [ ] `POST /logout` sets `revoked_at = now()` on the refresh token
- [ ] `POST /logout` clears the `HttpOnly` cookie
- [ ] `POST /logout-all` revokes all non-revoked tokens for user
- [ ] Both return 200 even if token already revoked (idempotent)
- [ ] Emits `user.logged_out` event

**Dependencies:** M0-B3

---

#### M0-B7 — Auth middleware (JWT validation + tenant resolution)
**Status:** `todo` | **Priority:** P0 | **Est:** 3h

**Description:**
`internal/platform/auth/middleware.go`

Chi middleware that:
1. Extracts `Authorization: Bearer <token>` header
2. Validates JWT signature + expiry
3. Loads tenant from `tslug` claim (verify tenant exists and is not suspended)
4. Injects `user_id`, `tenant_id`, `tenant_slug`, `role` into context
5. Sets `search_path` context key for pgx pool (M0-A3)

Also provide `RequireRole(roles ...string)` middleware that checks the role claim.

**Acceptance Criteria:**
- [ ] Missing/malformed token → 401 `{"error":"unauthorized"}`
- [ ] Expired token → 401 `{"error":"token_expired"}`
- [ ] Suspended tenant → 403 `{"error":"tenant_suspended"}`
- [ ] Valid token injects all claims into context
- [ ] `RequireRole("owner", "manager")` returns 403 for operator
- [ ] Public routes (`/health`, `/api/v1/auth/*`) bypass middleware
- [ ] Test: valid token passes through
- [ ] Test: expired token rejected
- [ ] Test: wrong role returns 403

**Dependencies:** M0-B2, M0-A3

---

#### M0-B8 — Role-based access control (RBAC)
**Status:** `todo` | **Priority:** P1 | **Est:** 3h

**Description:**
Four roles with explicit permission matrix:

| Action | owner | manager | operator | viewer |
|---|---|---|---|---|
| Manage users | ✓ | | | |
| Manage items/recipes | ✓ | ✓ | | |
| Create production jobs | ✓ | ✓ | ✓ | |
| Complete/cancel jobs | ✓ | ✓ | ✓ | |
| View all data | ✓ | ✓ | ✓ | ✓ |
| Export / import data | ✓ | ✓ | | |
| Manage tenant settings | ✓ | | | |

Implement as `RequireRole` middleware and document the matrix. Store role in JWT claim; no per-request DB lookup for role.

**Acceptance Criteria:**
- [ ] `RequireRole` middleware usable on any route
- [ ] Role comes from JWT claim (no DB query per request)
- [ ] Viewer can GET any resource, cannot POST/PUT/DELETE
- [ ] Operator can manage production jobs but not items or users
- [ ] Tests cover boundary cases (operator tries to create item → 403)

**Dependencies:** M0-B7

---

### M0-C: Tenant Management

---

#### M0-C1 — Tenant schema provisioning
**Status:** `todo` | **Priority:** P0 | **Est:** 4h

**Description:**
`internal/platform/tenant/provisioner.go`

When a new tenant is created (via signup or admin endpoint):
1. Create Postgres schema `tenant_<slug>`
2. Run all migrations from `migrations/tenant/` against that schema
3. Record the schema creation in `public.tenants`

This must be atomic: if migration fails, drop the schema and roll back the tenant row.

**First tenant migration (`migrations/tenant/0001_tenant_schema.sql`):**
```sql
-- items catalog
CREATE TABLE items (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  sku         TEXT UNIQUE NOT NULL,
  name        TEXT NOT NULL,
  description TEXT,
  unit        TEXT NOT NULL,        -- kg, L, each, etc.
  category    TEXT,
  attributes  JSONB NOT NULL DEFAULT '{}',
  is_active   BOOLEAN NOT NULL DEFAULT true,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- locations (warehouse > zone > bin)
CREATE TABLE locations (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  code        TEXT UNIQUE NOT NULL,
  name        TEXT NOT NULL,
  parent_id   UUID REFERENCES locations(id),
  type        TEXT NOT NULL DEFAULT 'zone',  -- warehouse|zone|bin
  is_active   BOOLEAN NOT NULL DEFAULT true
);

-- lots
CREATE TABLE lots (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  item_id       UUID NOT NULL REFERENCES items(id),
  lot_number    TEXT NOT NULL,
  supplier_ref  TEXT,
  received_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at    TIMESTAMPTZ,
  status        TEXT NOT NULL DEFAULT 'active',  -- active|quarantine|consumed|expired
  attributes    JSONB NOT NULL DEFAULT '{}',
  UNIQUE(item_id, lot_number)
);
CREATE INDEX lots_item_id_idx ON lots(item_id);
CREATE INDEX lots_expires_at_idx ON lots(expires_at) WHERE expires_at IS NOT NULL;

-- stock_movements (append-only ledger)
CREATE TABLE stock_movements (
  id              BIGSERIAL,
  lot_id          UUID NOT NULL REFERENCES lots(id),
  location_id     UUID NOT NULL REFERENCES locations(id),
  quantity        NUMERIC(18,6) NOT NULL,  -- positive=in, negative=out
  movement_type   TEXT NOT NULL,  -- receipt|production_in|production_out|transfer_in|transfer_out|adjustment|waste
  reference_id    UUID,           -- production_job_id or transfer_id
  idempotency_key TEXT UNIQUE NOT NULL,
  notes           TEXT,
  actor_id        UUID NOT NULL,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
) PARTITION BY RANGE (created_at);
CREATE TABLE stock_movements_2026_04 PARTITION OF stock_movements
  FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');
CREATE INDEX sm_lot_id_idx      ON stock_movements(lot_id, created_at DESC);
CREATE INDEX sm_location_id_idx ON stock_movements(location_id, created_at DESC);
CREATE INDEX sm_type_idx        ON stock_movements(movement_type, created_at DESC);

-- current_stock view (never a stored column)
CREATE VIEW current_stock AS
  SELECT
    lot_id,
    location_id,
    SUM(quantity) AS quantity
  FROM stock_movements
  GROUP BY lot_id, location_id
  HAVING SUM(quantity) != 0;

-- tenant-level events (partitioned, same as public)
CREATE TABLE events (
  id          BIGSERIAL,
  actor_id    UUID,
  event_type  TEXT NOT NULL,
  entity_type TEXT,
  entity_id   UUID,
  payload     JSONB NOT NULL DEFAULT '{}',
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
) PARTITION BY RANGE (created_at);
CREATE TABLE events_2026_04 PARTITION OF events
  FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');
```

**Acceptance Criteria:**
- [ ] `ProvisionTenant(ctx, db, slug, name, timezone)` creates schema + runs migrations atomically
- [ ] Failed migration drops schema and returns error
- [ ] Idempotent: calling twice returns error `tenant already exists`
- [ ] All tables from first migration exist after provisioning
- [ ] `current_stock` view exists and returns correct SUM
- [ ] Cross-tenant isolation test passes after provisioning two tenants

**Dependencies:** M0-A4

---

#### M0-C2 — Tenant migration runner (per-tenant schema upgrades)
**Status:** `todo` | **Priority:** P0 | **Est:** 3h

**Description:**
When new tenant migrations are added (e.g., `0002_recipes.sql`), they must be applied to ALL existing tenant schemas. `cmd/migrate` handles this:

```
go run ./cmd/migrate up tenant:sunrise-dairy    # single tenant
go run ./cmd/migrate up tenant:*                # all tenants
go run ./cmd/migrate status tenant:*            # show status per tenant
```

Also: on new tenant creation, run ALL tenant migrations (not just the latest).

**Acceptance Criteria:**
- [ ] `tenant:*` flag iterates all rows in `public.tenants` and applies migrations to each schema
- [ ] Goose version table per schema: `tenant_<slug>.goose_db_version`
- [ ] Reports which tenants are behind (not on latest version)
- [ ] Dry-run mode: `--dry-run` prints SQL without executing
- [ ] Test: 3 tenants at different versions, upgrade all to latest, verify status

**Dependencies:** M0-C1

---

#### M0-C3 — Row-Level Security (defense in depth)
**Status:** `todo` | **Priority:** P1 | **Est:** 3h

**Description:**
Schema-per-tenant is the primary isolation mechanism. RLS is defense in depth.

Enable RLS on all tenant tables via migration. Policy: only connections with `search_path` set to the current tenant's schema can see rows. Since tenant schemas are separate, RLS here protects against a programming error where `search_path` is not set.

In practice, this means:
- `ALTER TABLE items ENABLE ROW LEVEL SECURITY`
- Policy uses `current_setting('app.tenant_slug')` which is set in `BeforeAcquire`
- Add `SET app.tenant_slug = '<slug>'` to the `BeforeAcquire` hook alongside `search_path`

**Acceptance Criteria:**
- [ ] RLS enabled on: items, lots, locations, stock_movements, events (tenant schema)
- [ ] Direct `psql` query with wrong `app.tenant_slug` returns zero rows
- [ ] Application queries unaffected (correct slug in context)
- [ ] RLS bypass test: simulate misconfigured `search_path`, verify RLS still blocks data

**Dependencies:** M0-A3, M0-C1

---

## M1 — Inventory (Weeks 4–6)

### M1-A: Items Catalog

---

#### M1-A1 — Items catalog migration
**Status:** `todo` | **Priority:** P0 | **Est:** 1h

**Description:**
Migration already included in M0-C1 (`0001_tenant_schema.sql`). This ticket covers adding indexes and constraints that were deferred.

Additional migration `0002_items_indexes.sql`:
```sql
CREATE INDEX items_sku_idx      ON items(sku);
CREATE INDEX items_name_idx     ON items USING gin(to_tsvector('english', name));
CREATE INDEX items_category_idx ON items(category) WHERE category IS NOT NULL;
CREATE INDEX items_active_idx   ON items(is_active) WHERE is_active = true;
```

**Acceptance Criteria:**
- [ ] Full-text search index on `name` (gin/tsvector)
- [ ] Category index with partial filter for non-null
- [ ] Migration applies cleanly

**Dependencies:** M0-C1

---

#### M1-A2 — Item CRUD endpoints
**Status:** `todo` | **Priority:** P0 | **Est:** 5h

**Description:**
RESTful item management. All endpoints require auth middleware + appropriate role.

```
POST   /api/v1/items           → 201 (manager+)
GET    /api/v1/items           → 200 (viewer+)
GET    /api/v1/items/:id       → 200 (viewer+)
PUT    /api/v1/items/:id       → 200 (manager+)
DELETE /api/v1/items/:id       → 204 soft-delete (manager+) — sets is_active=false
```

Use `sqlc` for all queries. Define SQL in `internal/inventory/queries/items.sql`, generate with `sqlc generate`.

**Request body (POST/PUT):**
```json
{
  "sku": "WHL-001",
  "name": "Whole Milk",
  "description": "Pasteurized whole milk",
  "unit": "L",
  "category": "dairy-input",
  "attributes": { "fat_pct": 3.5, "supplier": "Farm Co" }
}
```

**Acceptance Criteria:**
- [ ] `POST` creates item, returns 201 with full item body
- [ ] `GET` list returns paginated items (cursor-based pagination)
- [ ] `GET :id` returns 404 if not found or inactive
- [ ] `PUT` updates only provided fields (partial update)
- [ ] `DELETE` soft-deletes (sets `is_active=false`), does not destroy ledger history
- [ ] SKU unique constraint returns 409 with descriptive error
- [ ] `attributes` JSONB stored and returned correctly
- [ ] `updated_at` updated on every PUT
- [ ] sqlc generated queries (no raw string SQL in Go handlers)
- [ ] Tests: happy path CRUD, 404, 409 duplicate SKU

**Dependencies:** M0-A3, M0-B7, M0-C1, M1-A1

---

#### M1-A3 — Item search and pagination
**Status:** `todo` | **Priority:** P1 | **Est:** 3h

**Description:**
Extend `GET /api/v1/items` with:
- Full-text search: `?q=whole+milk` (uses gin/tsvector index)
- Category filter: `?category=dairy-input`
- Active filter: `?active=true` (default) / `?active=false`
- Cursor-based pagination: `?cursor=<opaque_cursor>&limit=50` (default limit 50, max 200)
- Sort: `?sort=name|sku|created_at` + `?dir=asc|desc`

Cursor encodes `(created_at, id)` as base64 JSON (stable sort, no offset).

**Acceptance Criteria:**
- [ ] `?q=milk` returns items matching "milk" in name (FTS)
- [ ] `?category=dairy-input` filters correctly
- [ ] Pagination cursor is opaque (clients don't parse it)
- [ ] `next_cursor` is null on last page
- [ ] Over-limit request capped at 200
- [ ] Empty result returns `{"items": [], "next_cursor": null}`
- [ ] Sort works for all three fields in both directions

**Dependencies:** M1-A2

---

#### M1-A4 — Units of measure (UoM) + conversions
**Status:** `todo` | **Priority:** P1 | **Est:** 4h

**Description:**
`migrations/tenant/0003_uom.sql`:
```sql
CREATE TABLE units_of_measure (
  code        TEXT PRIMARY KEY,  -- kg, g, L, mL, each
  name        TEXT NOT NULL,
  base_unit   TEXT REFERENCES units_of_measure(code),
  to_base     NUMERIC(18,6),     -- multiplier to convert to base unit
  unit_type   TEXT NOT NULL      -- mass|volume|count|length
);

CREATE TABLE item_uom_conversions (
  item_id         UUID REFERENCES items(id),
  from_unit       TEXT REFERENCES units_of_measure(code),
  to_unit         TEXT REFERENCES units_of_measure(code),
  factor          NUMERIC(18,6) NOT NULL,
  PRIMARY KEY (item_id, from_unit, to_unit)
);
```

Seed standard units: kg, g, mg, t, L, mL, each, box, case.

`internal/inventory/uom/converter.go` — pure function:
```go
func Convert(qty decimal.Decimal, from, to string, conversions []ItemConversion) (decimal.Decimal, error)
```

**Acceptance Criteria:**
- [ ] Standard units seeded in migration
- [ ] `Convert(1, "kg", "g", nil)` = 1000 (uses base unit chain)
- [ ] Item-specific conversions override standard (e.g., "1 case = 24 bottles")
- [ ] Returns error for unknown or incompatible unit types
- [ ] Never uses float64 (shopspring/decimal throughout)
- [ ] Unit tests: kg→g, L→mL, item-specific case→each

**Dependencies:** M1-A1

---

### M1-B: Lots

---

#### M1-B1 — Lot CRUD endpoints
**Status:** `todo` | **Priority:** P0 | **Est:** 5h

**Description:**
Lots are the core traceability unit. Every stock movement references a lot.

```
POST   /api/v1/lots           → 201 (operator+)
GET    /api/v1/lots           → 200 (viewer+)
GET    /api/v1/lots/:id       → 200 (viewer+)
PUT    /api/v1/lots/:id       → 200 (manager+) — restricted fields only
GET    /api/v1/items/:id/lots → 200 lots for an item (viewer+)
```

**POST request:**
```json
{
  "item_id": "...",
  "lot_number": "LOT-2026-001",
  "supplier_ref": "PO-4521",
  "expires_at": "2026-10-01T00:00:00Z",
  "attributes": { "cert_of_analysis": "https://..." }
}
```

**Acceptance Criteria:**
- [ ] Lot number unique per item (not globally)
- [ ] `expires_at` optional
- [ ] Status starts as `active`
- [ ] `PUT` allows updating `expires_at`, `attributes`, `status` only
- [ ] List endpoint supports filter by `item_id`, `status`, `expiring_before` date
- [ ] 404 for unknown lot
- [ ] Tests: happy path, duplicate lot_number per item → 409

**Dependencies:** M0-B7, M0-C1

---

#### M1-B2 — Lot status transitions
**Status:** `todo` | **Priority:** P1 | **Est:** 2h

**Description:**
Valid transitions:
- `active` → `quarantine` (manual or auto on expiry)
- `quarantine` → `active` (QC clearance)
- `active` → `consumed` (when stock_movements sum = 0)
- `active` | `quarantine` → `expired` (cron job checks `expires_at < now()`)

`PATCH /api/v1/lots/:id/status` — explicit status change with reason.

Background job (river) runs hourly: mark lots as `expired` where `expires_at < now() AND status = 'active'`, emit `lot.expired` event.

**Acceptance Criteria:**
- [ ] Invalid transitions return 422 with message
- [ ] `expired` status auto-set by background job
- [ ] `lot.expired` event written in same transaction
- [ ] Cannot move stock out of `quarantine` lot (422)
- [ ] Test: status machine covers all transitions

**Dependencies:** M1-B1

---

### M1-C: Locations

---

#### M1-C1 — Location CRUD endpoints
**Status:** `todo` | **Priority:** P1 | **Est:** 3h

**Description:**
Hierarchical locations: `warehouse > zone > bin`. Self-referential `parent_id`.

```
POST /api/v1/locations        → 201 (manager+)
GET  /api/v1/locations        → 200 tree or flat list
GET  /api/v1/locations/:id    → 200
PUT  /api/v1/locations/:id    → 200 (manager+)
```

**GET** supports `?format=tree` (nested JSON) or `?format=flat` (default).

**Acceptance Criteria:**
- [ ] Location hierarchy depth unlimited (self-ref FK)
- [ ] Cannot set `parent_id` to create a cycle (check in application layer)
- [ ] `?format=tree` returns properly nested structure
- [ ] Deactivating a location deactivates all children
- [ ] Cannot deactivate if location has current stock (`current_stock` view check)

**Dependencies:** M0-C1

---

### M1-D: Stock Ledger (Core)

---

#### M1-D1 — Receipt movement endpoint
**Status:** `todo` | **Priority:** P0 | **Est:** 4h

**Description:**
`POST /api/v1/stock/receive` — record incoming stock.

**Request:**
```json
{
  "lot_id": "...",
  "location_id": "...",
  "quantity": "500.000",
  "notes": "PO-4521 delivery",
  "idempotency_key": "recv-po4521-20260426"
}
```

Writes a `stock_movements` row with `movement_type=receipt`, `quantity` positive.
Emits `stock.received` event in same transaction.

**Acceptance Criteria:**
- [ ] Quantity must be positive (422 otherwise)
- [ ] `idempotency_key` unique — duplicate returns 200 with original response (not 409)
- [ ] Quantity stored as `NUMERIC(18,6)` via shopspring/decimal (never float64)
- [ ] `current_stock` view reflects new balance immediately
- [ ] `stock.received` event written in same transaction
- [ ] 404 if lot_id or location_id not found
- [ ] Tests: happy path, duplicate idempotency key, negative quantity

**Dependencies:** M0-B7, M0-C1

---

#### M1-D2 — Adjustment movement endpoint
**Status:** `todo` | **Priority:** P1 | **Est:** 3h

**Description:**
`POST /api/v1/stock/adjust` — corrections, waste recording, cycle count reconciliation.

**Request:**
```json
{
  "lot_id": "...",
  "location_id": "...",
  "quantity": "-12.5",
  "movement_type": "waste",
  "reason": "spillage during transfer",
  "idempotency_key": "adj-20260426-001"
}
```

`movement_type` must be one of: `adjustment`, `waste`, `cycle_count`.
Quantity can be positive or negative (corrections can add or remove stock).

Cannot result in negative balance for the lot+location (422 with current balance in error body).

**Acceptance Criteria:**
- [ ] Blocks if resulting balance < 0
- [ ] Current balance included in 422 error: `{"error":"insufficient_stock","current_qty":"10.5"}`
- [ ] Uses `SELECT FOR UPDATE` on aggregated balance before inserting movement
- [ ] `stock.adjusted` event written in same transaction
- [ ] Tests: valid adjustment, would-go-negative rejection, idempotency

**Dependencies:** M1-D1

---

#### M1-D3 — Transfer movement endpoint
**Status:** `todo` | **Priority:** P1 | **Est:** 3h

**Description:**
`POST /api/v1/stock/transfer` — move stock between locations.

**Request:**
```json
{
  "lot_id": "...",
  "from_location_id": "...",
  "to_location_id": "...",
  "quantity": "50.0",
  "idempotency_key": "xfer-20260426-001"
}
```

Writes TWO movements atomically:
1. `transfer_out` at `from_location`, quantity negative
2. `transfer_in` at `to_location`, quantity positive

Both share the same `reference_id` (a transfer UUID).

**Acceptance Criteria:**
- [ ] Both movements written in one transaction
- [ ] `from_location` and `to_location` must differ (422 otherwise)
- [ ] Blocks if `from_location` would go negative
- [ ] `reference_id` same UUID on both movements
- [ ] `stock.transferred` event written in same transaction
- [ ] Tests: happy path, same location → 422, insufficient stock → 422

**Dependencies:** M1-D2

---

#### M1-D4 — Stock query endpoints
**Status:** `todo` | **Priority:** P1 | **Est:** 3h

**Description:**
```
GET /api/v1/stock                          current stock (all items, all locations)
GET /api/v1/stock?item_id=...             filter by item
GET /api/v1/stock?location_id=...         filter by location
GET /api/v1/stock/movements               movement history
GET /api/v1/stock/movements?lot_id=...    movements for a lot
GET /api/v1/stock/movements?item_id=...   movements for all lots of an item
```

`/stock` queries the `current_stock` view (materialized on each request — no stored balance column).

**Response (current stock):**
```json
{
  "items": [
    {
      "item_id": "...",
      "item_name": "Whole Milk",
      "sku": "WHL-001",
      "unit": "L",
      "total_qty": "4500.000",
      "lots": [
        { "lot_id": "...", "lot_number": "LOT-001", "location": "Cold Store", "qty": "4500.000", "expires_at": "..." }
      ]
    }
  ]
}
```

**Acceptance Criteria:**
- [ ] `/stock` joins `current_stock` view with items + lots + locations
- [ ] Zero-balance lots excluded from response (HAVING clause handles this)
- [ ] Movements history returns most-recent-first, paginated
- [ ] Movements include: type, qty, lot, location, actor, timestamp, notes
- [ ] `?from=2026-01-01&to=2026-04-30` date range filter on movements

**Dependencies:** M0-C1, M1-D1

---

#### M1-D5 — Low-stock alert system
**Status:** `todo` | **Priority:** P2 | **Est:** 3h

**Description:**
Items have an optional `min_stock_qty` threshold (add to items table via new migration).

River background job runs every 15 minutes: compare `current_stock` view totals against `min_stock_qty`. For items below threshold, emit `stock.low` event if one hasn't been emitted in the last 6 hours (debounce).

**Acceptance Criteria:**
- [ ] `items.min_stock_qty NUMERIC(18,6)` column added via migration
- [ ] River job `CheckLowStockJob` registered and runs every 15 minutes
- [ ] `stock.low` event payload includes: item_id, item_name, current_qty, threshold_qty
- [ ] Debounce: no duplicate alerts within 6 hours per item
- [ ] Test: item below threshold → event emitted; second run within 6h → no duplicate event

**Dependencies:** M1-D4

---

### M1-E: CSV Import

---

#### M1-E1 — CSV import: items
**Status:** `todo` | **Priority:** P1 | **Est:** 4h

**Description:**
`POST /api/v1/import/items` — multipart file upload.

**CSV format:**
```csv
sku,name,unit,category,description,attributes_json
WHL-001,Whole Milk,L,dairy-input,"Pasteurized whole milk","{""fat_pct"":3.5}"
```

Process:
1. Parse CSV, validate each row
2. Upsert items (insert or update by SKU)
3. Return import report: rows_processed, rows_created, rows_updated, rows_failed, errors

**Acceptance Criteria:**
- [ ] Max file size 10MB
- [ ] UTF-8 encoding required
- [ ] Row-level errors collected (not fail-fast): e.g., "row 5: invalid unit 'pounds'"
- [ ] Import is idempotent (upsert by SKU)
- [ ] Returns 200 with report even if some rows failed
- [ ] Returns 422 if entire file is unparseable
- [ ] Test: happy path 100 rows, partial failure, duplicate SKU upsert

**Dependencies:** M1-A2

---

#### M1-E2 — CSV import: opening stock balances
**Status:** `todo` | **Priority:** P1 | **Est:** 4h

**Description:**
`POST /api/v1/import/stock` — bulk receipt movements to establish opening balances.

**CSV format:**
```csv
sku,lot_number,location_code,quantity,expires_at,supplier_ref
WHL-001,LOT-2026-001,COLD-01,500,2026-10-01,PO-4521
```

Process:
1. For each row: find or create item (by SKU), find or create lot (by item+lot_number), find location (by code), write receipt movement
2. All movements share an `import_batch_id` (UUID, returned in response)
3. Idempotency: if same `(item_id, lot_number, location_code)` already has an opening balance movement for this batch, skip

**Acceptance Criteria:**
- [ ] All movements written in a single transaction per batch
- [ ] `import_batch_id` returned in response for audit trail
- [ ] Partial failure rolls back the entire batch (not partial apply)
- [ ] Dry-run mode: `?dry_run=true` validates but does not commit
- [ ] Test: 50-row import, dry run validation, rollback on error

**Dependencies:** M1-D1, M1-E1

---

### M1-F: Frontend — Inventory UI

---

#### M1-F1 — Next.js project scaffold
**Status:** `todo` | **Priority:** P0 | **Est:** 3h

**Description:**
Initialize `frontend/` directory with Next.js 14 (App Router), TypeScript, Tailwind CSS, shadcn/ui, TanStack Query v5.

```
frontend/
  app/
    (auth)/login/page.tsx
    (auth)/signup/page.tsx
    (app)/layout.tsx        -- auth guard + sidebar
    (app)/dashboard/page.tsx
    (app)/inventory/page.tsx
    (app)/inventory/[id]/page.tsx
    (app)/lots/page.tsx
    (app)/stock/page.tsx
  lib/
    api.ts          -- typed fetch wrapper
    queryClient.ts
  components/
    ui/             -- shadcn components
    layout/
```

**Acceptance Criteria:**
- [ ] `pnpm dev` starts at `localhost:3000`
- [ ] `pnpm build` exits 0
- [ ] `pnpm tsc --noEmit` exits 0
- [ ] Auth guard: redirect to `/login` if no access token
- [ ] Token storage: access token in memory (not localStorage), refresh via cookie
- [ ] TanStack Query v5 configured with retry=1, staleTime=30s
- [ ] shadcn/ui theme configured (neutral base, dark mode ready)

**Dependencies:** M0-B5

---

#### M1-F2 — Items list + detail pages
**Status:** `todo` | **Priority:** P1 | **Est:** 5h

**Description:**
`/inventory` page:
- Table with columns: SKU, Name, Unit, Category, Status
- Search box (debounced 300ms, calls `?q=...`)
- Category filter dropdown
- Active/inactive toggle
- "New Item" button opens slide-over form
- Row click navigates to `/inventory/:id`

`/inventory/:id` page:
- Item detail form (editable inline)
- "Save" button (PUT)
- Lots tab: all lots for this item with current balance
- Movement history tab: last 100 movements for this item

**Acceptance Criteria:**
- [ ] Search triggers API call with 300ms debounce
- [ ] Pagination: "Load more" button (cursor-based)
- [ ] Optimistic update on edit (rollback on error)
- [ ] Toast notification on save success/failure
- [ ] Mobile responsive (works on tablet)

**Dependencies:** M1-F1, M1-A2, M1-A3

---

#### M1-F3 — Stock overview page
**Status:** `todo` | **Priority:** P1 | **Est:** 4h

**Description:**
`/stock` page:
- Summary cards: total SKUs, total lots, items below threshold
- Table: item name, SKU, unit, total qty, lot count, status (OK / low / critical)
- Drill down: click item row → expand inline to show lot breakdown by location
- "Receive Stock" button → opens form modal
- "Adjust Stock" button → opens form modal
- "Transfer Stock" button → opens form modal

**Acceptance Criteria:**
- [ ] Refreshes every 60 seconds (TanStack Query refetchInterval)
- [ ] Low stock items highlighted in amber, critical (0 qty) in red
- [ ] Receive/Adjust/Transfer modals validate before submit
- [ ] Lot expiry shown with days-remaining badge
- [ ] Operator-friendly: large buttons, min 44px tap targets

**Dependencies:** M1-F1, M1-D1, M1-D2, M1-D3, M1-D4

---

## M2 — Recipe / BOM (Weeks 7–8)

### M2-A: Recipe Data Model

---

#### M2-A1 — Recipe + recipe_versions migration
**Status:** `todo` | **Priority:** P0 | **Est:** 2h

**Description:**
`migrations/tenant/0004_recipes.sql`:
```sql
CREATE TABLE recipes (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name        TEXT NOT NULL,
  description TEXT,
  category    TEXT,
  output_unit TEXT NOT NULL,
  is_active   BOOLEAN NOT NULL DEFAULT true,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE recipe_versions (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  recipe_id       UUID NOT NULL REFERENCES recipes(id),
  version_number  INT NOT NULL,
  status          TEXT NOT NULL DEFAULT 'draft',  -- draft|published|archived
  batch_size      NUMERIC(18,6) NOT NULL,
  batch_unit      TEXT NOT NULL,
  yield_pct       NUMERIC(5,2) NOT NULL DEFAULT 100.00,
  notes           TEXT,
  cost_per_unit   NUMERIC(18,6),  -- computed on publish
  published_at    TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(recipe_id, version_number)
);

CREATE TABLE recipe_inputs (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  recipe_version_id UUID NOT NULL REFERENCES recipe_versions(id),
  item_id           UUID NOT NULL REFERENCES items(id),
  quantity          NUMERIC(18,6) NOT NULL,
  unit              TEXT NOT NULL,
  waste_pct         NUMERIC(5,2) NOT NULL DEFAULT 0.00,
  is_optional       BOOLEAN NOT NULL DEFAULT false,
  substitute_item_id UUID REFERENCES items(id),
  sort_order        INT NOT NULL DEFAULT 0
);

CREATE TABLE recipe_outputs (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  recipe_version_id UUID NOT NULL REFERENCES recipe_versions(id),
  item_id           UUID NOT NULL REFERENCES items(id),
  quantity          NUMERIC(18,6) NOT NULL,
  unit              TEXT NOT NULL,
  output_type       TEXT NOT NULL DEFAULT 'primary',  -- primary|co-product|by-product
  sort_order        INT NOT NULL DEFAULT 0
);
```

**Acceptance Criteria:**
- [ ] Migration applies cleanly to tenant schema
- [ ] Version number auto-increments per recipe (trigger or application logic)
- [ ] `cost_per_unit` nullable (populated on publish)

**Dependencies:** M0-C1

---

#### M2-A2 — Recipe CRUD endpoints
**Status:** `todo` | **Priority:** P0 | **Est:** 5h

**Description:**
```
POST   /api/v1/recipes                         create recipe + first draft version (manager+)
GET    /api/v1/recipes                         list (viewer+)
GET    /api/v1/recipes/:id                     recipe + current published version (viewer+)
GET    /api/v1/recipes/:id/versions            all versions (viewer+)
GET    /api/v1/recipes/:id/versions/:v         specific version (viewer+)
POST   /api/v1/recipes/:id/versions            create new draft version (manager+)
PUT    /api/v1/recipes/:id/versions/:v         update draft (manager+)
POST   /api/v1/recipes/:id/versions/:v/publish publish version (manager+)
POST   /api/v1/recipes/:id/versions/:v/archive archive version (manager+)
```

Publishing: compute `cost_per_unit` at publish time. Cannot publish if draft has no inputs or no primary output.

**Acceptance Criteria:**
- [ ] Cannot edit a published or archived version (422)
- [ ] Cannot publish if no inputs or no primary output
- [ ] At most one published version per recipe at a time
- [ ] Publishing old version archives current published version
- [ ] `version_number` sequential per recipe (1, 2, 3...)
- [ ] Tests: full lifecycle draft→publish→new draft→publish

**Dependencies:** M2-A1, M0-B7

---

### M2-B: BOM Engine (Pure Domain Logic)

---

#### M2-B1 — BOM engine core
**Status:** `todo` | **Priority:** P0 | **Est:** 6h

**Description:**
`internal/recipe/engine/bom.go` — pure function, zero dependencies on HTTP or DB.

```go
type RecipeVersion struct {
    ID          uuid.UUID
    BatchSize   decimal.Decimal
    BatchUnit   string
    YieldPct    decimal.Decimal
    Inputs      []RecipeInput
    Outputs     []RecipeOutput
}

type ScaleResult struct {
    ScaledInputs  []ScaledInput   // qty needed for target batch
    ScaledOutputs []ScaledOutput  // expected yield for each output
    TotalCost     decimal.Decimal
    CostPerUnit   decimal.Decimal
}

func Scale(rv RecipeVersion, targetQty decimal.Decimal, itemCosts map[uuid.UUID]decimal.Decimal) ScaleResult
```

**Scaling math:**
```
scale_factor = targetQty / batchSize
input_needed = input.qty × scale_factor × (1 + waste_pct/100)
output_expected = output.qty × scale_factor × (yield_pct/100)   -- primary output only
cost = Σ (input_needed × itemCosts[input.item_id])
cost_per_unit = cost / output_expected
```

**Acceptance Criteria:**
- [ ] No imports of `net/http`, `database/sql`, or any external I/O
- [ ] Never uses `float64` (all math via shopspring/decimal)
- [ ] Co-product and by-product outputs scaled but not included in `cost_per_unit` denominator
- [ ] Substitution: if `substitute_item_id` provided and cheaper, use it
- [ ] Returns error if `targetQty <= 0`
- [ ] Returns error if `itemCosts` missing a required input's cost
- [ ] Table-driven tests: 10+ scenarios including edge cases (0% waste, 100% yield, co-products)

**Dependencies:** M2-A1

---

#### M2-B2 — Recursive BOM explosion
**Status:** `todo` | **Priority:** P2 | **Est:** 5h

**Description:**
Sub-recipes: an input item may itself be a recipe output (e.g., "Cheese Brine" is made from salt + water, then used in Cheese recipe).

`internal/recipe/engine/explode.go`:
```go
func Explode(
    rootVersion RecipeVersion,
    targetQty decimal.Decimal,
    versionsByItemID map[uuid.UUID]RecipeVersion, // sub-recipes
    itemCosts map[uuid.UUID]decimal.Decimal,
    depth int, // max recursion depth = 5
) (ExplodedBOM, error)
```

Returns flat list of raw material requirements (leaf nodes only) with full cost rollup.

**Acceptance Criteria:**
- [ ] Handles 3-level deep BOM (A needs B, B needs C, C is raw material)
- [ ] Detects cycles (A→B→A) and returns error
- [ ] Max depth 5 enforced
- [ ] All quantities and costs remain decimal throughout
- [ ] Tests: 3-level BOM, cycle detection, missing sub-recipe

**Dependencies:** M2-B1

---

### M2-C: Frontend — Recipe UI

---

#### M2-C1 — Recipe list + builder pages
**Status:** `todo` | **Priority:** P1 | **Est:** 6h

**Description:**
`/recipes` page:
- List of recipes with: name, category, current published version, cost/unit
- "New Recipe" button

`/recipes/:id` page with tabs:
- **BOM** tab: visual ingredient list with quantities, waste %, optional flag, substitutes
- **Versions** tab: version history (draft/published/archived) with timestamps
- **Cost** tab: cost breakdown table (per-input cost contribution %)

Recipe builder:
- Dynamic input rows (add/remove ingredients)
- Real-time cost calculation (client-side using engine output from API)
- Unit selector per ingredient
- Yield % slider with live output qty preview

**Acceptance Criteria:**
- [ ] Add/remove ingredient rows without page reload
- [ ] Cost preview updates on qty/unit change (debounced API call to `/bom/scale`)
- [ ] Publish button disabled when validation fails (no inputs or no primary output)
- [ ] Version history shows diff between versions (which inputs changed)
- [ ] Mobile-friendly ingredient list

**Dependencies:** M1-F1, M2-A2

---

## M3 — Production (Weeks 9–11)

### M3-A: Production Jobs

---

#### M3-A1 — Production jobs migration
**Status:** `todo` | **Priority:** P0 | **Est:** 2h

**Description:**
`migrations/tenant/0005_production.sql`:
```sql
CREATE TABLE production_jobs (
  id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  job_number          TEXT UNIQUE NOT NULL,  -- auto-generated: PRD-2026-0001
  recipe_version_id   UUID NOT NULL REFERENCES recipe_versions(id),
  status              TEXT NOT NULL DEFAULT 'draft',
  -- draft|scheduled|in_progress|completed|cancelled
  planned_qty         NUMERIC(18,6) NOT NULL,
  actual_qty          NUMERIC(18,6),
  planned_start_at    TIMESTAMPTZ,
  actual_start_at     TIMESTAMPTZ,
  completed_at        TIMESTAMPTZ,
  cancelled_at        TIMESTAMPTZ,
  cancel_reason       TEXT,
  notes               TEXT,
  created_by          UUID NOT NULL,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE production_consumption (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  job_id      UUID NOT NULL REFERENCES production_jobs(id),
  lot_id      UUID NOT NULL REFERENCES lots(id),
  location_id UUID NOT NULL REFERENCES locations(id),
  item_id     UUID NOT NULL REFERENCES items(id),
  planned_qty NUMERIC(18,6) NOT NULL,
  actual_qty  NUMERIC(18,6),
  unit        TEXT NOT NULL,
  is_substitute BOOLEAN NOT NULL DEFAULT false,
  UNIQUE(job_id, lot_id)
);

CREATE TABLE production_outputs (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  job_id         UUID NOT NULL REFERENCES production_jobs(id),
  lot_id         UUID NOT NULL REFERENCES lots(id),
  location_id    UUID NOT NULL REFERENCES locations(id),
  item_id        UUID NOT NULL REFERENCES items(id),
  planned_qty    NUMERIC(18,6) NOT NULL,
  actual_qty     NUMERIC(18,6),
  unit           TEXT NOT NULL,
  output_type    TEXT NOT NULL DEFAULT 'primary'
);

CREATE INDEX prod_jobs_status_idx    ON production_jobs(status, planned_start_at);
CREATE INDEX prod_jobs_recipe_idx    ON production_jobs(recipe_version_id);
CREATE INDEX prod_consumption_job_idx ON production_consumption(job_id);
CREATE INDEX prod_outputs_job_idx     ON production_outputs(job_id);
CREATE INDEX prod_consumption_lot_idx ON production_consumption(lot_id);
CREATE INDEX prod_outputs_lot_idx     ON production_outputs(lot_id);
```

**Dependencies:** M2-A1, M0-C1

---

#### M3-A2 — Production job lifecycle endpoints
**Status:** `todo` | **Priority:** P0 | **Est:** 8h

**Description:**
```
POST /api/v1/production/jobs               create draft job (operator+)
GET  /api/v1/production/jobs               list jobs (viewer+)
GET  /api/v1/production/jobs/:id           job detail with consumption + outputs (viewer+)
POST /api/v1/production/jobs/:id/schedule  draft → scheduled (manager+)
POST /api/v1/production/jobs/:id/start     scheduled → in_progress (operator+)
POST /api/v1/production/jobs/:id/complete  in_progress → completed (operator+)
POST /api/v1/production/jobs/:id/cancel    any → cancelled (manager+)
```

**Start job:**
1. Validate sufficient stock for all planned inputs (via `current_stock` view + SELECT FOR UPDATE)
2. Write `production_consumption` rows (planned qtys)
3. Write `production_out` movements for each input lot (reservation — negative stock)
4. Set `actual_start_at`
5. All in one transaction

**Complete job:**
1. Accept actual consumption qtys (may differ from planned — yield variance)
2. Reconcile: if actual < planned, release unreserved stock (positive adjustment movement)
3. Write `production_outputs` rows with actual qtys
4. Write `production_in` stock movements for output lots
5. Update WAC for output items (M4)
6. Emit `production_job.completed` event
7. All in one transaction

**Cancel job:**
1. If `in_progress`: reverse all consumption movements (release reserved lots)
2. Set `cancelled_at` + `cancel_reason`
3. Emit `production_job.cancelled` event

**Acceptance Criteria:**
- [ ] Insufficient stock on start → 422 with breakdown of which lots are short
- [ ] Start is atomic: all movements or none
- [ ] Complete accepts partial actual qtys (production variance)
- [ ] Cancel reverses reserved stock correctly
- [ ] Job number auto-generated sequential (PRD-YYYY-NNNN)
- [ ] Tests: full lifecycle happy path, insufficient stock, cancel from in_progress

**Dependencies:** M3-A1, M1-D1, M0-B8

---

### M3-B: Traceability

---

#### M3-B1 — Forward + backward trace queries
**Status:** `todo` | **Priority:** P0 | **Est:** 6h

**Description:**
`internal/production/trace/trace.go`

**Forward trace** — "where did this lot end up?":
Starting from a lot_id, find all production jobs that consumed it, then find all output lots from those jobs, then recursively trace those output lots. Uses recursive CTE.

**Backward trace** — "what went into this lot?":
Starting from an output lot_id, find the production job that created it, then find all input lots, then recursively trace those input lots back to raw materials.

```
GET /api/v1/traceability/forward/:lot_id   → consumption chain
GET /api/v1/traceability/backward/:lot_id  → ingredient chain
GET /api/v1/traceability/recall/:lot_id    → all lots that might be affected by a recall
```

**SQL (backward trace):**
```sql
WITH RECURSIVE trace AS (
  -- base: find the job that produced this lot
  SELECT po.job_id, pc.lot_id AS input_lot_id, 1 AS depth
  FROM production_outputs po
  JOIN production_consumption pc ON pc.job_id = po.job_id
  WHERE po.lot_id = $1

  UNION ALL

  -- recursive: trace input lots through their own production history
  SELECT po2.job_id, pc2.lot_id, t.depth + 1
  FROM trace t
  JOIN production_outputs po2 ON po2.lot_id = t.input_lot_id
  JOIN production_consumption pc2 ON pc2.job_id = po2.job_id
  WHERE t.depth < 10
)
SELECT DISTINCT input_lot_id, depth FROM trace ORDER BY depth;
```

**Acceptance Criteria:**
- [ ] Forward trace returns: lot → jobs → output lots, recursively (max depth 10)
- [ ] Backward trace returns: lot → job → input lots → their origins
- [ ] Recall query returns all lots in the forward chain from a given lot
- [ ] Handles cycles gracefully (depth limit)
- [ ] Response includes lot numbers, item names, quantities, timestamps
- [ ] Performance: <500ms for 5-level deep trace on 1M movement rows (index-backed)
- [ ] Tests: 3-level chain, recall impact list

**Dependencies:** M3-A2

---

### M3-C: Frontend — Production UI

---

#### M3-C1 — Production jobs list + execution pages
**Status:** `todo` | **Priority:** P1 | **Est:** 6h

**Description:**
`/production` page:
- Kanban-style columns: Draft | Scheduled | In Progress | Completed
- Each card: job number, recipe name, planned qty, scheduled date, status badge
- "New Job" button

`/production/:id` page:
- Job header: recipe, version, planned qty, status, dates
- **Inputs** tab: ingredient list with lot picker (select which lots to consume), actual qty entry
- **Outputs** tab: output lot creator (assign lot numbers to produced items), actual qty entry
- Action buttons: Start / Complete / Cancel (context-dependent)

**Lot picker:**
- Shows available lots for each input item with current balance
- FEFO (First Expired, First Out) suggested automatically
- Allow override with warning

**Acceptance Criteria:**
- [ ] Kanban re-orders on status change (optimistic update)
- [ ] Lot picker shows expiry dates + available qty
- [ ] FEFO suggestion highlighted (not enforced)
- [ ] Completion form validates actual qtys are reasonable (warning if >20% variance)
- [ ] Operator-mode layout: large buttons, simple form for factory floor use

**Dependencies:** M1-F1, M3-A2

---

#### M3-C2 — Traceability explorer UI
**Status:** `todo` | **Priority:** P1 | **Est:** 4h

**Description:**
`/traceability` page:
- Lot search box (search by lot number or item name)
- Two tabs: Forward Trace | Backward Trace
- Tree/table visualization of the trace chain
- Recall impact button: "How many lots affected?" → show impact count + download CSV

**Acceptance Criteria:**
- [ ] Search finds lots by partial lot number match
- [ ] Forward/backward trace displayed as indented table (not force-graph — too slow)
- [ ] Each row shows: lot number, item, qty, date, status
- [ ] Recall impact CSV export includes: lot_id, lot_number, item_name, qty, location, status
- [ ] Empty state when no trace found (raw material with no production history)

**Dependencies:** M1-F1, M3-B1

---

## M4 — Cost Engine (Week 12)

---

#### M4-A1 — Weighted average cost (WAC) migration
**Status:** `todo` | **Priority:** P1 | **Est:** 2h

**Description:**
`migrations/tenant/0006_costing.sql`:
```sql
CREATE TABLE item_costs (
  item_id         UUID PRIMARY KEY REFERENCES items(id),
  current_wac     NUMERIC(18,6) NOT NULL DEFAULT 0,
  last_updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE cost_history (
  id          BIGSERIAL PRIMARY KEY,
  item_id     UUID NOT NULL REFERENCES items(id),
  wac         NUMERIC(18,6) NOT NULL,
  trigger     TEXT NOT NULL,  -- receipt|production|manual
  reference_id UUID,          -- movement_id or job_id
  snapshot_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX cost_history_item_idx ON cost_history(item_id, snapshot_at DESC);
```

**Dependencies:** M0-C1

---

#### M4-A2 — WAC recalculation on receipt
**Status:** `todo` | **Priority:** P1 | **Est:** 4h

**Description:**
`internal/costing/wac.go`

On every receipt movement, recalculate WAC for the item:
```
new_wac = (current_qty × current_wac + received_qty × unit_cost) / (current_qty + received_qty)
```

`unit_cost` must be provided in the receipt request (new field: `unit_cost NUMERIC(18,6)`).

Write new WAC to `item_costs`, append row to `cost_history`. All in same transaction as the receipt movement.

**Acceptance Criteria:**
- [ ] Receipt endpoint now requires `unit_cost` (validated > 0)
- [ ] WAC computed correctly with shopspring/decimal (no float64)
- [ ] `item_costs` upserted atomically with movement
- [ ] `cost_history` row written with trigger=`receipt`
- [ ] First receipt for an item: WAC = unit_cost
- [ ] Tests: first receipt, second receipt different cost, zero-balance item reset

**Dependencies:** M1-D1, M4-A1

---

#### M4-A3 — WAC recalculation on production completion
**Status:** `todo` | **Priority:** P1 | **Est:** 4h

**Description:**
On production job completion, compute cost of output items:
```
production_cost = Σ (actual_input_qty × input_item_wac)
output_cost_per_unit = production_cost / Σ primary_output_qty
```

Distribute production_cost across co-products by their relative output quantities.
Update `item_costs.current_wac` for all output items.

**Acceptance Criteria:**
- [ ] WAC updated for all output items on completion
- [ ] Co-product WAC calculated proportionally
- [ ] By-product WAC set to 0 (no cost allocated)
- [ ] `cost_history` row written for each output item with trigger=`production`
- [ ] Tests: single output, multiple outputs including co-product

**Dependencies:** M3-A2, M4-A2

---

#### M4-A4 — Cost query endpoints
**Status:** `todo` | **Priority:** P2 | **Est:** 3h

**Description:**
```
GET /api/v1/costs/items                    current WAC for all items
GET /api/v1/costs/items/:id/history        WAC history with sparkline data
GET /api/v1/costs/jobs/:id                 job cost breakdown (planned vs actual)
GET /api/v1/costs/recipes/:version_id      theoretical cost at current WAC
```

**Acceptance Criteria:**
- [ ] History returns up to 90 data points for sparkline
- [ ] Job cost breakdown shows: planned cost, actual cost, variance %, variance amount
- [ ] Recipe theoretical cost calls BOM engine with current item_costs

**Dependencies:** M4-A2, M4-A3, M2-B1

---

## M5 — Dashboard (Week 13)

---

#### M5-A1 — Dashboard API endpoints
**Status:** `todo` | **Priority:** P1 | **Est:** 5h

**Description:**
Single `GET /api/v1/dashboard` endpoint returns all dashboard data in one request (no N+1 waterfalls on the frontend).

**Response:**
```json
{
  "low_stock_items": [...],          // items below min_stock_qty
  "expiring_lots": [...],            // lots expiring in next 30 days
  "jobs_in_progress": [...],         // production jobs with status=in_progress
  "recent_movements": [...],         // last 20 stock movements
  "kpis": {
    "total_skus": 142,
    "total_lots_active": 89,
    "jobs_completed_7d": 12,
    "movements_7d": 340
  }
}
```

All sub-queries run in parallel (Go goroutines), assembled into response. Max 500ms total.

**Acceptance Criteria:**
- [ ] Single HTTP request returns all dashboard data
- [ ] Sub-queries parallelized (not sequential)
- [ ] Response cached for 60 seconds per tenant (Redis not required — in-memory map with TTL is fine)
- [ ] Never returns stale data older than 2 minutes
- [ ] Tests: response shape, parallel execution (mock slow queries)

**Dependencies:** M1-D5, M3-A2

---

#### M5-A2 — Dashboard frontend
**Status:** `todo` | **Priority:** P1 | **Est:** 5h

**Description:**
`/dashboard` page:

Top row — KPI cards:
- Total Active SKUs
- Low Stock Alerts (amber badge)
- Jobs In Progress
- Movements this week

Second row:
- Low Stock table (item, current qty, threshold, % remaining)
- Expiring Lots table (lot, item, location, expires_at, days_remaining)

Third row:
- Recent Activity feed (icon + description + timestamp)
- Jobs In Progress list (job number, recipe, started, % planned qty)

**Acceptance Criteria:**
- [ ] Dashboard loads in < 2 seconds
- [ ] Auto-refreshes every 60 seconds
- [ ] Low stock items are clickable → navigate to `/inventory/:id`
- [ ] Activity feed shows human-readable descriptions ("500 L Whole Milk received at Cold Store")
- [ ] All numbers formatted with locale-aware separators

**Dependencies:** M1-F1, M5-A1

---

## M6 — Onboarding & Polish (Weeks 14–15)

---

#### M6-A1 — Tenant onboarding wizard
**Status:** `todo` | **Priority:** P1 | **Est:** 4h

**Description:**
After signup, new tenants are guided through a 4-step wizard:
1. **Company details** — name, timezone, default unit system (metric/imperial)
2. **Locations setup** — add at least one warehouse and one zone
3. **Item categories** — define category tree (or use presets: dairy, produce, packaging, etc.)
4. **Import or skip** — upload CSV or manually create first items

Wizard state persisted in `tenant.onboarding_step` JSONB column.

**Acceptance Criteria:**
- [ ] Wizard shown on first login (dismissed once step 4 complete or skipped)
- [ ] Each step validates before advancing
- [ ] Back button works
- [ ] "Skip for now" available on steps 2–4
- [ ] Progress indicator (step 1 of 4)

**Dependencies:** M1-F1, M1-E1

---

#### M6-A2 — User management (invite, role change, deactivate)
**Status:** `todo` | **Priority:** P1 | **Est:** 5h

**Description:**
`/settings/users` page — owner only.

**Invite flow:**
1. Owner enters email + role
2. System generates invite token (UUID, stored in DB, expires 48h)
3. Sends invite email with link `/accept-invite?token=...`
4. Invitee sets password on that page → account created

```
POST /api/v1/users/invite          send invite (owner)
GET  /api/v1/users                 list users (owner)
PUT  /api/v1/users/:id/role        change role (owner)
POST /api/v1/users/:id/deactivate  deactivate (owner)
POST /api/v1/users/:id/reactivate  reactivate (owner)
```

**Acceptance Criteria:**
- [ ] Invite token expires after 48 hours
- [ ] Invite email contains correct link
- [ ] Cannot deactivate own account
- [ ] Cannot change own role
- [ ] Deactivated users cannot log in (401)
- [ ] Existing invite tokens invalidated when user manually created with same email

**Dependencies:** M0-B4, M0-B8

---

#### M6-B1 — Error handling standardization (RFC 7807)
**Status:** `todo` | **Priority:** P1 | **Est:** 3h

**Description:**
All error responses follow RFC 7807 Problem+JSON:
```json
{
  "type": "https://anupro.app/errors/insufficient_stock",
  "title": "Insufficient Stock",
  "status": 422,
  "detail": "Lot LOT-2026-001 has 10.5 L available, 50 L required.",
  "instance": "/api/v1/production/jobs/abc-123/start",
  "extensions": {
    "lot_id": "...",
    "available_qty": "10.5",
    "required_qty": "50.0"
  }
}
```

Implement central error handler in chi middleware. Define error types as Go sentinel values in `internal/platform/httpserver/errors.go`.

**Acceptance Criteria:**
- [ ] All 4xx/5xx responses use Problem+JSON format
- [ ] `Content-Type: application/problem+json` header on error responses
- [ ] 500 errors log full stack trace but return only `{"title":"Internal Server Error","status":500}`
- [ ] Error types documented in `docs/errors.md`
- [ ] Existing endpoints updated to use new error types

**Dependencies:** M0-A2

---

#### M6-B2 — API rate limiting per tenant
**Status:** `todo` | **Priority:** P2 | **Est:** 3h

**Description:**
In-memory sliding window rate limiter (no Redis required at this scale). Limits keyed by `tenant_id`.

Limits:
- Default: 500 req/min per tenant
- Import endpoints: 10 req/min per tenant
- Auth endpoints: 20 req/min per IP

Return `429 Too Many Requests` with `Retry-After` header.

**Acceptance Criteria:**
- [ ] 501st request within a minute returns 429
- [ ] `Retry-After` header present on 429
- [ ] Import endpoints have lower limit
- [ ] Auth endpoints rate-limited by IP (not tenant)
- [ ] Rate limit state not shared between API instances (acceptable — in-memory per node)

**Dependencies:** M0-B7

---

#### M6-B3 — OpenAPI spec + API docs
**Status:** `todo` | **Priority:** P2 | **Est:** 4h

**Description:**
Generate OpenAPI 3.1 spec from route definitions using `swaggo/swag` or write spec manually. Serve Swagger UI at `/api/docs` in non-production environments.

At minimum, document all endpoints added by M1–M5.

**Acceptance Criteria:**
- [ ] `GET /api/docs` returns Swagger UI in `ENV != production`
- [ ] All M0–M5 endpoints documented with request/response schemas
- [ ] Auth endpoints include security scheme documentation
- [ ] Spec validates with `openapi-spec-validator`

---

#### M6-B4 — Request validation middleware
**Status:** `todo` | **Priority:** P1 | **Est:** 3h

**Description:**
`internal/platform/httpserver/validate.go`

Central request body validation using `github.com/go-playground/validator/v10`. Define validation tags on all request structs. Return structured 422 errors listing every invalid field.

```json
{
  "type": "https://anupro.app/errors/validation_failed",
  "title": "Validation Failed",
  "status": 422,
  "extensions": {
    "fields": [
      { "field": "quantity", "error": "must be greater than 0" },
      { "field": "lot_id",   "error": "required" }
    ]
  }
}
```

**Acceptance Criteria:**
- [ ] All field errors returned at once (not fail-fast)
- [ ] Field names use `json` tag names (not Go struct names)
- [ ] Custom validator for `decimal.Decimal > 0`
- [ ] Custom validator for `lot_number` format

---

## M7 — Pilot Buffer (Week 16)

> Do not delete or schedule work into this week. Reserve entirely for:

- **Bug fixes** emerging from pilot customer onboarding
- **Performance profiling** (EXPLAIN ANALYZE on slow queries, add indexes as needed)
- **Customer-specific config** (default locations, item categories, unit presets)
- **Runbook documentation**: how to restore from backup, how to add a new tenant, how to run migrations, how to tail logs on Oracle VM
- **Load test**: simulate 10 concurrent tenants doing stock receives + production completions, verify no cross-tenant data leak under load
- **Security pass**: check all endpoints for missing auth, check all DB queries for injection (sqlc generated, so low risk)

---

## Phase 2 Backlog (Post-Pilot)

> Not in MVP. Captured here to avoid scope creep during M0–M7.

| ID | Feature | Notes |
|---|---|---|
| P2-01 | Suppliers & Purchase Orders | Proper PO lifecycle, supplier catalog, inbound QC |
| P2-02 | Quality Control workflows | Inspection lots, hold/release, certificate of analysis upload |
| P2-03 | NLP inventory queries | "How much milk do we have?" via Claude API |
| P2-04 | Mobile app (React Native) | Barcode scan for lot receipt + production |
| P2-05 | QuickBooks integration | Sync COGS, inventory valuation |
| P2-06 | Shopify integration | Finished goods reservation on order |
| P2-07 | Multi-currency | Exchange rates, reporting in local currency |
| P2-08 | IoT sensor integration | Tank level sensors → auto stock movements |
| P2-09 | Multi-stage production workflows | Job depends on job (sequential production) |
| P2-10 | Dynamic-ratio recipes | Recipe inputs vary by batch conditions |
| P2-11 | Demand forecasting | Simple moving average, reorder point suggestions |
| P2-12 | Customer portal | Read-only lot traceability for customers / auditors |
| P2-13 | SSO / SAML | Cognito or Clerk for enterprise customers |
| P2-14 | Audit export (FDA/FSSAI) | One-click compliance report PDF |
| P2-15 | Advanced costing: standard vs actual | Variance reporting, period close |

---

## Dependency Graph (Critical Path)

```
M0-A7 (config)
  └─ M0-A3 (pgx pool)
       ├─ M0-A4 (migrations) ─── M0-C1 (tenant provisioning)
       │    └─ M0-A11 (isolation test) ← GATE
       └─ M0-B7 (auth middleware)
            ├─ M0-B4 (signup) ─── M0-C1
            └─ M0-B5 (login)

M0-C1 (tenant provisioning)
  ├─ M1-D1 (receipt)
  │    ├─ M1-D2 (adjust)
  │    │    └─ M1-D3 (transfer)
  │    └─ M4-A2 (WAC on receipt)
  ├─ M2-A1 (recipe migration)
  │    └─ M2-B1 (BOM engine)
  └─ M3-A1 (production migration)
       └─ M3-A2 (job lifecycle)
            ├─ M3-B1 (traceability)
            └─ M4-A3 (WAC on production)
```

---

## Ticket Summary

| Milestone | Tickets | P0 | P1 | P2 | Est Total |
|---|---|---|---|---|---|
| M0-A Infra | 11 | 9 | 2 | 0 | ~39h |
| M0-B Auth | 8 | 5 | 3 | 0 | ~25h |
| M0-C Tenancy | 3 | 2 | 1 | 0 | ~10h |
| M1 Inventory | 14 | 5 | 7 | 2 | ~52h |
| M2 Recipe | 6 | 3 | 2 | 1 | ~26h |
| M3 Production | 7 | 3 | 3 | 1 | ~31h |
| M4 Costing | 4 | 0 | 3 | 1 | ~13h |
| M5 Dashboard | 2 | 0 | 2 | 0 | ~10h |
| M6 Polish | 6 | 0 | 4 | 2 | ~22h |
| **Total** | **61** | **27** | **27** | **7** | **~228h** |

> 228 hours ÷ 2 engineers at 30 productive hours/week = **~3.8 weeks to MVP feature-complete** (excluding pilot buffer and frontend polish). Adjust for ramp-up and review overhead.
