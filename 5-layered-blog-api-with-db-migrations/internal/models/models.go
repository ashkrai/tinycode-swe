package models

import (
	"database/sql"
	"time"
)

// ── User ─────────────────────────────────────────────────────────────────────

type User struct {
	ID           int64          `json:"id"`
	Username     string         `json:"username"`
	Email        string         `json:"email"`
	PasswordHash string         `json:"-"`
	Bio          sql.NullString `json:"bio,omitempty"`
	AvatarURL    sql.NullString `json:"avatar_url,omitempty"`
	IsActive     bool           `json:"is_active"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type CreateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Bio      string `json:"bio,omitempty"`
}

// ── Post ─────────────────────────────────────────────────────────────────────

type PostStatus string

const (
	PostStatusDraft     PostStatus = "draft"
	PostStatusPublished PostStatus = "published"
	PostStatusArchived  PostStatus = "archived"
)

type Post struct {
	ID                 int64          `json:"id"`
	AuthorID           int64          `json:"author_id"`
	Title              string         `json:"title"`
	Slug               string         `json:"slug"`
	Summary            sql.NullString `json:"summary,omitempty"`
	Body               string         `json:"body"`
	Status             PostStatus     `json:"status"`
	PublishedAt        sql.NullTime   `json:"published_at,omitempty"`
	ViewsCount         int64          `json:"views_count"`
	ReadingTimeMinutes int16          `json:"reading_time_minutes"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`

	// Joined fields (populated by specific queries)
	Author *User  `json:"author,omitempty"`
	Tags   []Tag  `json:"tags,omitempty"`
}

type CreatePostRequest struct {
	AuthorID int64   `json:"author_id"`
	Title    string  `json:"title"`
	Slug     string  `json:"slug"`
	Summary  string  `json:"summary,omitempty"`
	Body     string  `json:"body"`
	Status   string  `json:"status"`
	TagIDs   []int64 `json:"tag_ids,omitempty"`
}

// ── Tag ──────────────────────────────────────────────────────────────────────

type Tag struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateTagRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// ── Comment ──────────────────────────────────────────────────────────────────

type Comment struct {
	ID         int64         `json:"id"`
	PostID     int64         `json:"post_id"`
	AuthorID   int64         `json:"author_id"`
	ParentID   sql.NullInt64 `json:"parent_id,omitempty"`
	Body       string        `json:"body"`
	IsApproved bool          `json:"is_approved"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`

	// Joined fields
	Author  *User     `json:"author,omitempty"`
	Replies []Comment `json:"replies,omitempty"`
}

type CreateCommentRequest struct {
	PostID   int64  `json:"post_id"`
	AuthorID int64  `json:"author_id"`
	ParentID *int64 `json:"parent_id,omitempty"`
	Body     string `json:"body"`
}
