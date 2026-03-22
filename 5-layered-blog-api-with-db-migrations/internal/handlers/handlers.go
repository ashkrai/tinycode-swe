package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"blog-api/internal/models"
	"blog-api/internal/repository"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decode(r *http.Request, dst any) error {
	return json.NewDecoder(r.Body).Decode(dst)
}

func pathInt64(r *http.Request, key string) (int64, error) {
	return strconv.ParseInt(mux.Vars(r)[key], 10, 64)
}

func pagination(r *http.Request) (limit, offset int) {
	limit, offset = 20, 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return
}

// ── Users ─────────────────────────────────────────────────────────────────────

type UserHandler struct{ repo *repository.UserRepository }

func NewUserHandler(repo *repository.UserRepository) *UserHandler {
	return &UserHandler{repo: repo}
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Username == "" || req.Email == "" || req.Password == "" {
		writeError(w, http.StatusUnprocessableEntity, "username, email and password are required")
		return
	}
	// NOTE: hash the password (e.g. bcrypt) before storing in production.
	user, err := h.repo.Create(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

func (h *UserHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt64(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	user, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.repo.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, users)
}

// ── Posts ─────────────────────────────────────────────────────────────────────

type PostHandler struct{ repo *repository.PostRepository }

func NewPostHandler(repo *repository.PostRepository) *PostHandler {
	return &PostHandler{repo: repo}
}

func (h *PostHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreatePostRequest
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Title == "" || req.Slug == "" || req.Body == "" {
		writeError(w, http.StatusUnprocessableEntity, "title, slug and body are required")
		return
	}
	post, err := h.repo.Create(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, post)
}

func (h *PostHandler) GetBySlug(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug"]
	post, err := h.repo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "post not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Fire-and-forget view increment
	go func() { _ = h.repo.IncrementViews(context.Background(), post.ID) }()
	writeJSON(w, http.StatusOK, post)
}

func (h *PostHandler) ListPublished(w http.ResponseWriter, r *http.Request) {
	limit, offset := pagination(r)
	posts, err := h.repo.ListPublished(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, posts)
}

// ExplainPlan is an admin endpoint that returns the EXPLAIN ANALYZE output.
func (h *PostHandler) ExplainPlan(w http.ResponseWriter, r *http.Request) {
	plan, err := h.repo.ExplainListPublished(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"plan": plan})
}

// ── Tags ──────────────────────────────────────────────────────────────────────

type TagHandler struct{ repo *repository.TagRepository }

func NewTagHandler(repo *repository.TagRepository) *TagHandler {
	return &TagHandler{repo: repo}
}

func (h *TagHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateTagRequest
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	tag, err := h.repo.Create(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, tag)
}

func (h *TagHandler) List(w http.ResponseWriter, r *http.Request) {
	tags, err := h.repo.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tags)
}

// ── Comments ──────────────────────────────────────────────────────────────────

type CommentHandler struct{ repo *repository.CommentRepository }

func NewCommentHandler(repo *repository.CommentRepository) *CommentHandler {
	return &CommentHandler{repo: repo}
}

func (h *CommentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateCommentRequest
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Body == "" {
		writeError(w, http.StatusUnprocessableEntity, "body is required")
		return
	}
	comment, err := h.repo.Create(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, comment)
}

func (h *CommentHandler) ListByPost(w http.ResponseWriter, r *http.Request) {
	postID, err := pathInt64(r, "postID")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid postID")
		return
	}
	comments, err := h.repo.ListByPost(r.Context(), postID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, comments)
}

func (h *CommentHandler) Approve(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt64(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.repo.Approve(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "approved"})
}
