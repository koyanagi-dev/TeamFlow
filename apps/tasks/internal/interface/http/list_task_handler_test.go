package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	domain "teamflow-tasks/internal/domain/task"
	taskinfra "teamflow-tasks/internal/infrastructure/task"
	httpiface "teamflow-tasks/internal/interface/http"
	usecase "teamflow-tasks/internal/usecase/task"
)

func TestListTasksByProjectHandler_Success(t *testing.T) {
	repo := taskinfra.NewMemoryTaskRepository()

	// まずはユースケース経由でタスクを保存
	createUC := &usecase.CreateTaskUsecase{Repo: repo}
	listUC := &usecase.ListTasksByProjectUsecase{Repo: repo}

	ctx := context.Background()
	now := fixedNow()

	inputs := []usecase.CreateTaskInput{
		{
			ID:          "task-1",
			ProjectID:   "proj-1",
			Title:       "画面設計",
			Description: "一覧画面のUI設計",
			Status:      domain.StatusTodo,
			Priority:    domain.PriorityMedium,
			Now:         now,
		},
		{
			ID:          "task-2",
			ProjectID:   "proj-1",
			Title:       "API設計",
			Description: "Tasks API 設計",
			Status:      domain.StatusTodo,
			Priority:    domain.PriorityMedium,
			Now:         now,
		},
		{
			ID:          "task-3",
			ProjectID:   "proj-2",
			Title:       "別プロジェクトのタスク",
			Description: "",
			Status:      domain.StatusTodo,
			Priority:    domain.PriorityMedium,
			Now:         now,
		},
	}

	for _, in := range inputs {
		if _, err := createUC.Execute(ctx, in); err != nil {
			t.Fatalf("failed to create task %s: %v", in.ID, err)
		}
	}

	handler := httpiface.NewListTaskHandler(listUC, fixedNow, []byte("test-secret"))

	req := httptest.NewRequest(http.MethodGet, "/tasks?projectId=proj-1", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		var errorResp struct {
			Error  string `json:"error"`
			Detail string `json:"detail"`
		}
		if err := json.NewDecoder(res.Body).Decode(&errorResp); err == nil {
			t.Fatalf("expected status 200, got %d: error=%s, detail=%s", res.StatusCode, errorResp.Error, errorResp.Detail)
		} else {
			t.Fatalf("expected status 200, got %d", res.StatusCode)
		}
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
