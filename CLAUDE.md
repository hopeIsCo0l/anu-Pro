**Anu Pro**

**Final Analysis & Decision Document**
# Anu Pro — Project Context for Claude Code

## What This Is
Multi-tenant industrial operations platform for food & beverage manufacturers.
Go backend, Next.js frontend, schema-per-tenant Postgres.

## Tech Stack
- Backend: Go 1.23, modular monolith, chi, pgx/v5, sqlc, goose, river, slog
- Frontend: Next.js + TypeScript + shadcn/ui + TanStack Query
- Auth: Custom JWT (golang-jwt) + Argon2id
- DB: Postgres 16, schema-per-tenant
- Jobs: river (Postgres-backed, no SQS)
- Storage: S3-compatible (R2 in prod, MinIO locally)
- Decimals: shopspring/decimal (never float64)
- Testing: testify + testcontainers-go (real Postgres, no mocks)

## Infrastructure
- Stage 1 (dev): Oracle Cloud Always Free + self-hosted Postgres + Caddy + Cloudflare R2
- Stage 2 (first customer): Fly.io + Neon Pro + Cloudflare R2 + Vercel
- Stage 3 (100+ tenants): AWS

## Directory Structure
cmd/api, cmd/worker, cmd/migrate
internal/platform/{config,db,auth,tenant,storage,httpserver}
internal/{inventory,production,recipe,costing,audit}
migrations/public/, migrations/tenant/
deploy/ (Dockerfile, docker-compose.yml, fly.toml, oracle/)

## Critical Design Rules
1. Append-only stock_movements ledger — never UPDATE/DELETE rows
2. schema-per-tenant: search_path set via pgx BeforeAcquire hook
3. Reset search_path on connection release (prevents cross-tenant leaks)
4. All config from env vars only — no platform-specific APIs
5. Same Docker container runs everywhere
6. Never float64 for money or quantities
7. Run migrations as separate task, not on app startup
8. Cross-tenant isolation test must always pass

## Current Status
M0 (Foundations) in progress. Tickets defined for Weeks 1-3.
See docs/planning/ for full roadmap and ticket breakdown.

## What NOT to Build Yet
- Suppliers/POs (Phase 2)
- QC workflows (Phase 2)
- AI/NLP features (Phase 2)
- Mobile apps (Phase 2)
- Multi-currency (Phase 2)
_Multi-Tenant Industrial Operations Platform for Food & Beverage Manufacturing_

Prepared: April 2026

# Executive Summary

Anu Pro is a viable, well-positioned product targeting a real gap in the mid-market food & beverage manufacturing sector. The key strategic moves are:

- Start vertical (food & beverage), not horizontal
- Build a modular monolith in Go - not microservices
- Use a schema-per-tenant Postgres model with append-only stock ledger + event log
- Start on Oracle Cloud Always Free during development (~\$1/mo)
- Migrate to Fly.io + Neon + Cloudflare the day of first paying customer (~\$40/mo)
- Defer AWS until enterprise compliance or 100+ tenants demands it (~\$230+/mo)

**The product succeeds or fails on three specific things:**

- Ledger design correctness
- First pilot customer selection
- Ruthless scope discipline

# 1\. Product Positioning

## 1.1 One-line Positioning

_Anu Pro is a traceability-first operations platform for small-to-mid-market food processors who've outgrown spreadsheets but can't justify SAP._

## 1.2 Ideal Customer Profile (ICP)

- Food & beverage processor, 15-100 employees
- Revenue \$2M-\$25M
- Currently running on Excel + QuickBooks + possibly a basic inventory tool
- Has at least one regulatory audit experience (FSSAI, FDA, HACCP, SQF, BRC)
- A single decision-maker (owner or ops director) who can sign a contract

## 1.3 Out of Scope in Year 1

Chemicals, textiles, discrete assembly, international/multi-currency, enterprise (500+ employees).

## 1.4 Why This Wedge

Regulatory traceability is a forcing function - customers must buy something. The event-sourced ledger design naturally produces audit-ready output. The same architecture that satisfies food traceability will later serve chemicals and other regulated verticals.

# 2\. Technical Architecture

## 2.1 Stack Decisions

| **Layer**       | **Choice**                  | **Reason**                                               |
| --------------- | --------------------------- | -------------------------------------------------------- |
| Language        | Go 1.23                     | Single binary deploys, strong concurrency, fast compiles |
| Architecture    | Modular monolith            | Microservices are suicide at 2-3 engineers               |
| Router          | chi                         | Idiomatic, minimal, middleware-friendly                  |
| DB driver       | pgx/v5 (direct)             | Better performance, typed scanning                       |
| Query layer     | sqlc                        | Type-safe SQL codegen; ORMs worse than raw SQL in Go     |
| Migrations      | goose                       | Run as separate task, not app startup                    |
| Background jobs | river (Postgres-backed)     | Transactional with business data                         |
| Decimals        | shopspring/decimal          | Never float64 for quantities or money                    |
| Logging         | log/slog (stdlib)           | Stdlib, structured JSON, ships everywhere                |
| Testing         | testify + testcontainers-go | Real Postgres in tests; do NOT mock the DB               |
| Auth            | Custom JWT + Postgres       | Defer Cognito/Clerk until enterprise SSO needed          |

## 2.2 Multi-Tenancy Model

**Schema-per-tenant in a single Postgres database.**

- public schema holds tenants, users, roles, refresh tokens
- tenant\_&lt;slug&gt; schemas hold all business data (items, lots, recipes, production jobs, events)
- Middleware resolves tenant from JWT, then sets search_path per connection via pgx.BeforeAcquire hook
- Scales comfortably to a few hundred tenants on a single RDS or Neon instance
- Defense in depth: enable Row-Level Security as secondary guard

## 2.3 The Three Tables That Matter Most

Everything else is CRUD. These are where correctness lives:

- stock_movements - append-only ledger, quantity signed (positive=in, negative=out). Current stock is a SQL view, never a stored column.
- production_consumption + production_outputs - lot-level inputs and outputs per job, enables recursive CTE traceability (critical for FDA/FSSAI recall scenarios).
- events - append-only event log, one row per state change, written in the same transaction as the business write. Partitioned monthly from day one.

_These designs are non-negotiable. Every other table can be refactored later; changing these after you have customer data is painful._

## 2.4 BOM Engine Design Principles

- Recipes are versioned, never mutated. Production jobs reference a specific recipe_version_id.
- Support yield variance (expected 94% yield when 100kg of input goes in).
- Support co-products and by-products - whey from cheese, trim from butchering.
- Support substitutions at the input level with cost recalculation.
- Keep the engine pure - no HTTP, no DB, just domain logic on input structs.

Defer multi-stage workflows and dynamic-ratio recipes to Phase 2.

# 3\. Infrastructure - Staged Approach

## 3.1 Stage 1: Development (Months 0-3)

**Oracle Cloud Always Free + self-hosted Postgres + Cloudflare + R2**

- Cost: ~\$1/mo (domain only)
- Risk: acceptable - no real customers, can tolerate downtime
- Requirement: automated daily backups to R2, weekly tested restores

## 3.2 Stage 2: First Paying Customer (Months 3-18)

**Fly.io + Neon Pro + Cloudflare + R2 + Vercel**

- Cost: ~\$40/mo at pilot, ~\$80-150/mo at 50 tenants
- Migration from Stage 1: ~3 days with a 1-hour maintenance window
- Risk: managed - Neon handles Postgres, Fly handles containers, both SOC 2 certified

## 3.3 Stage 3: Enterprise / Scale (Year 2+)

**AWS (ECS Fargate + RDS Multi-AZ + S3 + CloudFront)**

- Cost: ~\$230/mo at pilot-equivalent, ~\$700/mo at 50 tenants
- Trigger: enterprise deal requiring VPC peering, SOC 2 Type II, or data residency
- Migration effort: ~3-5 weeks if platform-agnostic discipline was maintained

## 3.4 Portability Rules (Apply From Day One)

These are what make migration between stages cheap:

- All config via environment variables, no platform-specific APIs
- S3-compatible storage interface - works with R2, MinIO, S3, Backblaze B2
- Plain Docker container as the deployment artifact - same image runs anywhere
- Standard Postgres only - no Neon extensions, no RDS-specific features
- Structured logs to stdout - every platform can ingest
- docker-compose up reproduces production locally

_Violate these rules and migration becomes a quarter-long rewrite. Follow them and it's a week._

# 4\. MVP Scope

**16-week build, 2-3 engineers, pilot-ready deliverable.**

| **Weeks** | **Module**           | **Deliverable**                                            |
| --------- | -------------------- | ---------------------------------------------------------- |
| 1-3       | Foundations          | Auth, tenancy, CI/CD, IaC, base Go scaffolding, migrations |
| 4-6       | Inventory            | Items, lots, locations, stock ledger, basic UI, CSV import |
| 7-8       | Recipe               | BOM definition, versioning, cost computation, recipe UI    |
| 9-11      | Production           | Jobs, lot-level consumption, outputs, traceability query   |
| 12        | Cost engine          | Weighted average costing, cost history                     |
| 13        | Dashboard            | KPIs, low-stock alerts, recent activity feed               |
| 14-15     | Onboarding & polish  | Data import tooling, user docs, bug fixes                  |
| 16        | Pilot support buffer | Do not delete this week                                    |

## 4.1 Explicitly NOT in MVP

Phase 2 or later:

- Suppliers and purchase orders (ingest via 'receipt' movements for pilot)
- Quality control workflows
- NLP querying / AI features
- Mobile apps
- External integrations (QuickBooks, Shopify, etc.)
- Multi-currency
- IoT connectivity

# 5\. Top Risks - Ranked

| **#** | **Risk**                              | **Prob** | **Impact**   | **Mitigation**                                                                          |
| ----- | ------------------------------------- | -------- | ------------ | --------------------------------------------------------------------------------------- |
| 1     | Inventory math drifts from reality    | HIGH     | HIGH         | Append-only ledger, idempotency keys, SELECT FOR UPDATE, nightly reconciliation         |
| 2     | First pilot has messy data            | HIGH     | HIGH         | Budget 1 engineer-week per tenant onboarding, CSV import tool, allow 'unverified' items |
| 3     | Scope creep from pilot customer       | HIGH     | HIGH         | Distinguish configurable (JSONB) from custom code - say no, write it down               |
| 4     | BOM can't represent real recipes      | MED      | HIGH         | Interview 5 food manufacturers first; use JSONB attributes liberally                    |
| 5     | Multi-tenancy data leak               | LOW      | CATASTROPHIC | Schema-per-tenant + RLS + cross-tenant isolation tests                                  |
| 6     | Factory floor UX unusable on tablets  | MED      | HIGH         | Build 'operator mode' UI early, test on actual factory floors                           |
| 7     | Backup/restore silently broken        | MED      | CATASTROPHIC | Weekly automated restore tests, alerts on backup failures                               |
| 8     | Oracle free tier pulled               | LOW-MED  | MED          | Portable architecture means re-deploy is hours, not days                                |
| 9     | Event table grows to billions of rows | MED      | MED          | Monthly partitioning from day one; archive to R2 after 2 years                          |
| 10    | Competitor launches vertical food SKU | LOW      | HIGH         | Ship first 20 customers fast, win on food-specific depth                                |

# 6\. Financial Plan

## 6.1 Development Cost

- 16 weeks × 2-3 engineers at blended rates: ~\$150k-250k to pilot-ready
- If bootstrapping with founders' time: 4 months of opportunity cost

## 6.2 Infrastructure Cost (Monthly)

| **Stage**                 | **Tenants** | **\$/mo**  |
| ------------------------- | ----------- | ---------- |
| Development (Oracle free) | 0           | ~\$1       |
| Pilot (Fly + Neon)        | 1-10        | ~\$40-60   |
| Growth (Fly + Neon)       | 10-50       | ~\$80-150  |
| Enterprise (AWS)          | 50+         | ~\$300-700 |

## 6.3 Pricing Model

- Pilot (customers 1-5): flat \$500-1,000/mo in exchange for design partnership, logo rights, case studies. Do not set list pricing yet.
- Post-pilot target: \$299/mo base + \$29/user + metered transactions over 500/mo. Validate with real usage data before committing.

## 6.4 Unit Economics Target

- Break-even for 3-person team: ~\$500k ARR = ~60-70 tenants at \$800/mo average
- Realistic timeline to break-even: 18-24 months if pilots go well

# 7\. What to Do Monday Morning

## Week 1

- Set up the repo with the modular monolith layout
- Get docker compose up working - Postgres, MinIO, Go 'hello world'
- Write the Config struct and the pgx connection pool with tenant schema switching
- Write the first migration: tenants, users, events tables

## Weeks 2-3

- Provision Oracle VM, set up Postgres + Caddy + systemd
- Get CI/CD deploying to Oracle via GitHub Actions
- Get backups working BEFORE writing any business logic. Test the restore.
- Build auth: signup, login, JWT issuance, refresh rotation

## Weeks 4+

Execute the MVP plan in Section 4.

## At First Paying Customer

Execute the Stage 1 → Stage 2 migration plan.

# 8\. The Three Things That Matter Most

If you forget everything else in this document, remember:

**1\. The stock ledger + event log design is the foundation.**

Get it right once. It's non-negotiable for food traceability and defines what the product can ever do.

**2\. The first two pilot customers define the product.**

Pick them for fit - small, sophisticated, friendly - not for size. One great design partner beats five lukewarm evaluators.

**3\. Say no to 80% of feature requests in year one.**

Ship inventory, production, and traceability beautifully. Everything else is noise until those three are loved.

_- End of document -_