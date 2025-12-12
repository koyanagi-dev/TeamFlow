package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	domain "teamflow-projects/internal/domain/project"
	infra "teamflow-projects/internal/infrastructure/project"
	usecase "teamflow-projects/internal/usecase/project"
	httpiface "teamflow-projects/internal/interface/http"
)

// テスト用の時刻固定関数
func fixedNow() time.Time {
	return time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
}

func TestCreateProjectHandler_Success(t *testing.T) {
	repo := infra.NewMemoryProjectRepository()

	createUC := &usecase.CreateProjectUsecase{
		Repo: repo,
	}
	listUC := &usecase.ListProjectsUsecase{
		Repo: repo,
	}

	handler := httpiface.NewProjectHandler(createUC, listUC, fixedNow)

	body := map[string]string{
		"id":          "proj-1",
		"name":        "TeamFlow 開発",
		"description": "TeamFlow の開発プロジェクト",
	}

	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/projects", bytes.NewReader(b))
	req = req.WithContext(context.Background())
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", res.StatusCode)
	}

	var respBody struct {
		ID          string    `json:"id"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		CreatedAt   time.Time `json:"createdAt"`
		UpdatedAt   time.Time `json:"updatedAt"`
	}
	if err := json.NewDecoder(res.Body).Decode(&respBody); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if respBody.ID != body["id"] {
		t.Errorf("expected id=%s, got=%s", body["id"], respBody.ID)
	}
	if respBody.Name != body["name"] {
		t.Errorf("expected name=%s, got=%s", body["name"], respBody.Name)
	}
	if respBody.Description != body["description"] {
		t.Errorf("expected description=%s, got=%s", body["description"], respBody.Description)
	}

	// メモリリポジトリに保存されていることも確認
	stored, err := repo.FindByID(context.Background(), "proj-1")
	if err != nil {
		t.Fatalf("expected project to be stored, got error: %v", err)
	}
	if stored == nil {
		t.Fatalf("expected stored project to be non-nil")
	}
}

func TestCreateProjectHandler_InvalidJSON(t *testing.T) {
	repo := infra.NewMemoryProjectRepository()

	createUC := &usecase.CreateProjectUsecase{Repo: repo}
	listUC := &usecase.ListProjectsUsecase{Repo: repo}

	handler := httpiface.NewProjectHandler(createUC, listUC, fixedNow)

	req := httptest.NewRequest(http.MethodPost, "/projects", bytes.NewReader([]byte("{invalid")))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", res.StatusCode)
	}
}

func TestCreateProjectHandler_ValidationError(t *testing.T) {
	repo := infra.NewMemoryProjectRepository()

	createUC := &usecase.CreateProjectUsecase{Repo: repo}
	listUC := &usecase.ListProjectsUsecase{Repo: repo}

	handler := httpiface.NewProjectHandler(createUC, listUC, fixedNow)

	body := map[string]string{
		"id":          "proj-1",
		"name":        "",
		"description": "説明",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/projects", bytes.NewReader(b))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", res.StatusCode)
	}
}

func TestCreateProjectHandler_InternalError(t *testing.T) {
	// リポジトリを差し替えて、あえてエラーを起こす
	repo := &errorRepo{}

	createUC := &usecase.CreateProjectUsecase{Repo: repo}
	listUC := &usecase.ListProjectsUsecase{Repo: repo}

	handler := httpiface.NewProjectHandler(createUC, listUC, fixedNow)

	body := map[string]string{
		"id":          "proj-1",
		"name":        "TeamFlow 開発",
		"description": "TeamFlow の開発プロジェクト",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/projects", bytes.NewReader(b))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", res.StatusCode)
	}
}

// エラーを返すリポジトリ実装（内部エラーのテスト用）
type errorRepo struct{}

func (r *errorRepo) Save(_ context.Context, _ *domain.Project) error {
	return context.DeadlineExceeded
}

func (r *errorRepo) FindByID(_ context.Context, _ string) (*domain.Project, error) {
	return nil, context.DeadlineExceeded
}

func (r *errorRepo) List(_ context.Context) ([]*domain.Project, error) {
	return nil, context.DeadlineExceeded
}
