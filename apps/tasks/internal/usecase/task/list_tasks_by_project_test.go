package task_test

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "teamflow-tasks/internal/domain/task"
	usecase "teamflow-tasks/internal/usecase/task"
)

type listRepo struct {
	out []*domain.Task
}

func (r *listRepo) Save(context.Context, *domain.Task) error   { return nil }
func (r *listRepo) Update(context.Context, *domain.Task) error { return nil }
func (r *listRepo) FindByID(_ context.Context, id string) (*domain.Task, error) {
	for _, t := range r.out {
		if t.ID == id {
			return t, nil
		}
	}
	return nil, errors.New("not found")
}
func (r *listRepo) ListByProject(context.Context, string) ([]*domain.Task, error) {
	return r.out, nil
}

func TestListTasksByProject_Success(t *testing.T) {
	now := time.Now()

	t1, _ := domain.NewTask(
		"task-1",
		"proj-1",
		"T1",
		"",
		domain.TaskStatus("todo"),
		domain.TaskPriority("medium"),
		nil,
		now.Add(30*time.Minute),
	)
	t2, _ := domain.NewTask(
		"task-2",
		"proj-1",
		"T2",
		"",
		domain.TaskStatus("todo"),
		domain.TaskPriority("medium"),
		nil,
		now.Add(-30*time.Minute), // より古い
	)

	repo := &listRepo{
		out: []*domain.Task{t1, t2},
	}

	uc := &usecase.ListTasksByProjectUsecase{
		Repo: repo,
	}

	got, err := uc.Execute(context.Background(), usecase.ListTasksByProjectInput{
		ProjectID: "proj-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(got))
	}

	if got[0].CreatedAt.After(got[1].CreatedAt) {
		t.Fatalf("tasks are not sorted by CreatedAt ascending: %v then %v", got[0].CreatedAt, got[1].CreatedAt)
	}
}
