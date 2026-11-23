-- 2026-11-23: per-project board settings (WIP limits, swimlanes, card colours).
-- Apply once to an existing database:
--   docker exec -i <db container> psql -U pyramid -d pyramid -v ON_ERROR_STOP=1 \
--     < db/migrations/2026-11-23-board-settings.sql
ALTER TABLE projects ADD COLUMN IF NOT EXISTS board_settings JSONB;
