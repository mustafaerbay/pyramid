-- 2026-11-23: per-project custom fields and their values on tasks.
-- Apply once to an existing database:
--   docker exec -i <db container> psql -U pyramid -d pyramid -v ON_ERROR_STOP=1 \
--     < db/migrations/2026-11-23-custom-fields.sql
BEGIN;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS custom_fields JSONB;
ALTER TABLE tasks    ADD COLUMN IF NOT EXISTS custom_values JSONB;
COMMIT;
