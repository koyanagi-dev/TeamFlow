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
	// 後方互換性のため残す。Queryが指定されていない場合はこちらを使用
}

type ListTasksByProjectWithQueryInput struct {
	ProjectID string
	Query     *domain.TaskQuery
}

// Execute は既存のAPI向け（後方互換性のため残す）。
func (uc *ListTasksByProjectUsecase) Execute(ctx context.Context, in ListTasksByProjectInput) ([]*domain.Task, error) {
	tasks, err := uc.Repo.ListByProject(ctx, in.ProjectID)
	if err != nil {
		return nil, err
	}

	// 既存のソートロジックを維持（createdAt ASC）
	// 注意: フィルタリングは実装されていない（既存の挙動を維持）
	return tasks, nil
}

// ExecuteWithQuery はQuery Objectを受け取り、フィルタ/ソート/リミットを適用する。
func (uc *ListTasksByProjectUsecase) ExecuteWithQuery(ctx context.Context, in ListTasksByProjectWithQueryInput) ([]*domain.Task, error) {
	if in.Query == nil {
		// Queryがnilの場合は空のQueryを作成（全件取得、デフォルトソート）
		var err error
		in.Query, err = domain.NewTaskQuery()
		if err != nil {
			return nil, err
		}
	}

	tasks, err := uc.Repo.FindByProjectID(ctx, in.ProjectID, in.Query)
	if err != nil {
		return nil, err
	}

	return tasks, nil
}
