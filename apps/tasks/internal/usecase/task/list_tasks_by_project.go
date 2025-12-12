package task

import (
	"context"

	domain "teamflow-tasks/internal/domain/task"
)

// ListTasksByProjectUsecase は projectID ごとのタスク一覧取得ユースケース。
type ListTasksByProjectUsecase struct {
	Repo TaskRepository
}

type ListTasksByProjectInput struct {
	ProjectID  string
	Status     string
	AssigneeID string
}

func (uc *ListTasksByProjectUsecase) Execute(ctx context.Context, in ListTasksByProjectInput) ([]*domain.Task, error) {
	return uc.Repo.ListByProject(ctx, in.ProjectID)
}
