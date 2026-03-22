package repository

import (
	"context"
	"fmt"

	"github.com/ashkrai/blog-api/internal/model"
	"github.com/jmoiron/sqlx"
)

// PostRepository handles all Postgres operations for posts.
type PostRepository struct {
	db *sqlx.DB
}

func NewPostRepository(db *sqlx.DB) *PostRepository {
	return &PostRepository{db: db}
}

func (r *PostRepository) List(ctx context.Context) ([]model.Post, error) {
	var posts []model.Post
	err := r.db.SelectContext(ctx, &posts, `SELECT id, user_id, title, body, created_at, updated_at FROM posts ORDER BY created_at DESC`)
	return posts, err
}

func (r *PostRepository) GetByID(ctx context.Context, id int) (*model.Post, error) {
	var p model.Post
	err := r.db.GetContext(ctx, &p, `SELECT id, user_id, title, body, created_at, updated_at FROM posts WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PostRepository) Create(ctx context.Context, req model.CreatePostRequest) (*model.Post, error) {
	var p model.Post
	err := r.db.QueryRowxContext(ctx,
		`INSERT INTO posts (user_id, title, body) VALUES ($1, $2, $3)
		 RETURNING id, user_id, title, body, created_at, updated_at`,
		req.UserID, req.Title, req.Body,
	).StructScan(&p)
	return &p, err
}

func (r *PostRepository) Update(ctx context.Context, id int, req model.CreatePostRequest) (*model.Post, error) {
	var p model.Post
	err := r.db.QueryRowxContext(ctx,
		`UPDATE posts SET title=$1, body=$2, updated_at=NOW() WHERE id=$3
		 RETURNING id, user_id, title, body, created_at, updated_at`,
		req.Title, req.Body, id,
	).StructScan(&p)
	return &p, err
}

func (r *PostRepository) Delete(ctx context.Context, id int) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM posts WHERE id=$1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("post %d not found", id)
	}
	return nil
}

// BulkDelete deletes multiple posts in a single transaction.
// Returns the number of deleted rows.
func (r *PostRepository) BulkDelete(ctx context.Context, ids []int) (int64, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	query, args, err := sqlx.In(`DELETE FROM posts WHERE id IN (?)`, ids)
	if err != nil {
		return 0, fmt.Errorf("build IN query: %w", err)
	}
	// sqlx.In uses ? placeholders; rebind for postgres ($1, $2, …).
	query = tx.Rebind(query)

	res, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("bulk delete exec: %w", err)
	}
	n, _ := res.RowsAffected()

	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return n, nil
}
