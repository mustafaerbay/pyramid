-- 2026-11-22: man-day capacity and task effort.
-- Apply once to an existing database; db/schema.sql already has the final
-- shape for fresh volumes.
--   docker exec -i <db container> psql -U pyramid -d pyramid -v ON_ERROR_STOP=1 \
--     < db/migrations/2026-11-22-man-days.sql
BEGIN;

ALTER TABLE people
    ADD COLUMN monthly_capacity_man_days NUMERIC(5,2) NOT NULL DEFAULT 20
        CHECK (monthly_capacity_man_days >= 0 AND monthly_capacity_man_days <= 31);

ALTER TABLE tasks
    ADD COLUMN start_date DATE,
    ADD CONSTRAINT tasks_start_before_due CHECK (start_date IS NULL OR due_date IS NULL OR start_date <= due_date);

ALTER TABLE task_assignees
    ADD COLUMN man_days NUMERIC(6,2) NOT NULL DEFAULT 0
        CHECK (man_days >= 0 AND man_days <= 999);

COMMIT;
