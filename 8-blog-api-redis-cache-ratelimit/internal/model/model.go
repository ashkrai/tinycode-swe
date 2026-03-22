package model

import "time"

// User represents a blog user.
type User struct {
	ID        int       `db:"id"        json:"id"`
	Username  string    `db:"username"  json:"username"`
	Email     string    `db:"email"     json:"email"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// Post represents a blog post.
type Post struct {
	ID        int       `db:"id"         json:"id"`
	UserID    int       `db:"user_id"    json:"user_id"`
	Title     string    `db:"title"      json:"title"`
	Body      string    `db:"body"       json:"body"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// Tag represents a post tag.
type Tag struct {
	ID   int    `db:"id"   json:"id"`
	Name string `db:"name" json:"name"`
}

// Comment represents a comment on a post.
type Comment struct {
	ID        int       `db:"id"         json:"id"`
	PostID    int       `db:"post_id"    json:"post_id"`
	UserID    int       `db:"user_id"    json:"user_id"`
	Body      string    `db:"body"       json:"body"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// CreatePostRequest is the validated request body for creating/updating a post.
type CreatePostRequest struct {
	UserID int    `json:"user_id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

func (r *CreatePostRequest) Validate() map[string]string {
	errs := map[string]string{}
	if r.UserID <= 0 {
		errs["user_id"] = "must be a positive integer"
	}
	if len(r.Title) == 0 {
		errs["title"] = "required"
	}
	if len(r.Title) > 255 {
		errs["title"] = "max 255 characters"
	}
	if len(r.Body) == 0 {
		errs["body"] = "required"
	}
	return errs
}

// BulkDeleteRequest carries a list of post IDs to delete.
type BulkDeleteRequest struct {
	IDs []int `json:"ids"`
}

func (r *BulkDeleteRequest) Validate() map[string]string {
	errs := map[string]string{}
	if len(r.IDs) == 0 {
		errs["ids"] = "must contain at least one id"
	}
	return errs
}
