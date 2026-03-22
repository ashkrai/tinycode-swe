package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter registers all routes on a chi.Mux and returns it as http.Handler.
func NewRouter(h *Handler) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Health check
	r.Get("/healthz", h.Healthz)

	// Tasks
	r.Route("/tasks", func(r chi.Router) {
		r.Get("/", h.ListTasks)
		r.Post("/", h.CreateTask)

		// /bulk must come before /{id} so chi does not treat the
		// literal string "bulk" as a UUID-shaped path parameter.
		r.Delete("/bulk", h.BulkDeleteTasks)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.GetTask)
			r.Put("/", h.UpdateTask)
			r.Delete("/", h.DeleteTask)
		})
	})

	return r
}
