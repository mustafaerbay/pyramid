-- 2026-11-24: per-project work item rules (required fields per state,
-- defaults, read-only fields). Apply once to an existing database:
--   docker exec -i <db container> psql -U pyramid -d pyramid -v ON_ERROR_STOP=1 \
--     < db/migrations/2026-11-24-rules.sql
ALTER TABLE projects ADD COLUMN IF NOT EXISTS rules JSONB;
