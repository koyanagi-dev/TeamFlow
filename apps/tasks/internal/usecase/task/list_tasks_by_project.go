package task

import (
	"context"
	"sort"

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
	tasks, err := uc.Repo.ListByProject(ctx, in.ProjectID)
	if err != nil {
		return nil, err
	}

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
	})
	return tasks, nil
}
