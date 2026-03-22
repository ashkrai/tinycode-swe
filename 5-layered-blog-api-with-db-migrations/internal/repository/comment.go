package repository

import (
	"context"
	"database/sql"
	"fmt"

	"blog-api/internal/models"
)

type CommentRepository struct {
	db *sql.DB
}

func NewCommentRepository(db *sql.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

// Create inserts a new comment.
func (r *CommentRepository) Create(ctx context.Context, req models.CreateCommentRequest) (*models.Comment, error) {
	const q = `
		INSERT INTO comments (post_id, author_id, parent_id, body)
		VALUES ($1, $2, $3, $4)
		RETURNING id, post_id, author_id, parent_id, body, is_approved, created_at, updated_at`

	var parentID sql.NullInt64
	if req.ParentID != nil {
		parentID = sql.NullInt64{Int64: *req.ParentID, Valid: true}
	}

	var c models.Comment
	err := r.db.QueryRowContext(ctx, q,
		req.PostID, req.AuthorID, parentID, req.Body,
	).Scan(
		&c.ID, &c.PostID, &c.AuthorID, &c.ParentID,
		&c.Body, &c.IsApproved, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create comment: %w", err)
	}
	return &c, nil
}

// ListByPost returns approved comments for a post, nested one level deep.
func (r *CommentRepository) ListByPost(ctx context.Context, postID int64) ([]models.Comment, error) {
	const q = `
		SELECT c.id, c.post_id, c.author_id, c.parent_id, c.body,
		       c.is_approved, c.created_at, c.updated_at,
		       u.username
		  FROM comments c
		  JOIN users u ON u.id = c.author_id
		 WHERE c.post_id = $1
		   AND c.is_approved = true
		 ORDER BY c.parent_id NULLS FIRST, c.created_at ASC`

	rows, err := r.db.QueryContext(ctx, q, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	commentMap := make(map[int64]*models.Comment)
	var roots []models.Comment

	for rows.Next() {
		var c models.Comment
		var u models.User
		if err := rows.Scan(
			&c.ID, &c.PostID, &c.AuthorID, &c.ParentID,
			&c.Body, &c.IsApproved, &c.CreatedAt, &c.UpdatedAt,
			&u.Username,
		); err != nil {
			return nil, err
		}
		c.Author = &u
		commentMap[c.ID] = &c

		if !c.ParentID.Valid {
			roots = append(roots, c)
		} else if parent, ok := commentMap[c.ParentID.Int64]; ok {
			parent.Replies = append(parent.Replies, c)
		}
	}
	return roots, rows.Err()
}

// Approve marks a comment as approved.
func (r *CommentRepository) Approve(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE comments SET is_approved = true, updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("comment %d not found", id)
	}
	return nil
}
