-- 2026-11-24: which half of a split board column an item is in.
-- Apply once to an existing database:
--   docker exec -i <db container> psql -U pyramid -d pyramid -v ON_ERROR_STOP=1 \
--     < db/migrations/2026-11-24-board-done.sql
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS board_done BOOLEAN NOT NULL DEFAULT false;
