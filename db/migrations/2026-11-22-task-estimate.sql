-- 2026-11-22: a task's own man-day size (before splitting across assignees).
-- Apply once to an existing database after 2026-11-22-man-days.sql.
--   docker exec -i <db container> psql -U pyramid -d pyramid -v ON_ERROR_STOP=1 \
--     < db/migrations/2026-11-22-task-estimate.sql
BEGIN;

ALTER TABLE tasks
    ADD COLUMN estimate_man_days NUMERIC(7,2)
        CHECK (estimate_man_days >= 0 AND estimate_man_days <= 9999);

COMMIT;
