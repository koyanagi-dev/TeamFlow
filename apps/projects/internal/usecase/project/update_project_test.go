package project_test

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "teamflow-projects/internal/domain/project"
	usecase "teamflow-projects/internal/usecase/project"
)

type fakeUpdateRepo struct {
	stored  *domain.Project
	findErr error
	saveErr error
}

func (r *fakeUpdateRepo) Save(_ context.Context, p *domain.Project) error {
	r.stored = p
	return r.saveErr
}

func (r *fakeUpdateRepo) FindByID(_ context.Context, id string) (*domain.Project, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	if r.stored == nil {
		return nil, errors.New("not found")
	}
	if r.stored.ID != id {
		return nil, errors.New("not found")
	}
	return r.stored, nil
}

// List は Update のテストでは使わないのでダミーで OK
func (r *fakeUpdateRepo) List(_ context.Context) ([]*domain.Project, error) {
	if r.stored == nil {
		return []*domain.Project{}, nil
	}
	return []*domain.Project{r.stored}, nil
}

func TestUpdateProject_Success(t *testing.T) {
	ctx := context.Background()

	createdAt := time.Now().Add(-time.Hour)
	now := time.Now()

	// 既存プロジェクト
	existing, err := domain.NewProject("proj-1", "Old Name", "Old Desc", createdAt)
	if err != nil {
		t.Fatalf("unexpected error creating existing project: %v", err)
	}

	repo := &fakeUpdateRepo{
		stored: existing,
	}

	uc := &usecase.UpdateProjectUsecase{
		Repo: repo,
	}

	in := usecase.UpdateProjectInput{
		ID:          "proj-1",
		Name:        "New Name",
		Description: "New Desc",
		Now:         now,
	}

	p, err := uc.Execute(ctx, in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.Name != "New Name" {
		t.Errorf("expected Name=New Name, got=%s", p.Name)
	}

	if p.Description != "New Desc" {
		t.Errorf("expected Description=New Desc, got=%s", p.Description)
	}

	if !p.CreatedAt.Equal(createdAt) {
		t.Errorf("expected CreatedAt to remain unchanged, got=%v", p.CreatedAt)
	}

	if !p.UpdatedAt.Equal(now) {
		t.Errorf("expected UpdatedAt to be updated to now, got=%v", p.UpdatedAt)
	}

	if repo.stored != p {
		t.Fatalf("expected repo.stored to be updated project")
	}
}

func TestUpdateProject_EmptyName(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	existing, err := domain.NewProject("proj-1", "Old Name", "Old Desc", now.Add(-time.Hour))
	if err != nil {
		t.Fatalf("unexpected error creating existing project: %v", err)
	}

	repo := &fakeUpdateRepo{
		stored: existing,
	}

	uc := &usecase.UpdateProjectUsecase{
		Repo: repo,
	}

	in := usecase.UpdateProjectInput{
		ID:          "proj-1",
		Name:        "",
		Description: "New Desc",
		Now:         now,
	}

	p, err := uc.Execute(ctx, in)
	if err == nil {
		t.Fatalf("expected error for empty name, got nil")
	}

	if p != nil {
		t.Fatalf("expected project to be nil when validation fails")
	}
}

func TestUpdateProject_FindError(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	findErr := errors.New("db error")
	repo := &fakeUpdateRepo{
		findErr: findErr,
	}

	uc := &usecase.UpdateProjectUsecase{
		Repo: repo,
	}

	in := usecase.UpdateProjectInput{
		ID:          "proj-1",
		Name:        "New Name",
		Description: "New Desc",
		Now:         now,
	}

	p, err := uc.Execute(ctx, in)
	if err == nil {
		t.Fatalf("expected error from FindByID, got nil")
	}

	if !errors.Is(err, findErr) {
		t.Fatalf("expected error %v, got %v", findErr, err)
	}

	if p != nil {
		t.Fatalf("expected project to be nil when find fails")
	}
}

func TestUpdateProject_SaveError(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	existing, err := domain.NewProject("proj-1", "Old Name", "Old Desc", now.Add(-time.Hour))
	if err != nil {
		t.Fatalf("unexpected error creating existing project: %v", err)
	}

	saveErr := errors.New("db error")
	repo := &fakeUpdateRepo{
		stored:  existing,
		saveErr: saveErr,
	}

	uc := &usecase.UpdateProjectUsecase{
		Repo: repo,
	}

	in := usecase.UpdateProjectInput{
		ID:          "proj-1",
		Name:        "New Name",
		Description: "New Desc",
		Now:         now,
	}

	p, err := uc.Execute(ctx, in)
	if err == nil {
		t.Fatalf("expected error from Save, got nil")
	}

	if !errors.Is(err, saveErr) {
		t.Fatalf("expected error %v, got %v", saveErr, err)
	}

	if p == nil {
		t.Fatalf("expected project to be returned even when Save fails")
	}
}
