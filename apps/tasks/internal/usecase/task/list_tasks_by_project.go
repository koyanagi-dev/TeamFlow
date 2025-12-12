package task

import (
	"context"

	domain "teamflow-tasks/internal/domain/task"
)

// ListTasksByProjectUsecase は projectID ごとのタスク一覧取得ユースケース。
type ListTasksByProjectUsecase struct {
	Repo TaskRepository
}

func (uc *ListTasksByProjectUsecase) Execute(ctx context.Context, projectID string) ([]*domain.Task, error) {
	return uc.Repo.ListByProject(ctx, projectID)
}
