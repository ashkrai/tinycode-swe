package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// ── What is httptest? ─────────────────────────────────────────────────
//
// httptest lets you call your handlers directly in tests,
// without starting a real server.
//
// httptest.NewRecorder()  →  a fake ResponseWriter that records
//                            status code and body in memory
// httptest.NewRequest()   →  a fake *http.Request you construct yourself
//
// You call your handler with these fakes, then check what was recorded.
// No network, no port, instant.

// ── Test helpers ──────────────────────────────────────────────────────

// newTestRouter builds the full router wired to a fresh service.
// Each test gets a clean, empty store this way.
func newTestRouter() http.Handler {
	svc := NewTaskService()
	h := NewTaskHandler(svc)

	r := chi.NewRouter()
	r.Get("/tasks", h.List)
	r.Post("/tasks", h.Create)
	r.Get("/tasks/{id}", h.Get)
	r.Put("/tasks/{id}", h.Update)
	r.Delete("/tasks/{id}", h.Delete)
	return r
}

// do fires a request against the router and returns the recorder.
func do(router http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	var req *http.Request
	if body != nil {
		b, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// ── Table-driven tests ────────────────────────────────────────────────
//
// Table-driven means: define a slice of test cases (the "table"),
// then loop over them running the same test logic each time.
//
// Why?  Instead of writing 10 separate test functions that look almost
// identical, you write the logic once and vary only the inputs and
// expected outputs.  Adding a new case = adding one line to the table.

func TestCreateTask(t *testing.T) {
	tests := []struct {
		name       string          // what this test case is checking
		body       map[string]any  // request body we send
		wantStatus int             // HTTP status we expect back
		wantTitle  string          // title in the response (empty = don't check)
	}{
		{
			name:       "valid task",
			body:       map[string]any{"title": "Buy milk"},
			wantStatus: http.StatusCreated, // 201
			wantTitle:  "Buy milk",
		},
		{
			name:       "missing title",
			body:       map[string]any{"title": ""},
			wantStatus: http.StatusBadRequest, // 400
		},
		{
			name:       "empty body",
			body:       map[string]any{},
			wantStatus: http.StatusBadRequest, // 400
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router := newTestRouter()
			rr := do(router, http.MethodPost, "/tasks", tc.body)

			// Check status code.
			if rr.Code != tc.wantStatus {
				t.Errorf("got status %d, want %d", rr.Code, tc.wantStatus)
			}

			// If we expect a title, decode the response and check it.
			if tc.wantTitle != "" {
				var task Task
				json.NewDecoder(rr.Body).Decode(&task)
				if task.Title != tc.wantTitle {
					t.Errorf("got title %q, want %q", task.Title, tc.wantTitle)
				}
			}
		})
	}
}

func TestGetTask(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{
			name:       "task exists",
			path:       "/tasks/1",
			wantStatus: http.StatusOK, // 200
		},
		{
			name:       "task not found",
			path:       "/tasks/999",
			wantStatus: http.StatusNotFound, // 404
		},
		{
			name:       "bad id",
			path:       "/tasks/abc",
			wantStatus: http.StatusBadRequest, // 400
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router := newTestRouter()

			// Create a task first so ID 1 exists.
			do(router, http.MethodPost, "/tasks", map[string]any{"title": "Test task"})

			rr := do(router, http.MethodGet, tc.path, nil)

			if rr.Code != tc.wantStatus {
				t.Errorf("got status %d, want %d", rr.Code, tc.wantStatus)
			}
		})
	}
}

func TestDeleteTask(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{
			name:       "delete existing task",
			path:       "/tasks/1",
			wantStatus: http.StatusNoContent, // 204
		},
		{
			name:       "delete non-existent task",
			path:       "/tasks/999",
			wantStatus: http.StatusNotFound, // 404
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router := newTestRouter()

			// Create a task so ID 1 exists.
			do(router, http.MethodPost, "/tasks", map[string]any{"title": "Test task"})

			rr := do(router, http.MethodDelete, tc.path, nil)

			if rr.Code != tc.wantStatus {
				t.Errorf("got status %d, want %d", rr.Code, tc.wantStatus)
			}
		})
	}
}

func TestListTasks(t *testing.T) {
	router := newTestRouter()

	// Empty list at start.
	rr := do(router, http.MethodGet, "/tasks", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var tasks []Task
	json.NewDecoder(rr.Body).Decode(&tasks)
	if len(tasks) != 0 {
		t.Errorf("expected empty list, got %d tasks", len(tasks))
	}

	// Create two tasks, then list again.
	do(router, http.MethodPost, "/tasks", map[string]any{"title": "Task A"})
	do(router, http.MethodPost, "/tasks", map[string]any{"title": "Task B"})

	rr = do(router, http.MethodGet, "/tasks", nil)
	json.NewDecoder(rr.Body).Decode(&tasks)
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tasks))
	}
}

// TestRecoveryMiddleware proves the recovery middleware catches panics.
func TestRecoveryMiddleware(t *testing.T) {
	// Build a router with a handler that always panics.
	r := chi.NewRouter()
	r.Use(Recovery)
	r.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("something exploded")
	})

	rr := do(r, http.MethodGet, "/panic", nil)

	// Should get 500, not a crashed process.
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 after panic, got %d", rr.Code)
	}
}
