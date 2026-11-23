-- 2026-11-23: manual backlog order. Lower rank = higher in the list.
-- Existing tasks keep their number order. Apply once to an existing database:
--   docker exec -i <db container> psql -U pyramid -d pyramid -v ON_ERROR_STOP=1 \
--     < db/migrations/2026-11-23-task-rank.sql
BEGIN;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS rank DOUBLE PRECISION NOT NULL DEFAULT 0;
UPDATE tasks SET rank = number * 1000 WHERE rank = 0;
CREATE INDEX IF NOT EXISTS idx_tasks_project_rank ON tasks(project_id, rank);
COMMIT;
