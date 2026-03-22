package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/ashkrai/go-docker-api/internal/models"
	"github.com/go-chi/chi/v5"
)

// Handler holds shared dependencies.
type Handler struct {
	db *sql.DB
}

// New wires up a Handler.
func New(db *sql.DB) *Handler {
	return &Handler{db: db}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, models.ErrorResponse{Error: msg})
}

func urlID(r *http.Request) (int, bool) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	return id, err == nil
}

// ── Healthz ───────────────────────────────────────────────────────────────────

// Healthz godoc
//
//	GET /healthz
//	Responds 200 {"status":"ok"} when the API and database are reachable.
func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	if err := h.db.PingContext(r.Context()); err != nil {
		writeErr(w, http.StatusServiceUnavailable, "database unreachable: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ── ListUsers ─────────────────────────────────────────────────────────────────

// ListUsers godoc
//
//	GET /api/v1/users
//	Returns all users ordered by id.
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT id, name, email, created_at, updated_at
		   FROM users
		  ORDER BY id`)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	users := make([]models.User, 0)
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt, &u.UpdatedAt); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, users)
}

// ── CreateUser ────────────────────────────────────────────────────────────────

// CreateUser godoc
//
//	POST /api/v1/users
//	Body: {"name":"Alice","email":"alice@ashkrai.com"}
//	Returns 201 with the created user.
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.Name == "" || req.Email == "" {
		writeErr(w, http.StatusUnprocessableEntity, "name and email are required")
		return
	}

	var u models.User
	err := h.db.QueryRowContext(r.Context(),
		`INSERT INTO users (name, email)
		 VALUES ($1, $2)
		 RETURNING id, name, email, created_at, updated_at`,
		req.Name, req.Email,
	).Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, u)
}

// ── GetUser ───────────────────────────────────────────────────────────────────

// GetUser godoc
//
//	GET /api/v1/users/{id}
//	Returns a single user or 404.
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "id must be an integer")
		return
	}

	var u models.User
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, name, email, created_at, updated_at
		   FROM users
		  WHERE id = $1`, id,
	).Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt, &u.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		writeErr(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, u)
}

// ── UpdateUser ────────────────────────────────────────────────────────────────

// UpdateUser godoc
//
//	PUT /api/v1/users/{id}
//	Body: {"name":"...","email":"..."}
//	Returns the updated user or 404.
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "id must be an integer")
		return
	}

	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" || req.Email == "" {
		writeErr(w, http.StatusUnprocessableEntity, "name and email are required")
		return
	}

	var u models.User
	err := h.db.QueryRowContext(r.Context(),
		`UPDATE users
		    SET name       = $1,
		        email      = $2,
		        updated_at = NOW()
		  WHERE id = $3
		  RETURNING id, name, email, created_at, updated_at`,
		req.Name, req.Email, id,
	).Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt, &u.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		writeErr(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, u)
}

// ── DeleteUser ────────────────────────────────────────────────────────────────

// DeleteUser godoc
//
//	DELETE /api/v1/users/{id}
//	Returns 204 on success, 404 if not found.
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, ok := urlID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "id must be an integer")
		return
	}

	res, err := h.db.ExecContext(r.Context(),
		`DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		writeErr(w, http.StatusNotFound, "user not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
