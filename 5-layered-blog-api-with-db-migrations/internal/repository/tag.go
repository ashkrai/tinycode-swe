package repository

import (
	"context"
	"database/sql"
	"fmt"

	"blog-api/internal/models"
)

type TagRepository struct {
	db *sql.DB
}

func NewTagRepository(db *sql.DB) *TagRepository {
	return &TagRepository{db: db}
}

// Create inserts a tag; on slug conflict it updates the name (upsert).
func (r *TagRepository) Create(ctx context.Context, req models.CreateTagRequest) (*models.Tag, error) {
	const q = `
		INSERT INTO tags (name, slug) VALUES ($1, $2)
		ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name
		RETURNING id, name, slug, created_at`

	var t models.Tag
	err := r.db.QueryRowContext(ctx, q, req.Name, req.Slug).
		Scan(&t.ID, &t.Name, &t.Slug, &t.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create tag: %w", err)
	}
	return &t, nil
}

// List returns all tags alphabetically.
func (r *TagRepository) List(ctx context.Context) ([]models.Tag, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, slug, created_at FROM tags ORDER BY name`)
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

// ListByPost returns all tags for a given post.
func (r *TagRepository) ListByPost(ctx context.Context, postID int64) ([]models.Tag, error) {
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
