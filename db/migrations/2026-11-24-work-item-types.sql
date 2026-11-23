-- 2026-11-24: per-project work item types on top of the built-in levels.
-- Apply once to an existing database:
--   docker exec -i <db container> psql -U pyramid -d pyramid -v ON_ERROR_STOP=1 \
--     < db/migrations/2026-11-24-work-item-types.sql
BEGIN;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS work_item_types JSONB;
ALTER TABLE tasks    ADD COLUMN IF NOT EXISTS custom_type TEXT;
COMMIT;
