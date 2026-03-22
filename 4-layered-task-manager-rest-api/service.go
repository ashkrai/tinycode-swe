package main

import (
	"errors"
	"sync"
)

// ── Service layer ─────────────────────────────────────────────────────
//
// Previously all the map logic lived inside the handlers.
// Now we pull it into a TaskService struct.
//
// Why?
//   - Handlers only deal with HTTP (read request, write response).
//   - Service deals with business logic (create task, validate, store).
//   - This makes each piece easy to test independently.
//     You can test the service with no HTTP at all.
//
// The handler calls the service.
// The service does not know anything about HTTP.

// Sentinel errors — named errors the handler can check against.
// Instead of comparing error strings ("task not found") we compare
// the error value itself.  Much safer.
var (
	ErrNotFound     = errors.New("task not found")
	ErrTitleMissing = errors.New("title is required")
)

type TaskService struct {
	mu     sync.Mutex
	tasks  map[int]Task
	nextID int
}

func NewTaskService() *TaskService {
	return &TaskService{
		tasks:  map[int]Task{},
		nextID: 1,
	}
}

// List returns all tasks as a slice.
func (s *TaskService) List() []Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	list := make([]Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		list = append(list, t)
	}
	return list
}

// Create validates and stores a new task.
// Returns the created task (with its new ID) or an error.
func (s *TaskService) Create(title string) (Task, error) {
	// Validation lives here — not in the handler.
	// If you add more rules later (max length, forbidden words, etc.)
	// you change this one place, not every handler.
	if title == "" {
		return Task{}, ErrTitleMissing
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	t := Task{
		ID:    s.nextID,
		Title: title,
		Done:  false,
	}
	s.tasks[s.nextID] = t
	s.nextID++
	return t, nil
}

// Get returns a single task by ID.
func (s *TaskService) Get(id int) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, exists := s.tasks[id]
	if !exists {
		return Task{}, ErrNotFound
	}
	return t, nil
}

// Update replaces a task's title and done status.
func (s *TaskService) Update(id int, title string, done bool) (Task, error) {
	if title == "" {
		return Task{}, ErrTitleMissing
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.tasks[id]
	if !exists {
		return Task{}, ErrNotFound
	}

	updated := Task{ID: id, Title: title, Done: done}
	s.tasks[id] = updated
	return updated, nil
}

// Delete removes a task by ID.
func (s *TaskService) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.tasks[id]
	if !exists {
		return ErrNotFound
	}

	delete(s.tasks, id)
	return nil
}
