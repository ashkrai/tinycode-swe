package tests

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ashkrai/blog-api/internal/handler"
	"github.com/ashkrai/blog-api/internal/model"
	"github.com/ashkrai/blog-api/internal/service"
	"github.com/go-chi/chi/v5"
)

// ---- stub service ----

type stubService struct {
	listResult []model.Post
	getResult  *model.Post
	getErr     error
}

func (s *stubService) ListPosts(_ context.Context) ([]model.Post, error) {
	return s.listResult, nil
}
func (s *stubService) GetPost(_ context.Context, _ int) (*model.Post, error) {
	return s.getResult, s.getErr
}
func (s *stubService) CreatePost(_ context.Context, req model.CreatePostRequest) (*model.Post, error) {
	return &model.Post{ID: 99, Title: req.Title}, nil
}
func (s *stubService) UpdatePost(_ context.Context, id int, req model.CreatePostRequest) (*model.Post, error) {
	return &model.Post{ID: id, Title: req.Title}, nil
}
func (s *stubService) DeletePost(_ context.Context, _ int) error { return nil }
func (s *stubService) BulkDeletePosts(_ context.Context, ids []int) (int64, error) {
	return int64(len(ids)), nil
}
func (s *stubService) CacheHealthy(_ context.Context) bool { return true }

var _ service.PostServiceIface = (*stubService)(nil)

// ---- router helper ----

func setupRouter(ph *handler.PostHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/posts", ph.ListPosts)
	r.Post("/posts", ph.CreatePost)
	r.Delete("/posts/bulk", ph.BulkDeletePosts)
	r.Get("/posts/{id}", ph.GetPost)
	r.Put("/posts/{id}", ph.UpdatePost)
	r.Delete("/posts/{id}", ph.DeletePost)
	return r
}

// ---- table-driven tests ----

func TestCreatePostValidation(t *testing.T) {
	longTitle := make([]byte, 256)
	for i := range longTitle {
		longTitle[i] = 'a'
	}

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"empty body", `{}`, http.StatusUnprocessableEntity},
		{"missing title", `{"user_id":1,"body":"hi"}`, http.StatusUnprocessableEntity},
		{"missing body", `{"user_id":1,"title":"hi"}`, http.StatusUnprocessableEntity},
		{"zero user_id", `{"user_id":0,"title":"t","body":"b"}`, http.StatusUnprocessableEntity},
		{"title too long", `{"user_id":1,"title":"` + string(longTitle) + `","body":"b"}`, http.StatusUnprocessableEntity},
		{"invalid JSON", `not-json`, http.StatusBadRequest},
		{"valid", `{"user_id":1,"title":"Hello","body":"World"}`, http.StatusCreated},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &stubService{}
			ph := handler.NewPostHandlerFromService(svc)
			r := setupRouter(ph)

			req := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.wantStatus {
				t.Errorf("want %d, got %d — body: %s", tc.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestBulkDeleteValidation(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"empty ids array", `{"ids":[]}`, http.StatusUnprocessableEntity},
		{"invalid JSON", `{bad}`, http.StatusBadRequest},
		{"valid single", `{"ids":[1]}`, http.StatusOK},
		{"valid multiple", `{"ids":[1,2,3]}`, http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &stubService{}
			ph := handler.NewPostHandlerFromService(svc)
			r := setupRouter(ph)

			req := httptest.NewRequest(http.MethodDelete, "/posts/bulk", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.wantStatus {
				t.Errorf("[%s] want %d, got %d — body: %s", tc.name, tc.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestListPostsReturnsJSON(t *testing.T) {
	svc := &stubService{listResult: []model.Post{{ID: 1, Title: "Hello"}}}
	ph := handler.NewPostHandlerFromService(svc)
	r := setupRouter(ph)

	req := httptest.NewRequest(http.MethodGet, "/posts", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var posts []model.Post
	if err := json.NewDecoder(w.Body).Decode(&posts); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(posts) != 1 || posts[0].ID != 1 {
		t.Errorf("unexpected posts: %+v", posts)
	}
}

func TestGetPostNotFound(t *testing.T) {
	svc := &stubService{getErr: sql.ErrNoRows}
	ph := handler.NewPostHandlerFromService(svc)
	r := setupRouter(ph)

	req := httptest.NewRequest(http.MethodGet, "/posts/999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

func TestDeletePostReturns204(t *testing.T) {
	svc := &stubService{}
	ph := handler.NewPostHandlerFromService(svc)
	r := setupRouter(ph)

	req := httptest.NewRequest(http.MethodDelete, "/posts/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("want 204, got %d", w.Code)
	}
}

func TestUpdatePostInvalidID(t *testing.T) {
	svc := &stubService{}
	ph := handler.NewPostHandlerFromService(svc)
	r := setupRouter(ph)

	req := httptest.NewRequest(http.MethodPut, "/posts/abc", bytes.NewBufferString(`{"user_id":1,"title":"t","body":"b"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}
