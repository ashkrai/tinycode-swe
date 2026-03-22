package models

import "time"

// User is a row in the users table.
type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateUserRequest is the JSON body for POST /api/v1/users.
type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// UpdateUserRequest is the JSON body for PUT /api/v1/users/{id}.
type UpdateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// ErrorResponse is the standard error envelope.
type ErrorResponse struct {
	Error string `json:"error"`
}
