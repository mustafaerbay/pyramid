-- 2026-11-24: per-sprint capacity settings per person.
-- Apply once to an existing database:
--   docker exec -i <db container> psql -U pyramid -d pyramid -v ON_ERROR_STOP=1 \
--     < db/migrations/2026-11-24-sprint-capacity.sql
CREATE TABLE IF NOT EXISTS sprint_capacity (
    sprint_id  UUID NOT NULL REFERENCES sprints(id) ON DELETE CASCADE,
    person_id  UUID NOT NULL REFERENCES people(id) ON DELETE CASCADE,
    days_off   INTEGER NOT NULL DEFAULT 0 CHECK (days_off >= 0 AND days_off <= 60),
    per_day    NUMERIC(4,2) CHECK (per_day >= 0 AND per_day <= 1.5),
    activity   TEXT,
    PRIMARY KEY (sprint_id, person_id)
);
