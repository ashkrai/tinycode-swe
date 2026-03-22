CREATE TYPE post_status AS ENUM ('draft', 'published', 'archived');

CREATE TABLE IF NOT EXISTS posts (
    id            BIGSERIAL    PRIMARY KEY,
    author_id     BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title         VARCHAR(300) NOT NULL,
    slug          VARCHAR(350) NOT NULL UNIQUE,
    summary       TEXT,
    body          TEXT         NOT NULL,
    status        post_status  NOT NULL DEFAULT 'draft',
    published_at  TIMESTAMPTZ,
    views_count   BIGINT       NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Index the FK for JOIN performance
CREATE INDEX idx_posts_author_id ON posts (author_id);

-- Partial index: only index published rows for the common listing query
CREATE INDEX idx_posts_status_published ON posts (status, published_at DESC)
    WHERE status = 'published';

CREATE INDEX idx_posts_slug       ON posts (slug);
CREATE INDEX idx_posts_created_at ON posts (created_at DESC);
