package taskinfra_test

import (
	"context"
	"testing"
	"time"

	domain "teamflow-tasks/internal/domain/task"
	infra "teamflow-tasks/internal/infrastructure/task"
	usecase "teamflow-tasks/internal/usecase/task"
)

func TestMemoryTaskRepository_SaveAndListByProject(t *testing.T) {
	repo := infra.NewMemoryTaskRepository()

	uc := &usecase.CreateTaskUsecase{
		Repo: repo,
	}

	ctx := context.Background()
	now := time.Now()

	// proj-1 向けに 2 件、proj-2 向けに 1 件作成
	inputs := []usecase.CreateTaskInput{
		{
			ID:          "task-1",
			ProjectID:   "proj-1",
			Title:       "画面設計",
			Description: "プロジェクト一覧画面のUIを設計する",
			Status:      string(domain.StatusTodo),
			Priority:    string(domain.PriorityMedium),
			Now:         now,
		},
		{
			ID:          "task-2",
			ProjectID:   "proj-1",
			Title:       "API 設計",
			Description: "Tasks API の設計",
			Status:      string(domain.StatusTodo),
			Priority:    string(domain.PriorityMedium),
			Now:         now.Add(-1 * time.Hour), // より古い
		},
		{
			ID:          "task-3",
			ProjectID:   "proj-2",
			Title:       "別プロジェクトのタスク",
			Description: "",
			Status:      string(domain.StatusTodo),
			Priority:    string(domain.PriorityMedium),
			Now:         now.Add(-30 * time.Minute),
		},
	}

	for _, in := range inputs {
		if _, err := uc.Execute(ctx, in); err != nil {
			t.Fatalf("failed to create task %s: %v", in.ID, err)
		}
	}

	// proj-1 だけが取れることを確認
	got, err := repo.ListByProject(ctx, "proj-1")
	if err != nil {
		t.Fatalf("ListByProject returned error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 tasks for proj-1, got %d", len(got))
	}

	if got[0].CreatedAt.After(got[1].CreatedAt) {
		t.Fatalf("expected ascending order by CreatedAt, got %v then %v", got[0].CreatedAt, got[1].CreatedAt)
	}

	for _, task := range got {
		if task.ProjectID != "proj-1" {
			t.Errorf("expected ProjectID=proj-1, got %s", task.ProjectID)
		}
	}
}
