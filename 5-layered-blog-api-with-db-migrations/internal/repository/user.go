package repository

import (
	"context"
	"database/sql"
	"fmt"

	"blog-api/internal/models"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create inserts a new user and returns the persisted record.
func (r *UserRepository) Create(ctx context.Context, req models.CreateUserRequest) (*models.User, error) {
	const q = `
		INSERT INTO users (username, email, password_hash, bio)
		VALUES ($1, $2, $3, $4)
		RETURNING id, username, email, password_hash, bio, avatar_url, is_active, created_at, updated_at`

	var u models.User
	err := r.db.QueryRowContext(ctx, q,
		req.Username, req.Email,
		req.Password, // caller must hash before passing
		nullString(req.Bio),
	).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash,
		&u.Bio, &u.AvatarURL, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return &u, nil
}

// GetByID fetches a single user by primary key.
func (r *UserRepository) GetByID(ctx context.Context, id int64) (*models.User, error) {
	const q = `
		SELECT id, username, email, password_hash, bio, avatar_url, is_active, created_at, updated_at
		  FROM users WHERE id = $1`

	var u models.User
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash,
		&u.Bio, &u.AvatarURL, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &u, nil
}

// List returns all active users ordered by creation date.
func (r *UserRepository) List(ctx context.Context) ([]models.User, error) {
	const q = `
		SELECT id, username, email, password_hash, bio, avatar_url, is_active, created_at, updated_at
		  FROM users
		 WHERE is_active = true
		 ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(
			&u.ID, &u.Username, &u.Email, &u.PasswordHash,
			&u.Bio, &u.AvatarURL, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}
