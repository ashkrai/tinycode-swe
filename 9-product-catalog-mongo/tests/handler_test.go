package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/ashkrai/product-catalog/internal/handler"
	"github.com/ashkrai/product-catalog/internal/model"
	"github.com/ashkrai/product-catalog/internal/service"
)

// ── stub service ─────────────────────────────────────────────────────────────

type stubService struct {
	listResult     []model.Product
	getResult      *model.Product
	getErr         error
	createResult   *model.Product
	summaryResult  []model.CategorySummary
}

func (s *stubService) ListProducts(_ context.Context, _ model.ListFilter) ([]model.Product, error) {
	return s.listResult, nil
}
func (s *stubService) GetProduct(_ context.Context, _ string) (*model.Product, error) {
	return s.getResult, s.getErr
}
func (s *stubService) CreateProduct(_ context.Context, req model.CreateProductRequest) (*model.Product, error) {
	if s.createResult != nil {
		return s.createResult, nil
	}
	return &model.Product{
		ID:       primitive.NewObjectID(),
		Name:     req.Name,
		Category: req.Category,
		Price:    req.Price,
	}, nil
}
func (s *stubService) UpdateProduct(_ context.Context, id string, req model.CreateProductRequest) (*model.Product, error) {
	return &model.Product{Name: req.Name, Category: req.Category}, nil
}
func (s *stubService) DeleteProduct(_ context.Context, _ string) error  { return nil }
func (s *stubService) BulkDeleteProducts(_ context.Context, ids []string) (int64, error) {
	return int64(len(ids)), nil
}
func (s *stubService) CategorySummary(_ context.Context) ([]model.CategorySummary, error) {
	return s.summaryResult, nil
}
func (s *stubService) DBHealthy(_ context.Context) bool { return true }

var _ service.ProductServiceIface = (*stubService)(nil)

// ── router helper ─────────────────────────────────────────────────────────────

func buildRouter(svc service.ProductServiceIface) *chi.Mux {
	ph := handler.NewProductHandler(svc)
	hh := handler.NewHealthzHandler(svc)
	r := chi.NewRouter()
	r.Get("/healthz", hh.Healthz)
	r.Route("/products", func(r chi.Router) {
		r.Get("/analytics/categories", ph.CategorySummary)
		r.Get("/", ph.ListProducts)
		r.Post("/", ph.CreateProduct)
		r.Delete("/bulk", ph.BulkDeleteProducts)
		r.Get("/{id}", ph.GetProduct)
		r.Put("/{id}", ph.UpdateProduct)
		r.Delete("/{id}", ph.DeleteProduct)
	})
	return r
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestHealthz(t *testing.T) {
	r := buildRouter(&stubService{})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
}

func TestCreateProduct_Validation(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"empty body", `{}`, http.StatusUnprocessableEntity},
		{"missing category", `{"name":"widget","price":9.99}`, http.StatusUnprocessableEntity},
		{"missing name", `{"category":"electronics","price":9.99}`, http.StatusUnprocessableEntity},
		{"negative price", `{"name":"w","category":"c","price":-1}`, http.StatusUnprocessableEntity},
		{"negative stock", `{"name":"w","category":"c","price":1,"stock":-1}`, http.StatusUnprocessableEntity},
		{"invalid JSON", `not-json`, http.StatusBadRequest},
		{"empty body string", ``, http.StatusBadRequest},
		{"valid minimal", `{"name":"Widget","category":"tools","price":0}`, http.StatusCreated},
		{"valid full", `{"name":"Pro Widget","category":"tools","price":29.99,"stock":100,"tags":["sale"],"attributes":{"color":"red"}}`, http.StatusCreated},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := buildRouter(&stubService{})
			req := httptest.NewRequest(http.MethodPost, "/products/",
				bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.wantStatus {
				t.Errorf("want %d, got %d — body: %s", tc.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestListProducts_ReturnsJSON(t *testing.T) {
	products := []model.Product{
		{ID: primitive.NewObjectID(), Name: "Widget A", Category: "tools", Price: 9.99},
		{ID: primitive.NewObjectID(), Name: "Widget B", Category: "tools", Price: 19.99},
	}
	r := buildRouter(&stubService{listResult: products})
	req := httptest.NewRequest(http.MethodGet, "/products/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var got []model.Product
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("want 2 products, got %d", len(got))
	}
}

func TestListProducts_QueryFilters(t *testing.T) {
	r := buildRouter(&stubService{listResult: []model.Product{}})

	tests := []struct {
		name  string
		query string
	}{
		{"by category", "/products/?category=electronics"},
		{"by tag", "/products/?tag=sale"},
		{"by min_price", "/products/?min_price=10.00"},
		{"by max_price", "/products/?max_price=50.00"},
		{"combined", "/products/?category=tools&min_price=5&max_price=100"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.query, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("want 200, got %d for %s", w.Code, tc.query)
			}
		})
	}
}

func TestGetProduct_InvalidID(t *testing.T) {
	r := buildRouter(&stubService{})
	tests := []struct {
		id         string
		wantStatus int
	}{
		{"abc", http.StatusBadRequest},
		{"not-an-objectid", http.StatusBadRequest},
		{"123", http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.id, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/products/"+tc.id, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.wantStatus {
				t.Errorf("id=%s: want %d, got %d", tc.id, tc.wantStatus, w.Code)
			}
		})
	}
}

func TestDeleteProduct_InvalidID(t *testing.T) {
	r := buildRouter(&stubService{})
	req := httptest.NewRequest(http.MethodDelete, "/products/badid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestDeleteProduct_ValidID(t *testing.T) {
	r := buildRouter(&stubService{})
	validID := primitive.NewObjectID().Hex()
	req := httptest.NewRequest(http.MethodDelete, "/products/"+validID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("want 204, got %d", w.Code)
	}
}

func TestBulkDelete_Validation(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"empty ids", `{"ids":[]}`, http.StatusUnprocessableEntity},
		{"missing ids", `{}`, http.StatusUnprocessableEntity},
		{"invalid JSON", `bad`, http.StatusBadRequest},
		{"valid", `{"ids":["507f1f77bcf86cd799439011","507f1f77bcf86cd799439012"]}`, http.StatusOK},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := buildRouter(&stubService{})
			req := httptest.NewRequest(http.MethodDelete, "/products/bulk",
				bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.wantStatus {
				t.Errorf("[%s] want %d, got %d — body: %s", tc.name, tc.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestCategorySummary(t *testing.T) {
	summaries := []model.CategorySummary{
		{Category: "electronics", ProductCount: 10, AveragePrice: 299.99, TotalStock: 500},
		{Category: "tools", ProductCount: 5, AveragePrice: 49.99, TotalStock: 200},
	}
	r := buildRouter(&stubService{summaryResult: summaries})
	req := httptest.NewRequest(http.MethodGet, "/products/analytics/categories", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var got []model.CategorySummary
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("want 2 summaries, got %d", len(got))
	}
	if got[0].Category != "electronics" {
		t.Errorf("want electronics first, got %s", got[0].Category)
	}
}

func TestUpdateProduct_Validation(t *testing.T) {
	validID := primitive.NewObjectID().Hex()
	r := buildRouter(&stubService{})

	tests := []struct {
		name       string
		id         string
		body       string
		wantStatus int
	}{
		{"invalid id", "badid", `{"name":"x","category":"c","price":1}`, http.StatusBadRequest},
		{"empty body", validID, `{}`, http.StatusUnprocessableEntity},
		{"invalid JSON", validID, `bad`, http.StatusBadRequest},
		{"valid", validID, `{"name":"Updated","category":"tools","price":19.99}`, http.StatusOK},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/products/"+tc.id,
				bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.wantStatus {
				t.Errorf("[%s] want %d, got %d — %s", tc.name, tc.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}
