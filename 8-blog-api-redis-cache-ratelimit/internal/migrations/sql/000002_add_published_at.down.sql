-- migrate: down

DROP INDEX IF EXISTS idx_posts_published_at;
ALTER TABLE posts DROP COLUMN IF EXISTS published_at;
