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
	httpiface "teamflow-projects/internal/interface/http"
	usecase "teamflow-projects/internal/usecase/project"
)

func seedProject(repo *infra.MemoryProjectRepository, id string) *domain.Project {
	now := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject(id, "Old Name", "Old Desc", now)
	// インメモリリポジトリに直接格納
	_ = repo.Save(context.Background(), p)
	return p
}

func TestUpdateProjectHandler_Success(t *testing.T) {
	repo := infra.NewMemoryProjectRepository()
	seed := seedProject(repo, "proj-1")

	uc := &usecase.UpdateProjectUsecase{
		Repo: repo,
	}

	handler := httpiface.NewUpdateProjectHandler(uc, fixedNow)

	body := map[string]string{
		"name":        "New Name",
		"description": "New Desc",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/projects/"+seed.ID, bytes.NewReader(b))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.StatusCode)
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

	if respBody.ID != seed.ID {
		t.Errorf("expected id=%s, got=%s", seed.ID, respBody.ID)
	}
	if respBody.Name != "New Name" {
		t.Errorf("expected name=New Name, got=%s", respBody.Name)
	}
	if respBody.Description != "New Desc" {
		t.Errorf("expected description=New Desc, got=%s", respBody.Description)
	}
}

func TestUpdateProjectHandler_InvalidJSON(t *testing.T) {
	repo := infra.NewMemoryProjectRepository()
	seedProject(repo, "proj-1")

	uc := &usecase.UpdateProjectUsecase{Repo: repo}
	handler := httpiface.NewUpdateProjectHandler(uc, fixedNow)

	req := httptest.NewRequest(http.MethodPut, "/projects/proj-1", bytes.NewReader([]byte("{invalid")))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", res.StatusCode)
	}
}

func TestUpdateProjectHandler_EmptyName(t *testing.T) {
	repo := infra.NewMemoryProjectRepository()
	seedProject(repo, "proj-1")

	uc := &usecase.UpdateProjectUsecase{Repo: repo}
	handler := httpiface.NewUpdateProjectHandler(uc, fixedNow)

	body := map[string]string{
		"name":        "",
		"description": "New Desc",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/projects/proj-1", bytes.NewReader(b))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", res.StatusCode)
	}
}

func TestUpdateProjectHandler_NotFound(t *testing.T) {
	repo := infra.NewMemoryProjectRepository() // 何も入れていない

	uc := &usecase.UpdateProjectUsecase{Repo: repo}
	handler := httpiface.NewUpdateProjectHandler(uc, fixedNow)

	body := map[string]string{
		"name":        "New Name",
		"description": "New Desc",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/projects/unknown", bytes.NewReader(b))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", res.StatusCode)
	}
}

func TestUpdateProjectHandler_InternalError(t *testing.T) {
	repo := &errorRepo{} // さっき作った内部エラー用

	uc := &usecase.UpdateProjectUsecase{Repo: repo}
	handler := httpiface.NewUpdateProjectHandler(uc, fixedNow)

	body := map[string]string{
		"name":        "New Name",
		"description": "New Desc",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/projects/proj-1", bytes.NewReader(b))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", res.StatusCode)
	}
}
