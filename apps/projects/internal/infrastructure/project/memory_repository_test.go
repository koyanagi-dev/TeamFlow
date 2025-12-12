package projectinfra

import (
	"context"
	"testing"
	"time"

	usecase "teamflow-projects/internal/usecase/project"
)

func TestMemoryProjectRepository_SaveStoresProject(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	repo := NewMemoryProjectRepository()

	uc := &usecase.CreateProjectUsecase{
		Repo: repo,
	}

	in := usecase.CreateProjectInput{
		ID:          "proj-1",
		Name:        "TeamFlow 開発",
		Description: "TeamFlow の開発プロジェクト",
		Now:         now,
	}

	p, err := uc.Execute(ctx, in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p == nil {
		t.Fatalf("expected project, got nil")
	}

	// 内部状態を直接検査（インメモリ実装なので許容）
	stored, ok := repo.projects[p.ID]
	if !ok {
		t.Fatalf("expected project with ID=%s to be stored", p.ID)
	}

	if stored != p {
		t.Fatalf("expected stored project pointer to equal returned project")
	}
}
