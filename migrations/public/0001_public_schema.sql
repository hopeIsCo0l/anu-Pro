-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS tenants (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    slug         TEXT        UNIQUE NOT NULL,
    name         TEXT        NOT NULL,
    timezone     TEXT        NOT NULL DEFAULT 'UTC',
    plan         TEXT        NOT NULL DEFAULT 'pilot',
    onboarding_step JSONB    NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    suspended_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID        NOT NULL REFERENCES tenants(id),
    email         TEXT        UNIQUE NOT NULL,
    password_hash TEXT        NOT NULL,
    full_name     TEXT        NOT NULL,
    role          TEXT        NOT NULL DEFAULT 'operator',
    is_active     BOOLEAN     NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_login_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS users_tenant_id_idx ON users (tenant_id);
CREATE INDEX IF NOT EXISTS users_email_idx     ON users (email);

CREATE TABLE IF NOT EXISTS invite_tokens (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        NOT NULL REFERENCES tenants(id),
    email      TEXT        NOT NULL,
    role       TEXT        NOT NULL DEFAULT 'operator',
    token_hash TEXT        UNIQUE NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id),
    token_hash TEXT        UNIQUE NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    user_agent TEXT,
    ip_addr    TEXT
);

CREATE INDEX IF NOT EXISTS refresh_tokens_user_id_idx ON refresh_tokens (user_id);

-- Public-level audit log (login, signup, tenant events).
-- Partitioned monthly from day one; add partitions via maintenance job.
CREATE TABLE IF NOT EXISTS events (
    id         BIGSERIAL,
    tenant_id  UUID        REFERENCES tenants(id),
    actor_id   UUID,
    event_type TEXT        NOT NULL,
    payload    JSONB       NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
) PARTITION BY RANGE (created_at);

CREATE TABLE IF NOT EXISTS events_2026_04 PARTITION OF events
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');

CREATE TABLE IF NOT EXISTS events_2026_05 PARTITION OF events
    FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');

CREATE TABLE IF NOT EXISTS events_2026_06 PARTITION OF events
    FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');

CREATE TABLE IF NOT EXISTS events_2026_07 PARTITION OF events
    FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');

CREATE TABLE IF NOT EXISTS events_2026_08 PARTITION OF events
    FOR VALUES FROM ('2026-08-01') TO ('2026-09-01');

CREATE INDEX IF NOT EXISTS events_tenant_idx ON events (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS events_type_idx   ON events (event_type, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS events_2026_08;
DROP TABLE IF EXISTS events_2026_07;
DROP TABLE IF EXISTS events_2026_06;
DROP TABLE IF EXISTS events_2026_05;
DROP TABLE IF EXISTS events_2026_04;
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS invite_tokens;
DROP INDEX IF EXISTS users_email_idx;
DROP INDEX IF EXISTS users_tenant_id_idx;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS tenants;
-- +goose StatementEnd
