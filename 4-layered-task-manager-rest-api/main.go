package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {
	// Wire everything together.
	// main.go is now tiny — it just connects the pieces.
	svc := NewTaskService()
	h := NewTaskHandler(svc)

	r := chi.NewRouter()

	// Attach middleware.
	// Order matters: Recovery wraps Logger wraps the handlers.
	// So Recovery catches any panic from Logger or the handlers.
	r.Use(Recovery)
	r.Use(Logger)

	r.Get("/tasks", h.List)
	r.Post("/tasks", h.Create)
	r.Get("/tasks/{id}", h.Get)
	r.Put("/tasks/{id}", h.Update)
	r.Delete("/tasks/{id}", h.Delete)

	fmt.Println("Server running on http://localhost:8080")
	http.ListenAndServe(":8080", r)
}
