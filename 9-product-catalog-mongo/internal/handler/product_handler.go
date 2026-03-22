package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/ashkrai/product-catalog/internal/model"
	"github.com/ashkrai/product-catalog/internal/service"
)

// ProductHandler wires HTTP requests to the service layer.
type ProductHandler struct {
	svc service.ProductServiceIface
}

func NewProductHandler(svc service.ProductServiceIface) *ProductHandler {
	return &ProductHandler{svc: svc}
}

// ListProducts GET /products
// Query params: category, tag, min_price, max_price
func (h *ProductHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	f := model.ListFilter{
		Category: r.URL.Query().Get("category"),
		Tag:      r.URL.Query().Get("tag"),
	}
	if v := r.URL.Query().Get("min_price"); v != "" {
		if p, err := strconv.ParseFloat(v, 64); err == nil {
			f.MinPrice = p
		}
	}
	if v := r.URL.Query().Get("max_price"); v != "" {
		if p, err := strconv.ParseFloat(v, 64); err == nil {
			f.MaxPrice = p
		}
	}

	products, err := h.svc.ListProducts(r.Context(), f)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, products)
}

// GetProduct GET /products/:id
func (h *ProductHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isValidObjectID(id) {
		jsonError(w, "invalid product id", http.StatusBadRequest)
		return
	}
	product, err := h.svc.GetProduct(r.Context(), id)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) || isInvalidIDErr(err) {
			jsonError(w, "product not found", http.StatusNotFound)
			return
		}
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, product)
}

// CreateProduct POST /products
func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req model.CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if errs := req.Validate(); len(errs) > 0 {
		jsonValidationErrors(w, errs)
		return
	}
	product, err := h.svc.CreateProduct(r.Context(), req)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, product)
}

// UpdateProduct PUT /products/:id
func (h *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isValidObjectID(id) {
		jsonError(w, "invalid product id", http.StatusBadRequest)
		return
	}
	var req model.CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if errs := req.Validate(); len(errs) > 0 {
		jsonValidationErrors(w, errs)
		return
	}
	product, err := h.svc.UpdateProduct(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) || isInvalidIDErr(err) {
			jsonError(w, "product not found", http.StatusNotFound)
			return
		}
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, product)
}

// DeleteProduct DELETE /products/:id
func (h *ProductHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isValidObjectID(id) {
		jsonError(w, "invalid product id", http.StatusBadRequest)
		return
	}
	if err := h.svc.DeleteProduct(r.Context(), id); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) || isInvalidIDErr(err) {
			jsonError(w, "product not found", http.StatusNotFound)
			return
		}
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// BulkDeleteProducts DELETE /products/bulk
func (h *ProductHandler) BulkDeleteProducts(w http.ResponseWriter, r *http.Request) {
	var req model.BulkDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if errs := req.Validate(); len(errs) > 0 {
		jsonValidationErrors(w, errs)
		return
	}
	n, err := h.svc.BulkDeleteProducts(r.Context(), req.IDs)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonOK(w, map[string]int64{"deleted": n})
}

// CategorySummary GET /products/analytics/categories
func (h *ProductHandler) CategorySummary(w http.ResponseWriter, r *http.Request) {
	summaries, err := h.svc.CategorySummary(r.Context())
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, summaries)
}

// --- helpers ---

func isValidObjectID(id string) bool {
	return len(id) == 24
}

func isInvalidIDErr(err error) bool {
	return err != nil && (err.Error() == "invalid id" ||
		contains(err.Error(), "invalid id"))
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub ||
		len(s) > 0 && func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func jsonValidationErrors(w http.ResponseWriter, errs map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	_ = json.NewEncoder(w).Encode(map[string]any{"errors": errs})
}

// HealthzHandler handles GET /healthz
type HealthzHandler struct {
	svc service.ProductServiceIface
}

func NewHealthzHandler(svc service.ProductServiceIface) *HealthzHandler {
	return &HealthzHandler{svc: svc}
}

func (h *HealthzHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	dbOK := h.svc.DBHealthy(r.Context())
	status := http.StatusOK
	if !dbOK {
		status = http.StatusServiceUnavailable
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": fmt.Sprintf("%s", map[bool]string{true: "ok", false: "degraded"}[dbOK]),
		"db":     dbOK,
	})
}
