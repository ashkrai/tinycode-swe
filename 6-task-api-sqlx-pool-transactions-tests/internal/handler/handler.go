package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ashkrai/taskapi/internal/model"
	"github.com/ashkrai/taskapi/internal/repository"
)

// Repo is the interface the Handler depends on — easy to mock in tests.
type Repo interface {
	GetAll(ctx context.Context) ([]model.Task, error)
	GetByID(ctx context.Context, id string) (model.Task, error)
	Create(ctx context.Context, req model.CreateTaskRequest) (model.Task, error)
	Update(ctx context.Context, id string, req model.UpdateTaskRequest) (model.Task, error)
	Delete(ctx context.Context, id string) error
	BulkDelete(ctx context.Context, ids []string) (int64, error)
	Ping(ctx context.Context) error
}

// Handler holds all HTTP handler methods and their dependencies.
type Handler struct {
	repo Repo
}

// New creates a Handler.
func New(repo Repo) *Handler {
	return &Handler{repo: repo}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeBody(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

// ── health ────────────────────────────────────────────────────────────────────

// Healthz  GET /healthz  — returns 200 when DB is reachable, 503 otherwise.
func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	resp := model.HealthResponse{Status: "ok", Database: "ok"}
	code := http.StatusOK

	if err := h.repo.Ping(r.Context()); err != nil {
		resp.Status = "degraded"
		resp.Database = "unreachable: " + err.Error()
		code = http.StatusServiceUnavailable
	}

	writeJSON(w, code, resp)
}

// ── tasks ─────────────────────────────────────────────────────────────────────

// ListTasks  GET /tasks
func (h *Handler) ListTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.repo.GetAll(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

// GetTask  GET /tasks/{id}
func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	task, err := h.repo.GetByID(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, task)
}

// CreateTask  POST /tasks
func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req model.CreateTaskRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	task, err := h.repo.Create(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, task)
}

// UpdateTask  PUT /tasks/{id}
func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req model.UpdateTaskRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	task, err := h.repo.Update(r.Context(), id, req)
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, task)
}

// DeleteTask  DELETE /tasks/{id}
func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.repo.Delete(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// BulkDeleteTasks  DELETE /tasks/bulk
func (h *Handler) BulkDeleteTasks(w http.ResponseWriter, r *http.Request) {
	var req model.BulkDeleteRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if len(req.IDs) == 0 {
		writeError(w, http.StatusBadRequest, "ids must not be empty")
		return
	}

	affected, err := h.repo.BulkDelete(r.Context(), req.IDs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"deleted": affected})
}
