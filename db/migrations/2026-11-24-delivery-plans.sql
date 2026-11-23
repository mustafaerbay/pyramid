-- 2026-11-24: saved delivery plans (lanes, types, time span, markers).
CREATE TABLE IF NOT EXISTS delivery_plans (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    shared     BOOLEAN NOT NULL DEFAULT false,
    config     JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
