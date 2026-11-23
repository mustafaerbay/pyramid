-- 2026-11-24: follow items, notification settings, project hooks (Slack, Teams, webhooks).
ALTER TYPE notification_type ADD VALUE IF NOT EXISTS 'follow';
CREATE TABLE IF NOT EXISTS task_followers (
    task_id    UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    person_id  UUID NOT NULL REFERENCES people(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (task_id, person_id)
);
ALTER TABLE users ADD COLUMN IF NOT EXISTS notification_prefs JSONB NOT NULL DEFAULT '{}';
CREATE TABLE IF NOT EXISTS project_hooks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    kind        TEXT NOT NULL,
    name        TEXT NOT NULL,
    url         TEXT NOT NULL,
    secret      TEXT,
    events      JSONB NOT NULL DEFAULT '[]',
    active      BOOLEAN NOT NULL DEFAULT true,
    last_status INT,
    last_error  TEXT,
    last_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS project_hooks_project ON project_hooks(project_id);
