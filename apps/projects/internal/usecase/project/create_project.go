package project

import (
	"context"
	"time"

	domain "teamflow-projects/internal/domain/project"
)

// ProjectRepository はプロジェクトの永続化・取得を担当する抽象。
type ProjectRepository interface {
	Save(ctx context.Context, p *domain.Project) error
	FindByID(ctx context.Context, id string) (*domain.Project, error)
	List(ctx context.Context) ([]*domain.Project, error)
}

// CreateProjectInput はプロジェクト作成ユースケースの入力。
type CreateProjectInput struct {
	ID          string
	Name        string
	Description string
	Now         time.Time
}

// CreateProjectUsecase はプロジェクト作成ユースケースを表す。
type CreateProjectUsecase struct {
	Repo ProjectRepository
}

// Execute は新しいプロジェクトを作成し、リポジトリに保存する。
func (uc *CreateProjectUsecase) Execute(ctx context.Context, in CreateProjectInput) (*domain.Project, error) {
	p, err := domain.NewProject(in.ID, in.Name, in.Description, in.Now)
	if err != nil {
		return nil, err
	}

	if err := uc.Repo.Save(ctx, p); err != nil {
		return p, err
	}

	return p, nil
}
