package project_test

import (
	"context"
	"testing"
	"time"

	domain "teamflow-projects/internal/domain/project"
	usecase "teamflow-projects/internal/usecase/project"
)

// List 用の簡単なフェイク
type listRepo struct {
	out []*domain.Project
}

func (r *listRepo) Save(context.Context, *domain.Project) error               { return nil }
func (r *listRepo) FindByID(context.Context, string) (*domain.Project, error) { return nil, nil }
func (r *listRepo) List(context.Context) ([]*domain.Project, error)           { return r.out, nil }

func TestListProjects_Success(t *testing.T) {
	now := time.Now()
	p1, _ := domain.NewProject("proj-1", "P1", "", now)
	p2, _ := domain.NewProject("proj-2", "P2", "", now)

	repo := &listRepo{
		out: []*domain.Project{p1, p2},
	}

	uc := &usecase.ListProjectsUsecase{
		Repo: repo,
	}

	got, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(got))
	}
}
