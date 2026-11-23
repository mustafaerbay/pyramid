-- 2026-11-24: per-project workflows. tasks.state is the item's state in its
-- project's workflow; tasks.status stays as that state's category.
-- Apply once to an existing database:
--   docker exec -i <db container> psql -U pyramid -d pyramid -v ON_ERROR_STOP=1 \
--     < db/migrations/2026-11-24-workflow.sql
BEGIN;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS workflow JSONB;
ALTER TABLE tasks    ADD COLUMN IF NOT EXISTS state TEXT;
UPDATE tasks SET state = status::text WHERE state IS NULL;
COMMIT;
