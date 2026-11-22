-- 2026-11-22: project versions (release plan).
-- Apply once to an existing database; db/schema.sql already has the final
-- shape for fresh volumes.
--   docker exec -i pyramid-db-1 psql -U postgres -d pyramid -v ON_ERROR_STOP=1 \
--     < db/migrations/2026-11-22-project-versions.sql
BEGIN;

CREATE TYPE version_status AS ENUM ('planned', 'in_progress', 'released', 'cancelled');

CREATE TABLE project_versions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT,
    status      version_status NOT NULL DEFAULT 'planned',
    target_date DATE,
    released_at DATE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, name)
);

ALTER TABLE milestones
    ADD COLUMN version_id UUID REFERENCES project_versions(id) ON DELETE SET NULL;
CREATE INDEX idx_milestones_version_id ON milestones (version_id);

-- Replaced by project_versions; the free-text values were dummy data.
ALTER TABLE projects DROP COLUMN release_version;

COMMIT;
