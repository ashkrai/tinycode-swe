package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/go-chi/chi/v5"
)

// ── 1. The data ───────────────────────────────────────────────────────
//
// Task is the shape of our data.
// The `json:"..."` tags control what the field is called in JSON.
//
//   Go struct field   →   JSON key
//   ID                →   "id"
//   Title             →   "title"
//   Done              →   "done"

type Task struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

// ── 2. The in-memory store ────────────────────────────────────────────
//
// This is our "database" — just a Go map living in memory.
// Key = task ID (int), Value = Task.
//
// sync.Mutex protects the map from being read and written
// at the same time from different requests (Go handles each
// request in its own goroutine — concurrent access = data corruption
// without a lock).

var (
	tasks  = map[int]Task{}
	nextID = 1
	mu     sync.Mutex
)

// ── 3. Helpers ────────────────────────────────────────────────────────

// writeJSON converts any Go value to JSON and writes it to the response.
// Every handler uses this to send data back to the client.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// ── 4. Handlers ───────────────────────────────────────────────────────
//
// Each handler is a function: receives a Request, writes to ResponseWriter.
// Think of ResponseWriter as the "reply envelope" you fill in and send back.

// GET /tasks — return all tasks as a JSON array
func listTasks(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	// Convert the map to a slice so JSON comes out as an array [...].
	// A map encodes as an object {...} which isn't what we want here.
	list := make([]Task, 0, len(tasks))
	for _, t := range tasks {
		list = append(list, t)
	}

	writeJSON(w, http.StatusOK, list) // 200
}

// POST /tasks — create a new task from the JSON body
func createTask(w http.ResponseWriter, r *http.Request) {
	// Decode the JSON body the client sent into a Task struct.
	var t Task
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{ // 400
			"error": "invalid JSON",
		})
		return
	}

	if t.Title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{ // 400
			"error": "title is required",
		})
		return
	}

	mu.Lock()
	t.ID = nextID
	nextID++
	tasks[t.ID] = t
	mu.Unlock()

	writeJSON(w, http.StatusCreated, t) // 201 — "something was created"
}

// GET /tasks/{id} — return a single task by ID
func getTask(w http.ResponseWriter, r *http.Request) {
	// chi.URLParam pulls the {id} piece out of the URL path.
	// "/tasks/42" → chi.URLParam(r, "id") → "42"
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{ // 400
			"error": "id must be a number",
		})
		return
	}

	mu.Lock()
	t, exists := tasks[id]
	mu.Unlock()

	if !exists {
		writeJSON(w, http.StatusNotFound, map[string]string{ // 404
			"error": "task not found",
		})
		return
	}

	writeJSON(w, http.StatusOK, t) // 200
}

// PUT /tasks/{id} — replace a task's data
func updateTask(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "id must be a number",
		})
		return
	}

	var updated Task
	if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid JSON",
		})
		return
	}

	mu.Lock()
	defer mu.Unlock()

	_, exists := tasks[id]
	if !exists {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "task not found",
		})
		return
	}

	// Keep the original ID regardless of what the client sent.
	updated.ID = id
	tasks[id] = updated

	writeJSON(w, http.StatusOK, updated) // 200
}

// DELETE /tasks/{id} — remove a task
func deleteTask(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "id must be a number",
		})
		return
	}

	mu.Lock()
	defer mu.Unlock()

	_, exists := tasks[id]
	if !exists {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "task not found",
		})
		return
	}

	delete(tasks, id)
	w.WriteHeader(http.StatusNoContent) // 204 — "done, nothing to return"
}

// ── 5. Wiring it all together ─────────────────────────────────────────

func main() {
	r := chi.NewRouter()

	// Each line says: "when this METHOD + PATH comes in, call this function"
	r.Get("/tasks", listTasks)
	r.Post("/tasks", createTask)
	r.Get("/tasks/{id}", getTask)
	r.Put("/tasks/{id}", updateTask)
	r.Delete("/tasks/{id}", deleteTask)

	fmt.Println("Server running on http://localhost:8080")
	http.ListenAndServe(":8080", r)
}