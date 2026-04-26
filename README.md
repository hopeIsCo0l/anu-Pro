# Anu Pro

Multi-tenant operations platform for food & beverage manufacturers. Built for small-to-mid-market processors who've outgrown spreadsheets but can't justify SAP.

## What It Does

- **Inventory** — lot-tracked stock movements via an append-only ledger. Current stock is always a view, never a stored column.
- **Recipe / BOM** — versioned bill-of-materials with yield variance, co-products, substitutions, and cost computation.
- **Production** — lot-level consumption and output recording per job. Enables recursive traceability for FDA/FSSAI recall queries.
- **Costing** — weighted average cost engine. Pure domain logic, no HTTP, no DB.
- **Audit** — append-only event log written in the same transaction as every business write. Partitioned monthly from day one.

## Tech Stack

| Layer | Choice |
|---|---|
| Language | Go 1.23 |
| Architecture | Modular monolith |
| Router | chi |
| DB driver | pgx/v5 (direct, no ORM) |
| Query layer | sqlc (type-safe codegen) |
| Migrations | goose (run as separate task) |
| Background jobs | river (Postgres-backed) |
| Decimals | shopspring/decimal (never float64) |
| Logging | log/slog (structured JSON) |
| Auth | Custom JWT (golang-jwt) + Argon2id |
| Storage | S3-compatible (Cloudflare R2 in prod, MinIO locally) |
| Frontend | Next.js + TypeScript + shadcn/ui + TanStack Query |
| Testing | testify + testcontainers-go (real Postgres, no mocks) |

## Directory Structure

```
cmd/
  api/        HTTP API server
  worker/     Background job worker
  migrate/    Database migration runner

internal/
  platform/
    config/       Env-var config (envconfig)
    db/           pgx connection pool + tenant search_path switching
    auth/         JWT issuance, validation, refresh rotation
    tenant/       Tenant resolution middleware
    storage/      S3-compatible storage interface
    httpserver/   Shared server setup
  inventory/      Items, lots, locations, stock ledger
  production/     Production jobs, consumption, outputs
  recipe/         BOM definition, versioning
  costing/        Weighted average cost engine
  audit/          Event log writes

migrations/
  public/     Shared schema (tenants, users, roles, refresh_tokens)
  tenant/     Per-tenant schema (items, lots, recipes, jobs, events, stock_movements)

deploy/
  Dockerfile
  docker-compose.yml
  fly.toml
  oracle/
```

## Local Development

### Prerequisites

- Go 1.23+
- Docker (for local Postgres + MinIO via docker-compose)

### Environment Variables

Copy `.env.example` and fill in:

```bash
cp .env.example .env
```

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP listen port |
| `ENV` | `development` | Runtime environment |
| `DATABASE_URL` | required | Postgres connection string |
| `JWT_SECRET` | required | Signing key for JWTs |
| `JWT_ACCESS_EXPIRY_MIN` | `15` | Access token TTL (minutes) |
| `JWT_REFRESH_EXPIRY_DAYS` | `30` | Refresh token TTL (days) |
| `STORAGE_ENDPOINT` | required | S3-compatible endpoint URL |
| `STORAGE_BUCKET` | required | Bucket name |
| `STORAGE_ACCESS_KEY` | required | Storage access key |
| `STORAGE_SECRET_KEY` | required | Storage secret key |
| `STORAGE_REGION` | `auto` | Storage region |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |

### Start Local Services

```bash
docker compose up -d
```

Starts Postgres 16 + MinIO locally. Matches the production container layout.

### Run Migrations

```bash
go run ./cmd/migrate
```

Migrations run as a separate task — never on app startup.

### Start the API

```bash
go run ./cmd/api
```

Health check: `GET /health` → `200 {"status":"ok"}`

### Start the Worker

```bash
go run ./cmd/worker
```

### Build

```bash
go build ./...
```

### Tests

```bash
go test ./...
```

Tests spin up real Postgres via testcontainers. No mocks.

## Architecture Decisions

### Multi-Tenancy

Schema-per-tenant in a single Postgres database:
- `public` schema — tenants, users, roles, refresh_tokens
- `tenant_<slug>` schemas — all business data

Middleware resolves tenant from the JWT, then sets `search_path` per connection via a `pgx.BeforeAcquire` hook. `search_path` resets on connection release to prevent cross-tenant leaks. Row-Level Security is an additional guard.

### The Three Tables That Matter

These are where correctness lives. Everything else can be refactored later; these cannot change after you have customer data.

**`stock_movements`** — append-only ledger. Quantity is signed (positive = in, negative = out). Current stock is computed via SQL view. Never UPDATE or DELETE rows.

**`production_consumption` + `production_outputs`** — lot-level inputs and outputs per job. Enables recursive CTE traceability for FDA/FSSAI recall scenarios.

**`events`** — append-only event log. One row per state change, written in the same transaction as the business write. Partitioned monthly.

### Portability Rules

All config via environment variables. S3-compatible storage interface. Standard Postgres only (no vendor extensions). Single Docker container runs everywhere — Oracle Cloud, Fly.io, AWS, locally.

## Infrastructure

| Stage | Stack | Cost |
|---|---|---|
| Development | Oracle Cloud Always Free + self-hosted Postgres + Caddy + R2 | ~$1/mo |
| First customer | Fly.io + Neon Pro + R2 + Vercel | ~$40/mo |
| 100+ tenants | AWS (ECS Fargate + RDS Multi-AZ + S3 + CloudFront) | ~$300+/mo |

## MVP Scope (16 weeks)

| Weeks | Module |
|---|---|
| 1–3 | Foundations — auth, tenancy, CI/CD, migrations |
| 4–6 | Inventory — items, lots, locations, stock ledger, CSV import |
| 7–8 | Recipe — BOM, versioning, cost computation |
| 9–11 | Production — jobs, lot-level consumption, outputs, traceability |
| 12 | Cost engine — weighted average costing |
| 13 | Dashboard — KPIs, low-stock alerts, activity feed |
| 14–15 | Onboarding & polish — data import, docs, bug fixes |
| 16 | Pilot support buffer |

**Not in MVP:** suppliers/POs, QC workflows, AI/NLP features, mobile apps, external integrations (QuickBooks, Shopify), multi-currency.

## Current Status

M0 (Foundations) in progress — project scaffold and config complete, working through core platform packages.

See `backlog.md` for full ticket breakdown.
