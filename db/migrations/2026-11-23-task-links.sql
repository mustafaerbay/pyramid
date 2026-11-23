-- 2026-11-23: "related" links between tasks (both ways, stored once with
-- task_id < other_id) and "duplicate of". Apply once to an existing database:
--   docker exec -i <db container> psql -U pyramid -d pyramid -v ON_ERROR_STOP=1 \
--     < db/migrations/2026-11-23-task-links.sql
BEGIN;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS duplicate_of_id UUID REFERENCES tasks(id) ON DELETE SET NULL;
CREATE TABLE IF NOT EXISTS task_related (
    task_id  UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    other_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, other_id),
    CHECK (task_id < other_id)
);
CREATE INDEX IF NOT EXISTS idx_task_related_other ON task_related(other_id);
COMMIT;
