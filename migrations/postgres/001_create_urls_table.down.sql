-- Drop PostgreSQL-specific elements
DROP TRIGGER IF EXISTS update_urls_updated_at ON urls;
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_urls_popular;
DROP INDEX IF EXISTS idx_urls_created_at;
DROP INDEX IF EXISTS idx_urls_short_code;

-- Drop table
DROP TABLE IF EXISTS urls;
