package model

import "time"

type Status string

const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

type Task struct {
	ID          string     `db:"id"          json:"id"`
	Title       string     `db:"title"       json:"title"`
	Description string     `db:"description" json:"description"`
	Status      Status     `db:"status"      json:"status"`
	CreatedAt   time.Time  `db:"created_at"  json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"  json:"updated_at"`
	DeletedAt   *time.Time `db:"deleted_at"  json:"deleted_at,omitempty"`
}

type CreateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      Status `json:"status"`
}

type UpdateTaskRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *Status `json:"status"`
}

type BulkDeleteRequest struct {
	IDs []string `json:"ids"`
}

type HealthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}
