-- migrate: up

ALTER TABLE posts ADD COLUMN IF NOT EXISTS published_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_posts_published_at ON posts(published_at DESC);
