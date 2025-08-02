CREATE TABLE IF NOT EXISTS urls (
    id INTEGER PRIMARY KEY,
    short_code VARCHAR(20) UNIQUE NOT NULL,
    original_url TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    clicks INTEGER DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_urls_short_code ON urls(short_code);
