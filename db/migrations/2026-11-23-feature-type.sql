-- 2026-11-23: "feature" work item type (Epic → Feature → Story/Bug → Task).
-- Apply once to an existing database (ADD VALUE can't run in a transaction):
--   docker exec -i <db container> psql -U pyramid -d pyramid -v ON_ERROR_STOP=1 \
--     < db/migrations/2026-11-23-feature-type.sql
ALTER TYPE task_type ADD VALUE IF NOT EXISTS 'feature' BEFORE 'story';
