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

// テスト用の固定時刻
func fixedNow() time.Time {
	return time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
}

func TestCreateTaskHandler_Success(t *testing.T) {
	repo := taskinfra.NewMemoryTaskRepository()

	createUC := &usecase.CreateTaskUsecase{Repo: repo}
	listUC := &usecase.ListTasksByProjectUsecase{Repo: repo}
	updateUC := &usecase.UpdateTaskUsecase{Repo: repo}

	handler := httpiface.NewTaskHandler(createUC, listUC, updateUC, fixedNow)

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

func TestCreateTaskHandler_InvalidJSON(t *testing.T) {
	repo := taskinfra.NewMemoryTaskRepository()
	createUC := &usecase.CreateTaskUsecase{Repo: repo}
	listUC := &usecase.ListTasksByProjectUsecase{Repo: repo}
	updateUC := &usecase.UpdateTaskUsecase{Repo: repo}

	handler := httpiface.NewTaskHandler(createUC, listUC, updateUC, fixedNow)

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
	listUC := &usecase.ListTasksByProjectUsecase{Repo: repo}
	updateUC := &usecase.UpdateTaskUsecase{Repo: repo}

	handler := httpiface.NewTaskHandler(createUC, listUC, updateUC, fixedNow)

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

func TestListTasksByProjectHandler_Success(t *testing.T) {
	repo := taskinfra.NewMemoryTaskRepository()

	// まずはユースケース経由でタスクを保存
	createUC := &usecase.CreateTaskUsecase{Repo: repo}
	listUC := &usecase.ListTasksByProjectUsecase{Repo: repo}
	updateUC := &usecase.UpdateTaskUsecase{Repo: repo}

	ctx := context.Background()
	now := fixedNow()

	inputs := []usecase.CreateTaskInput{
		{
			ID:          "task-1",
			ProjectID:   "proj-1",
			Title:       "画面設計",
			Description: "一覧画面のUI設計",
			Status:      string(domain.StatusTodo),
			Priority:    string(domain.PriorityMedium),
			Now:         now,
		},
		{
			ID:          "task-2",
			ProjectID:   "proj-1",
			Title:       "API設計",
			Description: "Tasks API 設計",
			Status:      string(domain.StatusTodo),
			Priority:    string(domain.PriorityMedium),
			Now:         now,
		},
		{
			ID:          "task-3",
			ProjectID:   "proj-2",
			Title:       "別プロジェクトのタスク",
			Description: "",
			Status:      string(domain.StatusTodo),
			Priority:    string(domain.PriorityMedium),
			Now:         now,
		},
	}

	for _, in := range inputs {
		if _, err := createUC.Execute(ctx, in); err != nil {
			t.Fatalf("failed to create task %s: %v", in.ID, err)
		}
	}

	handler := httpiface.NewTaskHandler(createUC, listUC, updateUC, fixedNow)

	req := httptest.NewRequest(http.MethodGet, "/tasks?projectId=proj-1", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.StatusCode)
	}

	var respBody []struct {
		ID        string `json:"id"`
		ProjectID string `json:"projectId"`
		Title     string `json:"title"`
	}
	if err := json.NewDecoder(res.Body).Decode(&respBody); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(respBody) != 2 {
		t.Fatalf("expected 2 tasks for proj-1, got %d", len(respBody))
	}
	for _, tsk := range respBody {
		if tsk.ProjectID != "proj-1" {
			t.Errorf("expected projectId=proj-1, got %s", tsk.ProjectID)
		}
	}
}

func TestUpdateTaskHandler_Success(t *testing.T) {
	repo := taskinfra.NewMemoryTaskRepository()
	createUC := &usecase.CreateTaskUsecase{Repo: repo}
	updateUC := &usecase.UpdateTaskUsecase{Repo: repo}
	listUC := &usecase.ListTasksByProjectUsecase{Repo: repo}

	now := fixedNow()
	ctx := context.Background()

	// 事前にタスク作成
	_, err := createUC.Execute(ctx, usecase.CreateTaskInput{
		ID:          "task-1",
		ProjectID:   "proj-1",
		Title:       "initial",
		Description: "desc",
		Status:      string(domain.StatusTodo),
		Priority:    string(domain.PriorityMedium),
		Now:         now,
	})
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	handler := httpiface.NewTaskHandler(createUC, listUC, updateUC, fixedNow)

	body := map[string]string{
		"title":      "updated title",
		"status":     string(domain.StatusInProgress),
		"dueDate":    "2025-02-01T12:00:00Z",
		"assigneeId": "user-1",
		"priority":   string(domain.PriorityHigh),
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/tasks/task-1", bytes.NewReader(b))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.StatusCode)
	}

	var respBody struct {
		Title   string    `json:"title"`
		Status  string    `json:"status"`
		DueDate time.Time `json:"dueDate"`
	}
	if err := json.NewDecoder(res.Body).Decode(&respBody); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if respBody.Title != "updated title" {
		t.Errorf("expected updated title, got %s", respBody.Title)
	}
	if respBody.Status != string(domain.StatusInProgress) {
		t.Errorf("expected status in_progress, got %s", respBody.Status)
	}
	expectedDue := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)
	if !respBody.DueDate.Equal(expectedDue) {
		t.Errorf("expected dueDate %v, got %v", expectedDue, respBody.DueDate)
	}
}
