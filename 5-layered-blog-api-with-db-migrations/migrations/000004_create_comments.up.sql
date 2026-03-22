CREATE TABLE IF NOT EXISTS comments (
    id          BIGSERIAL   PRIMARY KEY,
    post_id     BIGINT      NOT NULL REFERENCES posts(id)    ON DELETE CASCADE,
    author_id   BIGINT      NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
    parent_id   BIGINT               REFERENCES comments(id) ON DELETE CASCADE,
    body        TEXT        NOT NULL,
    is_approved BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- FK indexes (always index FKs used in JOINs / WHERE clauses)
CREATE INDEX idx_comments_post_id   ON comments (post_id);
CREATE INDEX idx_comments_author_id ON comments (author_id);

-- Partial index: only index rows that have a parent (threaded replies)
CREATE INDEX idx_comments_parent_id ON comments (parent_id)
    WHERE parent_id IS NOT NULL;

-- Partial index: approved comments per post sorted by date (common read path)
CREATE INDEX idx_comments_approved_created ON comments (post_id, created_at DESC)
    WHERE is_approved = TRUE;
