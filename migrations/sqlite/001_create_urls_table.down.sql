-- Drop SQLite trigger
DROP TRIGGER IF EXISTS update_urls_updated_at;

-- Drop indexes
DROP INDEX IF EXISTS idx_urls_popular;
DROP INDEX IF EXISTS idx_urls_created_at;
DROP INDEX IF EXISTS idx_urls_short_code;

-- Drop table
DROP TABLE IF EXISTS urls;
