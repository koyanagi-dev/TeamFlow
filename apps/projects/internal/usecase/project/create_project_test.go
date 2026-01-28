package project_test

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "teamflow-projects/internal/domain/project"
	usecase "teamflow-projects/internal/usecase/project"
)

// fakeProjectRepo は ProjectRepository のテスト用フェイク実装。
type fakeProjectRepo struct {
	saved   *domain.Project
	err     error
	listOut []*domain.Project
}

func (r *fakeProjectRepo) Save(_ context.Context, p *domain.Project) error {
	r.saved = p
	return r.err
}

func (r *fakeProjectRepo) FindByID(_ context.Context, id string) (*domain.Project, error) {
	// Create のテストでは未使用なのでダミー
	return nil, errors.New("not implemented")
}

func (r *fakeProjectRepo) List(_ context.Context) ([]*domain.Project, error) {
	return r.listOut, nil
}

func TestNewProject_Success(t *testing.T) {
	now := time.Now()

	p, err := domain.NewProject("proj-1", "TeamFlow 開発", "TeamFlow の開発プロジェクト", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.ID != "proj-1" {
		t.Errorf("expected ID=proj-1, got=%s", p.ID)
	}

	if p.Name != "TeamFlow 開発" {
		t.Errorf("expected Name=TeamFlow 開発, got=%s", p.Name)
	}

	if p.Description != "TeamFlow の開発プロジェクト" {
		t.Errorf("expected Description to match, got=%s", p.Description)
	}

	if !p.CreatedAt.Equal(now) {
		t.Errorf("expected CreatedAt to equal now, got=%v", p.CreatedAt)
	}

	if !p.UpdatedAt.Equal(now) {
		t.Errorf("expected UpdatedAt to equal now, got=%v", p.UpdatedAt)
	}
}

func TestNewProject_InvalidName(t *testing.T) {
	now := time.Now()

	_, err := domain.NewProject("proj-1", "", "説明", now)
	if err == nil {
		t.Fatalf("expected error for empty name, got nil")
	}
}

func TestCreateProject_Success(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	repo := &fakeProjectRepo{}
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

	if p.ID != in.ID {
		t.Errorf("expected ID=%s, got=%s", in.ID, p.ID)
	}

	if p.Name != in.Name {
		t.Errorf("expected Name=%s, got=%s", in.Name, p.Name)
	}

	if p.Description != in.Description {
		t.Errorf("expected Description=%s, got=%s", in.Description, p.Description)
	}

	if repo.saved == nil {
		t.Fatalf("expected project to be saved, but repo.saved is nil")
	}
}

func TestCreateProject_EmptyName(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	repo := &fakeProjectRepo{}
	uc := &usecase.CreateProjectUsecase{
		Repo: repo,
	}

	in := usecase.CreateProjectInput{
		ID:          "proj-1",
		Name:        "",
		Description: "説明",
		Now:         now,
	}

	p, err := uc.Execute(ctx, in)
	if err == nil {
		t.Fatalf("expected error for empty name, got nil")
	}

	// p は nil のままで OK なので特にチェック不要
	_ = p

	if repo.saved != nil {
		t.Fatalf("expected repo.saved to be nil when validation fails")
	}
}

func TestCreateProject_RepositoryError(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	repoErr := errors.New("db error")
	repo := &fakeProjectRepo{
		err: repoErr,
	}
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
	if err == nil {
		t.Fatalf("expected error from repository, got nil")
	}

	if !errors.Is(err, repoErr) {
		t.Fatalf("expected error %v, got %v", repoErr, err)
	}

	if p == nil {
		t.Fatalf("expected project to be created before repo error")
	}

	if repo.saved == nil {
		t.Fatalf("expected repo.saved to be non-nil")
	}
}
