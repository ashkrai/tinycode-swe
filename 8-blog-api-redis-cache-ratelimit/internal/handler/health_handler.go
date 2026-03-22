package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/ashkrai/blog-api/internal/service"
	"github.com/jmoiron/sqlx"
)

type HealthzHandler struct {
	db  *sqlx.DB
	svc *service.PostService
}

func NewHealthzHandler(db *sqlx.DB, svc *service.PostService) *HealthzHandler {
	return &HealthzHandler{db: db, svc: svc}
}

func (h *HealthzHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second)
	defer cancel()

	dbOK := h.db.PingContext(ctx) == nil
	cacheOK := h.svc.CacheHealthy(ctx)

	status := http.StatusOK
	if !dbOK || !cacheOK {
		status = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]bool{
		"db":    dbOK,
		"cache": cacheOK,
	})
}
