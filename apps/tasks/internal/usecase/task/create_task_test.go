package task_test

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "teamflow-tasks/internal/domain/task"
	usecase "teamflow-tasks/internal/usecase/task"
)

// fakeTaskRepo は TaskRepository のテスト用フェイク実装。
type fakeTaskRepo struct {
	saved   *domain.Task
	err     error
	listOut []*domain.Task
}

func (r *fakeTaskRepo) Save(_ context.Context, t *domain.Task) error {
	r.saved = t
	return r.err
}

func (r *fakeTaskRepo) Update(_ context.Context, t *domain.Task) error {
	// mimic save behavior for update
	r.saved = t
	return r.err
}

func (r *fakeTaskRepo) FindByID(_ context.Context, id string) (*domain.Task, error) {
	if r.saved != nil && r.saved.ID == id {
		return r.saved, nil
	}
	for _, t := range r.listOut {
		if t.ID == id {
			return t, nil
		}
	}
	return nil, errors.New("not found")
}

func (r *fakeTaskRepo) ListByProject(_ context.Context, projectID string) ([]*domain.Task, error) {
	return r.listOut, nil
}

func TestNewTask_Success(t *testing.T) {
	now := time.Now()

	task, err := domain.NewTask(
		"task-1",
		"proj-1",
		"画面設計",
		"プロジェクト一覧画面のUIを設計する",
		domain.StatusTodo,
		domain.PriorityMedium,
		nil, // dueDate
		now,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if task.ID != "task-1" {
		t.Errorf("expected ID=task-1, got=%s", task.ID)
	}
	if task.ProjectID != "proj-1" {
		t.Errorf("expected ProjectID=proj-1, got=%s", task.ProjectID)
	}
	if task.Title != "画面設計" {
		t.Errorf("expected Title=画面設計, got=%s", task.Title)
	}
	if !task.CreatedAt.Equal(now) || !task.UpdatedAt.Equal(now) {
		t.Errorf("timestamps not set correctly")
	}
}

func TestNewTask_InvalidTitle(t *testing.T) {
	now := time.Now()

	_, err := domain.NewTask(
		"task-1",
		"proj-1",
		"",
		"説明",
		domain.StatusTodo,
		domain.PriorityMedium,
		nil,
		now,
	)
	if err == nil {
		t.Fatalf("expected error for empty title, got nil")
	}
}

func TestCreateTask_Success(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	repo := &fakeTaskRepo{}
	uc := &usecase.CreateTaskUsecase{
		Repo: repo,
	}

	in := usecase.CreateTaskInput{
		ID:          "task-1",
		ProjectID:   "proj-1",
		Title:       "画面設計",
		Description: "説明",
		Status:      domain.StatusTodo,
		Priority:    domain.PriorityMedium,
		Now:         now,
	}

	task, err := uc.Execute(ctx, in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if task == nil {
		t.Fatalf("expected task, got nil")
	}

	if repo.saved == nil {
		t.Fatalf("expected repo.saved to be non-nil")
	}
}

func TestCreateTask_RepositoryError(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	repoErr := errors.New("db error")
	repo := &fakeTaskRepo{
		err: repoErr,
	}

	uc := &usecase.CreateTaskUsecase{
		Repo: repo,
	}

	in := usecase.CreateTaskInput{
		ID:          "task-1",
		ProjectID:   "proj-1",
		Title:       "画面設計",
		Description: "説明",
		Status:      domain.StatusTodo,
		Priority:    domain.PriorityMedium,
		Now:         now,
	}

	task, err := uc.Execute(ctx, in)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !errors.Is(err, repoErr) {
		t.Fatalf("expected error %v, got %v", repoErr, err)
	}

	if task == nil {
		t.Fatalf("expected task to be non-nil when repo error")
	}
}
