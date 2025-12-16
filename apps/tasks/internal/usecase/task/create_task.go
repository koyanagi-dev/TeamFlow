package task

import (
	"context"
	"time"

	domain "teamflow-tasks/internal/domain/task"
)

// TaskRepository はタスクの永続化・取得を担当する抽象。
type TaskRepository interface {
	Save(ctx context.Context, t *domain.Task) error
	Update(ctx context.Context, t *domain.Task) error
	FindByID(ctx context.Context, id string) (*domain.Task, error)
	ListByProject(ctx context.Context, projectID string) ([]*domain.Task, error)
}

// CreateTaskInput はタスク作成ユースケースの入力。
type CreateTaskInput struct {
	ID          string
	ProjectID   string
	Title       string
	Description string
	Status      domain.TaskStatus
	Priority    domain.TaskPriority
	Now         time.Time
}

// CreateTaskUsecase はタスク作成ユースケースを表す。
type CreateTaskUsecase struct {
	Repo TaskRepository
}

// Execute は新しいタスクを作成し、リポジトリに保存する。
func (uc *CreateTaskUsecase) Execute(ctx context.Context, in CreateTaskInput) (*domain.Task, error) {
	// いまは dueDate 未対応なので nil 固定
	var dueDate *time.Time = nil

	t, err := domain.NewTask(
		in.ID,
		in.ProjectID,
		in.Title,
		in.Description,
		in.Status,
		in.Priority,
		dueDate,
		in.Now,
	)
	if err != nil {
		return nil, err
	}

	if err := uc.Repo.Save(ctx, t); err != nil {
		return t, err
	}

	return t, nil
}
