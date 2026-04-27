-- Replace non-partial unique indexes with partial ones that exclude soft-deleted rows.
-- GORM's AutoMigrate creates non-partial unique indexes from model tags; this migration
-- converts them to partial indexes so that soft-deleted session names can be reused.
DROP INDEX IF EXISTS idx_routers_session_name;
CREATE UNIQUE INDEX IF NOT EXISTS idx_routers_session_name ON routers(session_name) WHERE deleted_at IS NULL;
