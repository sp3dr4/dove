-- PostgreSQL-optimized URL shortener table
CREATE TABLE IF NOT EXISTS urls (
    id BIGSERIAL PRIMARY KEY,
    short_code VARCHAR(20) UNIQUE NOT NULL,
    original_url TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    clicks BIGINT DEFAULT 0,

    -- Constraints
    CONSTRAINT urls_short_code_length CHECK (length(short_code) >= 3),
    CONSTRAINT urls_original_url_not_empty CHECK (length(original_url) > 0),
    CONSTRAINT urls_clicks_non_negative CHECK (clicks >= 0)
);

-- Optimized indexes for PostgreSQL
CREATE UNIQUE INDEX IF NOT EXISTS idx_urls_short_code
ON urls(short_code);

CREATE INDEX IF NOT EXISTS idx_urls_created_at
ON urls(created_at DESC);

-- Partial index for frequently accessed URLs
CREATE INDEX IF NOT EXISTS idx_urls_popular
ON urls(clicks DESC, created_at DESC)
WHERE clicks > 0;

-- Auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_urls_updated_at
    BEFORE UPDATE ON urls
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add table comment for documentation
COMMENT ON TABLE urls IS 'URL shortener mappings with click tracking';
COMMENT ON COLUMN urls.short_code IS 'Unique short identifier for the URL';
COMMENT ON COLUMN urls.original_url IS 'The original long URL';
COMMENT ON COLUMN urls.clicks IS 'Number of times this short URL has been accessed';
COMMENT ON COLUMN urls.created_at IS 'When the short URL was created';
COMMENT ON COLUMN urls.updated_at IS 'When the record was last modified';
