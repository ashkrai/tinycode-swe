package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// ── Handlers ──────────────────────────────────────────────────────────
//
// Handlers are now THIN.  Each one does only three things:
//   1. Read the request  (parse URL params, decode JSON body)
//   2. Call the service  (where the actual logic lives)
//   3. Write the response (encode JSON, set status code)
//
// No business logic here.  No map access.  No validation.
// That all lives in service.go.

// TaskHandler holds a reference to the service.
// Every handler method is attached to this struct.
type TaskHandler struct {
	svc *TaskService
}

func NewTaskHandler(svc *TaskService) *TaskHandler {
	return &TaskHandler{svc: svc}
}

// ── helper ────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// errorResponse maps our service errors to HTTP status codes.
// Handlers call this instead of writing status codes manually.
func errorResponse(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
	case errors.Is(err, ErrTitleMissing):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "something went wrong"})
	}
}

// ── GET /tasks ────────────────────────────────────────────────────────

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	tasks := h.svc.List()
	writeJSON(w, http.StatusOK, tasks)
}

// ── POST /tasks ───────────────────────────────────────────────────────

// createRequest is the shape we expect in the request body.
// Using a dedicated type means we clearly separate
// "what the client sends" from "what we store".
type createRequest struct {
	Title string `json:"title"`
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	task, err := h.svc.Create(req.Title)
	if err != nil {
		errorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, task)
}

// ── GET /tasks/{id} ───────────────────────────────────────────────────

func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id must be a number"})
		return
	}

	task, err := h.svc.Get(id)
	if err != nil {
		errorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusOK, task)
}

// ── PUT /tasks/{id} ───────────────────────────────────────────────────

type updateRequest struct {
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id must be a number"})
		return
	}

	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	task, err := h.svc.Update(id, req.Title, req.Done)
	if err != nil {
		errorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusOK, task)
}

// ── DELETE /tasks/{id} ────────────────────────────────────────────────

func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id must be a number"})
		return
	}

	if err := h.svc.Delete(id); err != nil {
		errorResponse(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
