-- 2026-11-24: repositories linked to projects and what their webhooks report
-- (commits, pull/merge requests, pipeline runs), linked to work items.
CREATE TABLE IF NOT EXISTS project_repos (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    provider   TEXT NOT NULL,
    name       TEXT NOT NULL,
    url        TEXT NOT NULL,
    secret     TEXT NOT NULL,
    last_event_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS git_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repo_id     UUID NOT NULL REFERENCES project_repos(id) ON DELETE CASCADE,
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    kind        TEXT NOT NULL,
    external_id TEXT NOT NULL,
    title       TEXT NOT NULL,
    url         TEXT,
    author      TEXT,
    state       TEXT,
    ref         TEXT,
    sha         TEXT,
    items       JSONB NOT NULL DEFAULT '[]',
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (repo_id, kind, external_id)
);
CREATE INDEX IF NOT EXISTS git_events_project ON git_events(project_id, occurred_at DESC);
