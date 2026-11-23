-- 2026-11-24: dashboards with widgets, personal or shared.
CREATE TABLE IF NOT EXISTS dashboards (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    shared     BOOLEAN NOT NULL DEFAULT false,
    widgets    JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
