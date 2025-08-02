-- SQLite-optimized URL shortener table
CREATE TABLE IF NOT EXISTS urls (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    short_code TEXT UNIQUE NOT NULL,
    original_url TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    clicks INTEGER DEFAULT 0,

    -- SQLite constraints
    CHECK (length(short_code) >= 3),
    CHECK (length(original_url) > 0),
    CHECK (clicks >= 0)
);

-- SQLite-optimized indexes
CREATE UNIQUE INDEX IF NOT EXISTS idx_urls_short_code ON urls(short_code);
CREATE INDEX IF NOT EXISTS idx_urls_created_at ON urls(created_at DESC);

-- Partial index for popular URLs (SQLite 3.8.0+)
CREATE INDEX IF NOT EXISTS idx_urls_popular ON urls(clicks DESC, created_at DESC) WHERE clicks > 0;

-- SQLite trigger for updated_at (since SQLite doesn't have built-in timestamp updates)
CREATE TRIGGER IF NOT EXISTS update_urls_updated_at
    AFTER UPDATE ON urls
    FOR EACH ROW
    WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE urls SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
