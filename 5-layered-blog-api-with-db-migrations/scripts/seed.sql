-- scripts/seed.sql  —  idempotent sample data
-- Run: make seed
-- Or:  psql -h localhost -U postgres -d blog -f scripts/seed.sql

-- ── Users ─────────────────────────────────────────────────────────────────────
INSERT INTO users (username, email, password_hash, bio) VALUES
  ('alice',   'alice@example.com',   'hashed_pw_alice',   'Senior engineer and weekend baker.'),
  ('bob',     'bob@example.com',     'hashed_pw_bob',     'DevOps enthusiast. Coffee addict.'),
  ('charlie', 'charlie@example.com', 'hashed_pw_charlie', 'Open-source contributor.')
ON CONFLICT (email) DO NOTHING;

-- ── Tags ──────────────────────────────────────────────────────────────────────
INSERT INTO tags (name, slug) VALUES
  ('PostgreSQL',  'postgresql'),
  ('Go',          'go'),
  ('Performance', 'performance'),
  ('Tutorial',    'tutorial')
ON CONFLICT (slug) DO NOTHING;

-- ── Posts ─────────────────────────────────────────────────────────────────────
INSERT INTO posts (author_id, title, slug, summary, body, status, published_at) VALUES
(
  (SELECT id FROM users WHERE username = 'alice'),
  'Getting Started with PostgreSQL Schema Design',
  'getting-started-postgresql-schema-design',
  'A practical guide to designing robust schemas in PostgreSQL.',
  'When designing a schema, think carefully about your access patterns first.
Normalization is important, but so is knowing when to denormalize for reads.
Always index your foreign keys and frequently filtered columns.
Use partial indexes to keep index size small when only a subset of rows is queried.',
  'published',
  NOW() - INTERVAL '3 days'
),
(
  (SELECT id FROM users WHERE username = 'bob'),
  'golang-migrate in Practice',
  'golang-migrate-in-practice',
  'How to manage database migrations reliably with golang-migrate.',
  'golang-migrate is a simple, file-based migration runner.
Each migration has an up and a down file numbered sequentially.
Run migrate up to advance and migrate down 1 to roll back one step.
Always write idempotent migrations and test your down files regularly.',
  'published',
  NOW() - INTERVAL '1 day'
),
(
  (SELECT id FROM users WHERE username = 'charlie'),
  'Understanding EXPLAIN ANALYZE',
  'understanding-explain-analyze',
  'Use EXPLAIN ANALYZE to inspect query plans and catch slow scans.',
  'EXPLAIN ANALYZE executes your query and returns the real execution plan.
Look for Seq Scan on large tables — that usually signals a missing index.
The BUFFERS option shows cache hit and miss ratios.
Use the FORMAT JSON option for machine-readable output.',
  'draft',
  NULL
)
ON CONFLICT (slug) DO NOTHING;

-- ── post_tags ─────────────────────────────────────────────────────────────────
INSERT INTO post_tags (post_id, tag_id)
SELECT p.id, t.id
  FROM posts p, tags t
 WHERE p.slug = 'getting-started-postgresql-schema-design'
   AND t.slug IN ('postgresql', 'tutorial')
ON CONFLICT DO NOTHING;

INSERT INTO post_tags (post_id, tag_id)
SELECT p.id, t.id
  FROM posts p, tags t
 WHERE p.slug = 'golang-migrate-in-practice'
   AND t.slug IN ('go', 'tutorial')
ON CONFLICT DO NOTHING;

INSERT INTO post_tags (post_id, tag_id)
SELECT p.id, t.id
  FROM posts p, tags t
 WHERE p.slug = 'understanding-explain-analyze'
   AND t.slug IN ('postgresql', 'performance')
ON CONFLICT DO NOTHING;

-- ── Comments ──────────────────────────────────────────────────────────────────
INSERT INTO comments (post_id, author_id, body, is_approved)
SELECT
  (SELECT id FROM posts WHERE slug = 'getting-started-postgresql-schema-design'),
  (SELECT id FROM users WHERE username = 'bob'),
  'Great article Alice! The partial index tip saved us 200ms per query.',
  TRUE
WHERE NOT EXISTS (
  SELECT 1 FROM comments
   WHERE post_id = (SELECT id FROM posts WHERE slug = 'getting-started-postgresql-schema-design')
     AND author_id = (SELECT id FROM users WHERE username = 'bob')
);

INSERT INTO comments (post_id, author_id, body, is_approved)
SELECT
  (SELECT id FROM posts WHERE slug = 'getting-started-postgresql-schema-design'),
  (SELECT id FROM users WHERE username = 'charlie'),
  'Would love a follow-up post on covering indexes and index-only scans.',
  TRUE
WHERE NOT EXISTS (
  SELECT 1 FROM comments
   WHERE post_id = (SELECT id FROM posts WHERE slug = 'getting-started-postgresql-schema-design')
     AND author_id = (SELECT id FROM users WHERE username = 'charlie')
);
