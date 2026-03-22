package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"blog-api/internal/models"
)

type PostRepository struct {
	db *sql.DB
}

func NewPostRepository(db *sql.DB) *PostRepository {
	return &PostRepository{db: db}
}

// Create inserts a post and attaches tags in a single transaction.
func (r *PostRepository) Create(ctx context.Context, req models.CreatePostRequest) (*models.Post, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	const insertPost = `
		INSERT INTO posts (author_id, title, slug, summary, body, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, author_id, title, slug, summary, body, status,
		          published_at, views_count, reading_time_minutes, created_at, updated_at`

	status := models.PostStatus(req.Status)
	if status == "" {
		status = models.PostStatusDraft
	}

	var p models.Post
	err = tx.QueryRowContext(ctx, insertPost,
		req.AuthorID, req.Title, req.Slug,
		nullString(req.Summary), req.Body, status,
	).Scan(
		&p.ID, &p.AuthorID, &p.Title, &p.Slug, &p.Summary,
		&p.Body, &p.Status, &p.PublishedAt, &p.ViewsCount,
		&p.ReadingTimeMinutes, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert post: %w", err)
	}

	if len(req.TagIDs) > 0 {
		if err := attachTags(ctx, tx, p.ID, req.TagIDs); err != nil {
			return nil, err
		}
	}

	return &p, tx.Commit()
}

// GetBySlug fetches a post with its author and tags.
func (r *PostRepository) GetBySlug(ctx context.Context, slug string) (*models.Post, error) {
	const q = `
		SELECT p.id, p.author_id, p.title, p.slug, p.summary, p.body,
		       p.status, p.published_at, p.views_count, p.reading_time_minutes,
		       p.created_at, p.updated_at,
		       u.id, u.username, u.email
		  FROM posts p
		  JOIN users u ON u.id = p.author_id
		 WHERE p.slug = $1`

	var p models.Post
	var u models.User
	err := r.db.QueryRowContext(ctx, q, slug).Scan(
		&p.ID, &p.AuthorID, &p.Title, &p.Slug, &p.Summary, &p.Body,
		&p.Status, &p.PublishedAt, &p.ViewsCount, &p.ReadingTimeMinutes,
		&p.CreatedAt, &p.UpdatedAt,
		&u.ID, &u.Username, &u.Email,
	)
	if err != nil {
		return nil, fmt.Errorf("get post by slug: %w", err)
	}
	p.Author = &u
	p.Tags, err = r.tagsByPostID(ctx, p.ID)
	return &p, err
}

// ListPublished returns published posts newest-first (uses partial index).
func (r *PostRepository) ListPublished(ctx context.Context, limit, offset int) ([]models.Post, error) {
	const q = `
		SELECT p.id, p.author_id, p.title, p.slug, p.summary,
		       p.status, p.published_at, p.views_count, p.reading_time_minutes,
		       p.created_at, p.updated_at,
		       u.username
		  FROM posts p
		  JOIN users u ON u.id = p.author_id
		 WHERE p.status = 'published'
		 ORDER BY p.published_at DESC
		 LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, q, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var p models.Post
		var u models.User
		if err := rows.Scan(
			&p.ID, &p.AuthorID, &p.Title, &p.Slug, &p.Summary,
			&p.Status, &p.PublishedAt, &p.ViewsCount, &p.ReadingTimeMinutes,
			&p.CreatedAt, &p.UpdatedAt, &u.Username,
		); err != nil {
			return nil, err
		}
		p.Author = &u
		posts = append(posts, p)
	}
	return posts, rows.Err()
}

// ExplainListPublished runs EXPLAIN ANALYZE on the list query and logs the plan.
func (r *PostRepository) ExplainListPublished(ctx context.Context) (string, error) {
	const q = `
		EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
		SELECT p.id, p.title, p.slug, p.published_at, u.username
		  FROM posts p
		  JOIN users u ON u.id = p.author_id
		 WHERE p.status = 'published'
		 ORDER BY p.published_at DESC
		 LIMIT 20 OFFSET 0`

	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return "", fmt.Errorf("explain: %w", err)
	}
	defer rows.Close()

	var lines []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return "", err
		}
		lines = append(lines, line)
	}
	plan := strings.Join(lines, "\n")
	log.Printf("[EXPLAIN ANALYZE]\n%s", plan)
	return plan, rows.Err()
}

// IncrementViews atomically bumps the view counter for a post.
func (r *PostRepository) IncrementViews(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE posts SET views_count = views_count + 1 WHERE id = $1`, id)
	return err
}

// ── helpers ───────────────────────────────────────────────────────────────────

func attachTags(ctx context.Context, tx *sql.Tx, postID int64, tagIDs []int64) error {
	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO post_tags (post_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, tid := range tagIDs {
		if _, err := stmt.ExecContext(ctx, postID, tid); err != nil {
			return fmt.Errorf("attach tag %d: %w", tid, err)
		}
	}
	return nil
}

func (r *PostRepository) tagsByPostID(ctx context.Context, postID int64) ([]models.Tag, error) {
	const q = `
		SELECT t.id, t.name, t.slug, t.created_at
		  FROM tags t
		  JOIN post_tags pt ON pt.tag_id = t.id
		 WHERE pt.post_id = $1
		 ORDER BY t.name`

	rows, err := r.db.QueryContext(ctx, q, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var t models.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.CreatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}
