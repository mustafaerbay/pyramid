-- 2026-11-24: teams inside projects, and restricted projects.
-- Apply once to an existing database:
--   docker exec -i <db container> psql -U pyramid -d pyramid -v ON_ERROR_STOP=1 \
--     < db/migrations/2026-11-24-teams-permissions.sql
BEGIN;
CREATE TABLE IF NOT EXISTS project_teams (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    area_ids   JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_project_teams_name ON project_teams(project_id, lower(name));
CREATE TABLE IF NOT EXISTS project_team_members (
    team_id   UUID NOT NULL REFERENCES project_teams(id) ON DELETE CASCADE,
    person_id UUID NOT NULL REFERENCES people(id) ON DELETE CASCADE,
    PRIMARY KEY (team_id, person_id)
);
ALTER TABLE sprints         ADD COLUMN IF NOT EXISTS team_id UUID REFERENCES project_teams(id) ON DELETE SET NULL;
ALTER TABLE projects        ADD COLUMN IF NOT EXISTS restricted BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE project_members ADD COLUMN IF NOT EXISTS access TEXT NOT NULL DEFAULT 'write';
COMMIT;
