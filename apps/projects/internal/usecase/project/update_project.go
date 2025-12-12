package project

import (
	"context"
	"errors"
	"time"

	domain "teamflow-projects/internal/domain/project"
)

// UpdateProjectInput はプロジェクト更新ユースケースの入力。
type UpdateProjectInput struct {
	ID          string
	Name        string
	Description string
	Now         time.Time
}

// UpdateProjectUsecase はプロジェクト更新ユースケースを表す。
type UpdateProjectUsecase struct {
	Repo ProjectRepository
}

// Execute は既存プロジェクトを取得し、名前・説明・UpdatedAt を更新する。
func (uc *UpdateProjectUsecase) Execute(ctx context.Context, in UpdateProjectInput) (*domain.Project, error) {
	if in.Name == "" {
		return nil, errors.New("project name must not be empty")
	}

	// 既存プロジェクトを取得
	existing, err := uc.Repo.FindByID(ctx, in.ID)
	if err != nil {
		return nil, err
	}

	existing.Name = in.Name
	existing.Description = in.Description
	existing.UpdatedAt = in.Now

	if err := uc.Repo.Save(ctx, existing); err != nil {
		return existing, err
	}

	return existing, nil
}
