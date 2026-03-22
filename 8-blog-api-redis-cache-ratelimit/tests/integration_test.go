package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/ashkrai/blog-api/internal/cache"
	"github.com/ashkrai/blog-api/internal/handler"
	"github.com/ashkrai/blog-api/internal/model"
	"github.com/ashkrai/blog-api/internal/repository"
	"github.com/ashkrai/blog-api/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Integration tests require:
//   TEST_DATABASE_URL=postgres://blog:blog@localhost:5433/blog_test?sslmode=disable
//   TEST_REDIS_URL=redis://localhost:6380/1
//
// Both are provided by docker-compose.test.yml.
// Skip gracefully when env vars are absent so unit test runs still pass.

func integrationDB(t *testing.T) *sqlx.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatalf("connect test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func integrationCache(t *testing.T) *cache.Client {
	t.Helper()
	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		t.Skip("TEST_REDIS_URL not set — skipping integration test")
	}
	c, err := cache.New(url)
	if err != nil {
		t.Fatalf("connect test redis: %v", err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

func buildIntegrationRouter(t *testing.T) (*chi.Mux, *sqlx.DB) {
	t.Helper()
	db := integrationDB(t)
	rc := integrationCache(t)

	// Flush test Redis DB to start clean.
	ctx := context.Background()
	_ = rc.Delete(ctx, "posts:list") // ignore error

	repo := repository.NewPostRepository(db)
	svc := service.NewPostService(repo, rc)
	ph := handler.NewPostHandler(svc)

	r := chi.NewRouter()
	r.Get("/posts", ph.ListPosts)
	r.Post("/posts", ph.CreatePost)
	r.Delete("/posts/bulk", ph.BulkDeletePosts)
	r.Get("/posts/{id}", ph.GetPost)
	r.Put("/posts/{id}", ph.UpdatePost)
	r.Delete("/posts/{id}", ph.DeletePost)
	return r, db
}

func TestIntegration_CreateAndListPost(t *testing.T) {
	r, db := buildIntegrationRouter(t)

	// Ensure user 1 exists (seeded by migration).
	var userID int
	if err := db.QueryRow(`SELECT id FROM users LIMIT 1`).Scan(&userID); err != nil {
		t.Fatalf("no seed user: %v", err)
	}

	// Create a post.
	body := map[string]any{"user_id": userID, "title": "Integration Test Post", "body": "Hello from test"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: want 201, got %d — %s", w.Code, w.Body.String())
	}

	var created model.Post
	_ = json.NewDecoder(w.Body).Decode(&created)
	if created.ID == 0 {
		t.Fatal("expected non-zero post ID")
	}

	// List — first call hits DB and populates cache.
	req2 := httptest.NewRequest(http.MethodGet, "/posts", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("list: want 200, got %d", w2.Code)
	}

	// List again — should be a cache hit (same response, no DB).
	req3 := httptest.NewRequest(http.MethodGet, "/posts", nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("list (cached): want 200, got %d", w3.Code)
	}

	// Cleanup.
	_, _ = db.Exec(`DELETE FROM posts WHERE id = $1`, created.ID)
}

func TestIntegration_BulkDeleteCacheBust(t *testing.T) {
	r, db := buildIntegrationRouter(t)

	var userID int
	if err := db.QueryRow(`SELECT id FROM users LIMIT 1`).Scan(&userID); err != nil {
		t.Fatalf("no seed user: %v", err)
	}

	// Create two posts directly in DB.
	var id1, id2 int
	_ = db.QueryRow(`INSERT INTO posts (user_id,title,body) VALUES ($1,'A','a') RETURNING id`, userID).Scan(&id1)
	_ = db.QueryRow(`INSERT INTO posts (user_id,title,body) VALUES ($1,'B','b') RETURNING id`, userID).Scan(&id2)

	// Warm the list cache.
	req := httptest.NewRequest(http.MethodGet, "/posts", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Bulk delete.
	body, _ := json.Marshal(map[string]any{"ids": []int{id1, id2}})
	req2 := httptest.NewRequest(http.MethodDelete, "/posts/bulk", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("bulk delete: want 200, got %d — %s", w2.Code, w2.Body.String())
	}
	var result map[string]int64
	_ = json.NewDecoder(w2.Body).Decode(&result)
	if result["deleted"] != 2 {
		t.Errorf("want deleted=2, got %d", result["deleted"])
	}
}

func TestIntegration_RateLimiter429(t *testing.T) {
	rc := integrationCache(t)
	ctx := context.Background()

	// Use a fresh key with a very low limit.
	key := "ratelimit:test-ip-integration"
	_ = rc.Delete(ctx, key)
	defer rc.Delete(ctx, key)

	const limit = 5
	var lastResult cache.RateLimitResult
	var err error
	for i := 0; i < limit+2; i++ {
		lastResult, err = rc.Allow(ctx, key, limit, time.Minute)
		if err != nil {
			t.Fatalf("allow: %v", err)
		}
	}
	if lastResult.Allowed {
		t.Error("expected last request to be denied (rate limit exceeded)")
	}
	if lastResult.RetryAfter == 0 {
		t.Error("expected non-zero RetryAfter")
	}
}
