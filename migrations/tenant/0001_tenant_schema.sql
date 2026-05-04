-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS items (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    sku         TEXT        UNIQUE NOT NULL,
    name        TEXT        NOT NULL,
    description TEXT,
    unit        TEXT        NOT NULL,
    category    TEXT,
    min_stock_qty NUMERIC(18,6),
    attributes  JSONB       NOT NULL DEFAULT '{}',
    is_active   BOOLEAN     NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS items_sku_idx      ON items (sku);
CREATE INDEX IF NOT EXISTS items_category_idx ON items (category) WHERE category IS NOT NULL;
CREATE INDEX IF NOT EXISTS items_active_idx   ON items (is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS items_name_fts_idx ON items USING gin (to_tsvector('english', name));

CREATE TABLE IF NOT EXISTS locations (
    id        UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    code      TEXT    UNIQUE NOT NULL,
    name      TEXT    NOT NULL,
    parent_id UUID    REFERENCES locations(id),
    type      TEXT    NOT NULL DEFAULT 'zone',
    is_active BOOLEAN NOT NULL DEFAULT true
);

CREATE TABLE IF NOT EXISTS lots (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    item_id      UUID        NOT NULL REFERENCES items(id),
    lot_number   TEXT        NOT NULL,
    supplier_ref TEXT,
    received_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at   TIMESTAMPTZ,
    status       TEXT        NOT NULL DEFAULT 'active',
    attributes   JSONB       NOT NULL DEFAULT '{}',
    UNIQUE (item_id, lot_number)
);

CREATE INDEX IF NOT EXISTS lots_item_id_idx  ON lots (item_id);
CREATE INDEX IF NOT EXISTS lots_expires_idx  ON lots (expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS lots_status_idx   ON lots (status);

-- Append-only stock ledger. Never UPDATE or DELETE rows.
-- Current balance is derived from the current_stock view below.
CREATE TABLE IF NOT EXISTS stock_movements (
    id              BIGSERIAL,
    lot_id          UUID         NOT NULL REFERENCES lots(id),
    location_id     UUID         NOT NULL REFERENCES locations(id),
    quantity        NUMERIC(18,6) NOT NULL,
    movement_type   TEXT         NOT NULL,
    reference_id    UUID,
    idempotency_key TEXT         NOT NULL,
    unit_cost       NUMERIC(18,6),
    notes           TEXT,
    actor_id        UUID         NOT NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
) PARTITION BY RANGE (created_at);

CREATE TABLE IF NOT EXISTS stock_movements_2026_04 PARTITION OF stock_movements
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');

CREATE TABLE IF NOT EXISTS stock_movements_2026_05 PARTITION OF stock_movements
    FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');

CREATE TABLE IF NOT EXISTS stock_movements_2026_06 PARTITION OF stock_movements
    FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');

CREATE TABLE IF NOT EXISTS stock_movements_2026_07 PARTITION OF stock_movements
    FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');

CREATE TABLE IF NOT EXISTS stock_movements_2026_08 PARTITION OF stock_movements
    FOR VALUES FROM ('2026-08-01') TO ('2026-09-01');

CREATE INDEX IF NOT EXISTS sm_idem_idx        ON stock_movements (idempotency_key);
CREATE INDEX IF NOT EXISTS sm_lot_id_idx      ON stock_movements (lot_id,      created_at DESC);
CREATE INDEX IF NOT EXISTS sm_location_id_idx ON stock_movements (location_id, created_at DESC);
CREATE INDEX IF NOT EXISTS sm_type_idx        ON stock_movements (movement_type, created_at DESC);
CREATE INDEX IF NOT EXISTS sm_ref_idx         ON stock_movements (reference_id) WHERE reference_id IS NOT NULL;

-- Current stock is ALWAYS derived from this view, never from a stored column.
CREATE OR REPLACE VIEW current_stock AS
    SELECT
        lot_id,
        location_id,
        SUM(quantity) AS quantity
    FROM stock_movements
    GROUP BY lot_id, location_id
    HAVING SUM(quantity) != 0;

-- Tenant-level event log (one row per state change, same transaction as the write).
CREATE TABLE IF NOT EXISTS events (
    id          BIGSERIAL,
    actor_id    UUID,
    event_type  TEXT        NOT NULL,
    entity_type TEXT,
    entity_id   UUID,
    payload     JSONB       NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
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

CREATE INDEX IF NOT EXISTS events_type_idx   ON events (event_type,  created_at DESC);
CREATE INDEX IF NOT EXISTS events_entity_idx ON events (entity_type, entity_id,  created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW  IF EXISTS current_stock;
DROP TABLE IF EXISTS events_2026_08;
DROP TABLE IF EXISTS events_2026_07;
DROP TABLE IF EXISTS events_2026_06;
DROP TABLE IF EXISTS events_2026_05;
DROP TABLE IF EXISTS events_2026_04;
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS stock_movements_2026_08;
DROP TABLE IF EXISTS stock_movements_2026_07;
DROP TABLE IF EXISTS stock_movements_2026_06;
DROP TABLE IF EXISTS stock_movements_2026_05;
DROP TABLE IF EXISTS stock_movements_2026_04;
DROP TABLE IF EXISTS stock_movements;
DROP TABLE IF EXISTS lots;
DROP TABLE IF EXISTS locations;
DROP TABLE IF EXISTS items;
-- +goose StatementEnd
