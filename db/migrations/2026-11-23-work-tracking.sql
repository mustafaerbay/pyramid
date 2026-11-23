-- 2026-11-23: remaining and completed work per task, in man-days
-- (story_points already exists). Apply once to an existing database:
--   docker exec -i <db container> psql -U pyramid -d pyramid -v ON_ERROR_STOP=1 \
--     < db/migrations/2026-11-23-work-tracking.sql
BEGIN;
ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS remaining_man_days NUMERIC(7,2) CHECK (remaining_man_days >= 0 AND remaining_man_days <= 9999),
    ADD COLUMN IF NOT EXISTS completed_man_days NUMERIC(7,2) CHECK (completed_man_days >= 0 AND completed_man_days <= 9999);
COMMIT;
