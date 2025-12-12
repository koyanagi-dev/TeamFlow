package project

import (
	"context"

	domain "teamflow-projects/internal/domain/project"
)

// ListProjectsUsecase はプロジェクト一覧取得ユースケース。
type ListProjectsUsecase struct {
	Repo ProjectRepository
}

// Execute はすべてのプロジェクトを取得する。
func (uc *ListProjectsUsecase) Execute(ctx context.Context) ([]*domain.Project, error) {
	return uc.Repo.List(ctx)
}
