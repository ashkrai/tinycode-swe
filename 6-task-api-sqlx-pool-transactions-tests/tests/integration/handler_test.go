package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ashkrai/taskapi/internal/handler"
	"github.com/ashkrai/taskapi/internal/model"
	"github.com/ashkrai/taskapi/internal/repository"
)

// newTestServer spins up a real HTTP test server backed by the test DB.
func newTestServer(t *testing.T) (*httptest.Server, *repository.TaskRepository) {
	t.Helper()
	repo := repository.New(testDB(t))
	h := handler.New(repo)
	srv := httptest.NewServer(handler.NewRouter(h))
	t.Cleanup(srv.Close)
	return srv, repo
}

func doJSON(t *testing.T, method, url string, body any) *http.Response {
	t.Helper()
	var b []byte
	if body != nil {
		b, _ = json.Marshal(body)
	}
	req, _ := http.NewRequest(method, url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return resp
}

func assertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		t.Errorf("HTTP status: want %d, got %d", want, resp.StatusCode)
	}
}

func decodeTask(t *testing.T, resp *http.Response) model.Task {
	t.Helper()
	defer resp.Body.Close()
	var task model.Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		t.Fatalf("decode task: %v", err)
	}
	return task
}

// ── GET /healthz ──────────────────────────────────────────────────────────────

func TestHandler_Healthz(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	assertStatus(t, resp, http.StatusOK)

	var h model.HealthResponse
	json.NewDecoder(resp.Body).Decode(&h)
	if h.Status != "ok" || h.Database != "ok" {
		t.Errorf("unexpected health payload: %+v", h)
	}
}

// ── POST /tasks ───────────────────────────────────────────────────────────────

func TestHandler_CreateTask_201(t *testing.T) {
	srv, _ := newTestServer(t)
	resp := doJSON(t, http.MethodPost, srv.URL+"/tasks", map[string]string{
		"title": "Buy milk", "description": "2L", "status": "pending",
	})
	assertStatus(t, resp, http.StatusCreated)
	task := decodeTask(t, resp)
	if task.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestHandler_CreateTask_MissingTitle(t *testing.T) {
	srv, _ := newTestServer(t)
	resp := doJSON(t, http.MethodPost, srv.URL+"/tasks", map[string]string{"description": "no title"})
	resp.Body.Close()
	assertStatus(t, resp, http.StatusBadRequest)
}

// ── GET /tasks ────────────────────────────────────────────────────────────────

func TestHandler_ListTasks(t *testing.T) {
	srv, repo := newTestServer(t)
	repo.Create(context.Background(), model.CreateTaskRequest{Title: "T1"})
	repo.Create(context.Background(), model.CreateTaskRequest{Title: "T2"})

	resp, _ := http.Get(srv.URL + "/tasks")
	defer resp.Body.Close()
	assertStatus(t, resp, http.StatusOK)

	var tasks []model.Task
	json.NewDecoder(resp.Body).Decode(&tasks)
	if len(tasks) != 2 {
		t.Errorf("want 2, got %d", len(tasks))
	}
}

// ── GET /tasks/{id} ───────────────────────────────────────────────────────────

func TestHandler_GetTask_Found(t *testing.T) {
	srv, repo := newTestServer(t)
	task, _ := repo.Create(context.Background(), model.CreateTaskRequest{Title: "Fetchable"})
	resp, _ := http.Get(srv.URL + "/tasks/" + task.ID)
	resp.Body.Close()
	assertStatus(t, resp, http.StatusOK)
}

func TestHandler_GetTask_NotFound(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, _ := http.Get(srv.URL + "/tasks/no-such-id")
	resp.Body.Close()
	assertStatus(t, resp, http.StatusNotFound)
}

// ── PUT /tasks/{id} ───────────────────────────────────────────────────────────

func TestHandler_UpdateTask(t *testing.T) {
	srv, repo := newTestServer(t)
	task, _ := repo.Create(context.Background(), model.CreateTaskRequest{Title: "Old"})

	resp := doJSON(t, http.MethodPut, srv.URL+"/tasks/"+task.ID,
		map[string]string{"title": "New", "status": "done"})
	assertStatus(t, resp, http.StatusOK)

	updated := decodeTask(t, resp)
	if updated.Title != "New" {
		t.Errorf("title: want New, got %q", updated.Title)
	}
}

// ── DELETE /tasks/{id} ────────────────────────────────────────────────────────

func TestHandler_DeleteTask_204(t *testing.T) {
	srv, repo := newTestServer(t)
	task, _ := repo.Create(context.Background(), model.CreateTaskRequest{Title: "Doomed"})
	resp := doJSON(t, http.MethodDelete, srv.URL+"/tasks/"+task.ID, nil)
	resp.Body.Close()
	assertStatus(t, resp, http.StatusNoContent)
}

func TestHandler_DeleteTask_NotFound(t *testing.T) {
	srv, _ := newTestServer(t)
	resp := doJSON(t, http.MethodDelete, srv.URL+"/tasks/ghost", nil)
	resp.Body.Close()
	assertStatus(t, resp, http.StatusNotFound)
}

// ── DELETE /tasks/bulk ────────────────────────────────────────────────────────

func TestHandler_BulkDelete(t *testing.T) {
	srv, repo := newTestServer(t)
	a, _ := repo.Create(context.Background(), model.CreateTaskRequest{Title: "A"})
	b, _ := repo.Create(context.Background(), model.CreateTaskRequest{Title: "B"})
	c, _ := repo.Create(context.Background(), model.CreateTaskRequest{Title: "C"})

	resp := doJSON(t, http.MethodDelete, srv.URL+"/tasks/bulk",
		model.BulkDeleteRequest{IDs: []string{a.ID, b.ID}})
	assertStatus(t, resp, http.StatusOK)

	var result map[string]int64
	json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()

	if result["deleted"] != 2 {
		t.Errorf("deleted: want 2, got %d", result["deleted"])
	}

	// C must survive.
	getResp, _ := http.Get(srv.URL + "/tasks/" + c.ID)
	getResp.Body.Close()
	assertStatus(t, getResp, http.StatusOK)
}

func TestHandler_BulkDelete_EmptyIDs(t *testing.T) {
	srv, _ := newTestServer(t)
	resp := doJSON(t, http.MethodDelete, srv.URL+"/tasks/bulk",
		model.BulkDeleteRequest{IDs: []string{}})
	resp.Body.Close()
	assertStatus(t, resp, http.StatusBadRequest)
}
