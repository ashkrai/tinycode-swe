package integration_test

import (
	"context"
	"testing"

	"github.com/ashkrai/taskapi/internal/model"
	"github.com/ashkrai/taskapi/internal/repository"
)

func seedTask(t *testing.T, repo *repository.TaskRepository, title, status string) model.Task {
	t.Helper()
	task, err := repo.Create(context.Background(), model.CreateTaskRequest{
		Title:       title,
		Description: "desc",
		Status:      model.Status(status),
	})
	if err != nil {
		t.Fatalf("seedTask %q: %v", title, err)
	}
	return task
}

func isNotFound(err error) bool { return err == repository.ErrNotFound }

// ── Ping ─────────────────────────────────────────────────────────────────────

func TestRepository_Ping(t *testing.T) {
	repo := repository.New(testDB(t))
	if err := repo.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

// ── GetAll ───────────────────────────────────────────────────────────────────

func TestRepository_GetAll_Empty(t *testing.T) {
	repo := repository.New(testDB(t))
	tasks, err := repo.GetAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 0 {
		t.Errorf("want 0, got %d", len(tasks))
	}
}

func TestRepository_GetAll_ReturnsTasks(t *testing.T) {
	repo := repository.New(testDB(t))
	seedTask(t, repo, "A", "pending")
	seedTask(t, repo, "B", "done")

	tasks, err := repo.GetAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 2 {
		t.Errorf("want 2, got %d", len(tasks))
	}
}

func TestRepository_GetAll_ExcludesSoftDeleted(t *testing.T) {
	repo := repository.New(testDB(t))
	task := seedTask(t, repo, "Gone", "pending")
	_ = repo.Delete(context.Background(), task.ID)

	tasks, _ := repo.GetAll(context.Background())
	if len(tasks) != 0 {
		t.Errorf("deleted task must not appear, got %d", len(tasks))
	}
}

// ── GetByID ───────────────────────────────────────────────────────────────────

func TestRepository_GetByID_Found(t *testing.T) {
	repo := repository.New(testDB(t))
	created := seedTask(t, repo, "Find me", "in_progress")

	got, err := repo.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != created.ID {
		t.Errorf("ID mismatch: want %q got %q", created.ID, got.ID)
	}
}

func TestRepository_GetByID_NotFound(t *testing.T) {
	repo := repository.New(testDB(t))
	_, err := repo.GetByID(context.Background(), "ghost")
	if !isNotFound(err) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestRepository_Create_AssignsID(t *testing.T) {
	repo := repository.New(testDB(t))
	task, err := repo.Create(context.Background(), model.CreateTaskRequest{Title: "ID test"})
	if err != nil {
		t.Fatal(err)
	}
	if task.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestRepository_Create_DefaultStatus(t *testing.T) {
	repo := repository.New(testDB(t))
	task, err := repo.Create(context.Background(), model.CreateTaskRequest{Title: "No status"})
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != model.StatusPending {
		t.Errorf("want pending, got %q", task.Status)
	}
}

// ── Update ────────────────────────────────────────────────────────────────────

func TestRepository_Update_PartialPatch(t *testing.T) {
	repo := repository.New(testDB(t))
	task := seedTask(t, repo, "Old", "pending")

	newTitle := "New"
	updated, err := repo.Update(context.Background(), task.ID, model.UpdateTaskRequest{Title: &newTitle})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "New" {
		t.Errorf("title: want New, got %q", updated.Title)
	}
	if updated.Status != model.StatusPending {
		t.Errorf("status must not change, got %q", updated.Status)
	}
}

func TestRepository_Update_NotFound(t *testing.T) {
	repo := repository.New(testDB(t))
	title := "x"
	_, err := repo.Update(context.Background(), "ghost", model.UpdateTaskRequest{Title: &title})
	if !isNotFound(err) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestRepository_Delete_SoftDeletes(t *testing.T) {
	repo := repository.New(testDB(t))
	task := seedTask(t, repo, "Bye", "done")
	_ = repo.Delete(context.Background(), task.ID)

	_, err := repo.GetByID(context.Background(), task.ID)
	if !isNotFound(err) {
		t.Errorf("want ErrNotFound after delete, got %v", err)
	}
}

func TestRepository_Delete_NotFound(t *testing.T) {
	repo := repository.New(testDB(t))
	if err := repo.Delete(context.Background(), "ghost"); !isNotFound(err) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

// ── BulkDelete ────────────────────────────────────────────────────────────────

func TestRepository_BulkDelete_DeletesSelected(t *testing.T) {
	repo := repository.New(testDB(t))
	a := seedTask(t, repo, "A", "pending")
	b := seedTask(t, repo, "B", "pending")
	c := seedTask(t, repo, "C", "pending")

	affected, err := repo.BulkDelete(context.Background(), []string{a.ID, b.ID})
	if err != nil {
		t.Fatal(err)
	}
	if affected != 2 {
		t.Errorf("want 2 affected, got %d", affected)
	}
	if _, err := repo.GetByID(context.Background(), c.ID); err != nil {
		t.Errorf("task C must still exist: %v", err)
	}
}

func TestRepository_BulkDelete_EmptySlice(t *testing.T) {
	repo := repository.New(testDB(t))
	affected, err := repo.BulkDelete(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if affected != 0 {
		t.Errorf("want 0, got %d", affected)
	}
}

func TestRepository_BulkDelete_SkipsAlreadyDeleted(t *testing.T) {
	repo := repository.New(testDB(t))
	task := seedTask(t, repo, "Pre-deleted", "pending")
	_ = repo.Delete(context.Background(), task.ID)

	affected, err := repo.BulkDelete(context.Background(), []string{task.ID})
	if err != nil {
		t.Fatal(err)
	}
	if affected != 0 {
		t.Errorf("want 0, got %d", affected)
	}
}
