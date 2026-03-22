package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/ashkrai/taskapi/internal/model"
)

// ErrNotFound is returned when a task does not exist or has been soft-deleted.
var ErrNotFound = errors.New("task not found")

// TaskRepository handles every SQL operation for the tasks table.
type TaskRepository struct {
	db *sqlx.DB
}

// New returns a TaskRepository backed by db.
func New(db *sqlx.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

// ── Read ──────────────────────────────────────────────────────────────────────

// GetAll returns every non-deleted task ordered newest-first.
func (r *TaskRepository) GetAll(ctx context.Context) ([]model.Task, error) {
	const q = `
		SELECT id, title, description, status, created_at, updated_at, deleted_at
		FROM   tasks
		WHERE  deleted_at IS NULL
		ORDER  BY created_at DESC`

	var tasks []model.Task
	if err := r.db.SelectContext(ctx, &tasks, q); err != nil {
		return nil, fmt.Errorf("repository.GetAll: %w", err)
	}
	return tasks, nil
}

// GetByID returns a single non-deleted task or ErrNotFound.
func (r *TaskRepository) GetByID(ctx context.Context, id string) (model.Task, error) {
	const q = `
		SELECT id, title, description, status, created_at, updated_at, deleted_at
		FROM   tasks
		WHERE  id = $1 AND deleted_at IS NULL`

	var t model.Task
	err := r.db.GetContext(ctx, &t, q, id)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Task{}, ErrNotFound
	}
	if err != nil {
		return model.Task{}, fmt.Errorf("repository.GetByID: %w", err)
	}
	return t, nil
}

// ── Write ─────────────────────────────────────────────────────────────────────

// Create inserts a new task and returns it with all server-assigned fields.
func (r *TaskRepository) Create(ctx context.Context, req model.CreateTaskRequest) (model.Task, error) {
	if req.Status == "" {
		req.Status = model.StatusPending
	}

	now := time.Now().UTC()
	t := model.Task{
		ID:          uuid.New().String(),
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	const q = `
		INSERT INTO tasks (id, title, description, status, created_at, updated_at)
		VALUES (:id, :title, :description, :status, :created_at, :updated_at)`

	if _, err := r.db.NamedExecContext(ctx, q, t); err != nil {
		return model.Task{}, fmt.Errorf("repository.Create: %w", err)
	}
	return t, nil
}

// Update applies a partial patch to an existing task.
func (r *TaskRepository) Update(ctx context.Context, id string, req model.UpdateTaskRequest) (model.Task, error) {
	t, err := r.GetByID(ctx, id)
	if err != nil {
		return model.Task{}, err
	}

	if req.Title != nil {
		t.Title = *req.Title
	}
	if req.Description != nil {
		t.Description = *req.Description
	}
	if req.Status != nil {
		t.Status = *req.Status
	}
	t.UpdatedAt = time.Now().UTC()

	const q = `
		UPDATE tasks
		SET    title       = :title,
		       description = :description,
		       status      = :status,
		       updated_at  = :updated_at
		WHERE  id = :id AND deleted_at IS NULL`

	res, err := r.db.NamedExecContext(ctx, q, t)
	if err != nil {
		return model.Task{}, fmt.Errorf("repository.Update: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.Task{}, ErrNotFound
	}
	return t, nil
}

// Delete soft-deletes a single task.
func (r *TaskRepository) Delete(ctx context.Context, id string) error {
	const q = `
		UPDATE tasks SET deleted_at = NOW()
		WHERE  id = $1 AND deleted_at IS NULL`

	res, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("repository.Delete: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ── Transactional bulk delete ─────────────────────────────────────────────────

// BulkDelete soft-deletes multiple tasks inside a single transaction.
// If any step fails, the entire batch is rolled back.
func (r *TaskRepository) BulkDelete(ctx context.Context, ids []string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	// sqlx.In expands the slice into positional ? placeholders.
	updateQ, args, err := sqlx.In(`
		UPDATE tasks SET deleted_at = NOW()
		WHERE  id IN (?) AND deleted_at IS NULL`, ids)
	if err != nil {
		return 0, fmt.Errorf("repository.BulkDelete build query: %w", err)
	}
	// Rebind converts ? → $N for the postgres driver.
	updateQ = r.db.Rebind(updateQ)

	// ── BEGIN ─────────────────────────────────────────────────────────────────
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return 0, fmt.Errorf("repository.BulkDelete begin tx: %w", err)
	}

	rollback := func(cause error) (int64, error) {
		_ = tx.Rollback()
		return 0, cause
	}

	// Lock target rows to prevent concurrent modification.
	lockQ := r.db.Rebind(buildLockQuery(len(ids)))
	if _, err = tx.ExecContext(ctx, lockQ, args...); err != nil {
		return rollback(fmt.Errorf("repository.BulkDelete lock rows: %w", err))
	}

	res, err := tx.ExecContext(ctx, updateQ, args...)
	if err != nil {
		return rollback(fmt.Errorf("repository.BulkDelete exec: %w", err))
	}

	affected, _ := res.RowsAffected()

	if err = tx.Commit(); err != nil {
		return rollback(fmt.Errorf("repository.BulkDelete commit: %w", err))
	}
	// ── COMMITTED ─────────────────────────────────────────────────────────────

	return affected, nil
}

// buildLockQuery returns SELECT … FOR UPDATE with n ? placeholders.
func buildLockQuery(n int) string {
	ph := make([]string, n)
	for i := range ph {
		ph[i] = "?"
	}
	return fmt.Sprintf(
		"SELECT id FROM tasks WHERE id IN (%s) AND deleted_at IS NULL FOR UPDATE",
		strings.Join(ph, ", "),
	)
}

// Ping verifies the underlying database is reachable.
func (r *TaskRepository) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}
