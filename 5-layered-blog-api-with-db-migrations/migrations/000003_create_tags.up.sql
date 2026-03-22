CREATE TABLE IF NOT EXISTS tags (
    id         BIGSERIAL    PRIMARY KEY,
    name       VARCHAR(100) NOT NULL UNIQUE,
    slug       VARCHAR(110) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tags_slug ON tags (slug);

-- Many-to-many join table: posts <-> tags
CREATE TABLE IF NOT EXISTS post_tags (
    post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    tag_id  BIGINT NOT NULL REFERENCES tags(id)  ON DELETE CASCADE,
    PRIMARY KEY (post_id, tag_id)
);

-- Index for "posts by tag" query
CREATE INDEX idx_post_tags_tag_id  ON post_tags (tag_id);
CREATE INDEX idx_post_tags_post_id ON post_tags (post_id);
