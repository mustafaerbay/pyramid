-- 2026-11-23: saved work item queries (private or shared with everyone).
-- Apply once to an existing database:
--   docker exec -i <db container> psql -U pyramid -d pyramid -v ON_ERROR_STOP=1 \
--     < db/migrations/2026-11-23-saved-queries.sql
CREATE TABLE IF NOT EXISTS saved_queries (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    filters    JSONB NOT NULL DEFAULT '{}',
    group_by   TEXT NOT NULL DEFAULT 'status',
    shared     BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
