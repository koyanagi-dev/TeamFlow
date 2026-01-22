package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	domain "teamflow-tasks/internal/domain/task"
	taskinfra "teamflow-tasks/internal/infrastructure/task"
	httpiface "teamflow-tasks/internal/interface/http"
	usecase "teamflow-tasks/internal/usecase/task"
)

func TestCreateTaskHandler_Success(t *testing.T) {
	repo := taskinfra.NewMemoryTaskRepository()

	createUC := &usecase.CreateTaskUsecase{Repo: repo}

	handler := httpiface.NewCreateTaskHandler(createUC, fixedNow)

	body := map[string]string{
		"id":          "task-1",
		"projectId":   "proj-1",
		"title":       "画面設計",
		"description": "プロジェクト一覧画面のUIを設計する",
		"status":      string(domain.StatusTodo),
		"priority":    string(domain.PriorityMedium),
	}

	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(b))
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
		ProjectID   string    `json:"projectId"`
		Title       string    `json:"title"`
		Description string    `json:"description"`
		Status      string    `json:"status"`
		Priority    string    `json:"priority"`
		CreatedAt   time.Time `json:"createdAt"`
		UpdatedAt   time.Time `json:"updatedAt"`
	}
	if err := json.NewDecoder(res.Body).Decode(&respBody); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if respBody.ID != body["id"] {
		t.Errorf("expected id=%s, got=%s", body["id"], respBody.ID)
	}
	if respBody.ProjectID != body["projectId"] {
		t.Errorf("expected projectId=%s, got=%s", body["projectId"], respBody.ProjectID)
	}
	if respBody.Title != body["title"] {
		t.Errorf("expected title=%s, got=%s", body["title"], respBody.Title)
	}
	if respBody.Status != body["status"] {
		t.Errorf("expected status=%s, got=%s", body["status"], respBody.Status)
	}
	if respBody.Priority != body["priority"] {
		t.Errorf("expected priority=%s, got=%s", body["priority"], respBody.Priority)
	}
}

func TestCreateTaskHandler_StatusDoingNormalized(t *testing.T) {
	repo := taskinfra.NewMemoryTaskRepository()

	createUC := &usecase.CreateTaskUsecase{Repo: repo}

	handler := httpiface.NewCreateTaskHandler(createUC, fixedNow)

	body := map[string]string{
		"id":          "task-1",
		"projectId":   "proj-1",
		"title":       "画面設計",
		"description": "プロジェクト一覧画面のUIを設計する",
		"status":      "doing",
		"priority":    string(domain.PriorityMedium),
	}

	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(b))
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
		ProjectID   string    `json:"projectId"`
		Title       string    `json:"title"`
		Description string    `json:"description"`
		Status      string    `json:"status"`
		Priority    string    `json:"priority"`
		CreatedAt   time.Time `json:"createdAt"`
		UpdatedAt   time.Time `json:"updatedAt"`
	}
	if err := json.NewDecoder(res.Body).Decode(&respBody); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// status が "in_progress" に正規化されていることを確認
	if respBody.Status != string(domain.StatusInProgress) {
		t.Errorf("expected status='in_progress', got=%s", respBody.Status)
	}
}

func TestCreateTaskHandler_InvalidJSON(t *testing.T) {
	repo := taskinfra.NewMemoryTaskRepository()
	createUC := &usecase.CreateTaskUsecase{Repo: repo}

	handler := httpiface.NewCreateTaskHandler(createUC, fixedNow)

	req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader([]byte("{invalid")))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", res.StatusCode)
	}
}

func TestCreateTaskHandler_ValidationError(t *testing.T) {
	repo := taskinfra.NewMemoryTaskRepository()
	createUC := &usecase.CreateTaskUsecase{Repo: repo}

	handler := httpiface.NewCreateTaskHandler(createUC, fixedNow)

	// title を空にしてバリデーションエラーを引き起こす
	body := map[string]string{
		"id":          "task-1",
		"projectId":   "proj-1",
		"title":       "",
		"description": "説明",
		"status":      string(domain.StatusTodo),
		"priority":    string(domain.PriorityMedium),
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(b))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", res.StatusCode)
	}
}
