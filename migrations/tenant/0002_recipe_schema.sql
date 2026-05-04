-- +goose Up
-- +goose StatementBegin

-- Recipe header: what this recipe produces.
CREATE TABLE IF NOT EXISTS recipes (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name           TEXT        NOT NULL,
    output_item_id UUID        NOT NULL REFERENCES items(id),
    notes          TEXT,
    is_active      BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS recipes_output_item_idx ON recipes (output_item_id);
CREATE INDEX IF NOT EXISTS recipes_active_idx      ON recipes (is_active) WHERE is_active = TRUE;

-- recipe_versions: immutable BOM snapshots.
-- Once a version is referenced by a production job, never mutate it.
CREATE TABLE IF NOT EXISTS recipe_versions (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    recipe_id   UUID         NOT NULL REFERENCES recipes(id),
    version     INT          NOT NULL,
    output_qty  NUMERIC(18,6) NOT NULL,
    output_unit TEXT         NOT NULL,
    yield_pct   NUMERIC(5,2) NOT NULL DEFAULT 100.00,
    notes       TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (recipe_id, version)
);

CREATE INDEX IF NOT EXISTS rv_recipe_idx ON recipe_versions (recipe_id, version DESC);

-- recipe_inputs: ingredients (inputs) for a specific version.
CREATE TABLE IF NOT EXISTS recipe_inputs (
    id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    recipe_version_id UUID         NOT NULL REFERENCES recipe_versions(id),
    item_id           UUID         NOT NULL REFERENCES items(id),
    quantity          NUMERIC(18,6) NOT NULL,
    unit              TEXT         NOT NULL,
    is_optional       BOOLEAN      NOT NULL DEFAULT FALSE,
    notes             TEXT,
    sort_order        INT          NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS ri_version_idx ON recipe_inputs (recipe_version_id);
CREATE INDEX IF NOT EXISTS ri_item_idx    ON recipe_inputs (item_id);

-- recipe_outputs: co-products and by-products.
-- The primary output is always on recipe_versions (output_qty + output_unit).
-- This table is for secondary outputs only.
CREATE TABLE IF NOT EXISTS recipe_outputs (
    id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    recipe_version_id UUID         NOT NULL REFERENCES recipe_versions(id),
    item_id           UUID         NOT NULL REFERENCES items(id),
    quantity          NUMERIC(18,6) NOT NULL,
    unit              TEXT         NOT NULL,
    output_type       TEXT         NOT NULL DEFAULT 'co_product', -- co_product | by_product
    notes             TEXT
);

CREATE INDEX IF NOT EXISTS ro_version_idx ON recipe_outputs (recipe_version_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS recipe_outputs;
DROP TABLE IF EXISTS recipe_inputs;
DROP TABLE IF EXISTS recipe_versions;
DROP TABLE IF EXISTS recipes;
-- +goose StatementEnd
