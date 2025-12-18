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

func TestCreateTaskHandler_StatusDoingNormalized(t *testing.T) {
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

	handler := httpiface.NewTaskHandler(createUC, listUC, updateUC, fixedNow)

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

func TestPatchTaskHandler_Success(t *testing.T) {
	repo := taskinfra.NewMemoryTaskRepository()
	createUC := &usecase.CreateTaskUsecase{Repo: repo}
	updateUC := &usecase.UpdateTaskUsecase{Repo: repo}
	listUC := &usecase.ListTasksByProjectUsecase{Repo: repo}

	now := fixedNow()
	ctx := context.Background()

	// 事前にタスク作成
	createdTask, err := createUC.Execute(ctx, usecase.CreateTaskInput{
		ID:          "task-1",
		ProjectID:   "proj-1",
		Title:       "initial title",
		Description: "desc",
		Status:      domain.StatusTodo,
		Priority:    domain.PriorityMedium,
		Now:         now,
	})
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// 更新前の値を保存（UpdatedAt が更新されることを確認するため）
	originalUpdatedAt := createdTask.UpdatedAt
	originalCreatedAt := createdTask.CreatedAt

	// 更新時は異なる時刻を使用（updatedAt が更新されることを確認するため）
	updateTime := now.Add(1 * time.Hour)
	handler := httpiface.NewTaskHandler(createUC, listUC, updateUC, func() time.Time { return updateTime })

	// title のみを更新
	body := map[string]string{
		"title": "updated title",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/tasks/task-1", bytes.NewReader(b))
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

	var respBody struct {
		ID          string    `json:"id"`
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

	if respBody.Title != "updated title" {
		t.Errorf("expected title 'updated title', got %s", respBody.Title)
	}
	// createdAt は維持される
	if !respBody.CreatedAt.Equal(originalCreatedAt) {
		t.Errorf("expected createdAt to be unchanged, got %v", respBody.CreatedAt)
	}
	// updatedAt は更新される
	if !respBody.UpdatedAt.After(originalUpdatedAt) {
		t.Errorf("expected updatedAt to be after %v, got %v", originalUpdatedAt, respBody.UpdatedAt)
	}
	// 他のフィールドは変更されない
	if respBody.Description != createdTask.Description {
		t.Errorf("expected description to be unchanged, got %s", respBody.Description)
	}
	if respBody.Status != string(createdTask.Status) {
		t.Errorf("expected status to be unchanged, got %s", respBody.Status)
	}
}

func TestPatchTaskHandler_AllFieldsNotProvided(t *testing.T) {
	repo := taskinfra.NewMemoryTaskRepository()
	createUC := &usecase.CreateTaskUsecase{Repo: repo}
	updateUC := &usecase.UpdateTaskUsecase{Repo: repo}
	listUC := &usecase.ListTasksByProjectUsecase{Repo: repo}

	handler := httpiface.NewTaskHandler(createUC, listUC, updateUC, fixedNow)

	// 全フィールド未指定
	body := map[string]interface{}{}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/tasks/task-1", bytes.NewReader(b))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", res.StatusCode)
	}
}

func TestPatchTaskHandler_TitleEmpty(t *testing.T) {
	repo := taskinfra.NewMemoryTaskRepository()
	createUC := &usecase.CreateTaskUsecase{Repo: repo}
	updateUC := &usecase.UpdateTaskUsecase{Repo: repo}
	listUC := &usecase.ListTasksByProjectUsecase{Repo: repo}

	handler := httpiface.NewTaskHandler(createUC, listUC, updateUC, fixedNow)

	// title が空文字
	body := map[string]string{
		"title": "",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/tasks/task-1", bytes.NewReader(b))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", res.StatusCode)
	}
}

func TestPatchTaskHandler_TitleWhitespace(t *testing.T) {
	repo := taskinfra.NewMemoryTaskRepository()
	createUC := &usecase.CreateTaskUsecase{Repo: repo}
	updateUC := &usecase.UpdateTaskUsecase{Repo: repo}
	listUC := &usecase.ListTasksByProjectUsecase{Repo: repo}

	handler := httpiface.NewTaskHandler(createUC, listUC, updateUC, fixedNow)

	// title が空白のみ
	body := map[string]string{
		"title": "   ",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/tasks/task-1", bytes.NewReader(b))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", res.StatusCode)
	}
}

func TestPatchTaskHandler_TaskNotFound(t *testing.T) {
	repo := taskinfra.NewMemoryTaskRepository()
	createUC := &usecase.CreateTaskUsecase{Repo: repo}
	updateUC := &usecase.UpdateTaskUsecase{Repo: repo}
	listUC := &usecase.ListTasksByProjectUsecase{Repo: repo}

	handler := httpiface.NewTaskHandler(createUC, listUC, updateUC, fixedNow)

	body := map[string]string{
		"title": "updated title",
	}
	b, _ := json.Marshal(body)

	// 存在しないタスク ID
	req := httptest.NewRequest(http.MethodPatch, "/tasks/non-existent", bytes.NewReader(b))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", res.StatusCode)
	}
}

func TestPatchTaskHandler_UpdateStatus(t *testing.T) {
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
		Title:       "initial title",
		Description: "desc",
		Status:      domain.StatusTodo,
		Priority:    domain.PriorityMedium,
		Now:         now,
	})
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	updateTime := now.Add(1 * time.Hour)
	handler := httpiface.NewTaskHandler(createUC, listUC, updateUC, func() time.Time { return updateTime })

	// status のみを更新
	body := map[string]string{
		"status": "doing",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/tasks/task-1", bytes.NewReader(b))
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

	var respBody struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(res.Body).Decode(&respBody); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// status が "in_progress" に更新されていることを確認（domain の値）
	if respBody.Status != string(domain.StatusInProgress) {
		t.Errorf("expected status 'in_progress', got %s", respBody.Status)
	}
}

func TestPatchTaskHandler_UpdatePriority(t *testing.T) {
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
		Title:       "initial title",
		Description: "desc",
		Status:      domain.StatusTodo,
		Priority:    domain.PriorityMedium,
		Now:         now,
	})
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	updateTime := now.Add(1 * time.Hour)
	handler := httpiface.NewTaskHandler(createUC, listUC, updateUC, func() time.Time { return updateTime })

	// priority のみを更新
	body := map[string]string{
		"priority": "high",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/tasks/task-1", bytes.NewReader(b))
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

	var respBody struct {
		Priority string `json:"priority"`
	}
	if err := json.NewDecoder(res.Body).Decode(&respBody); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if respBody.Priority != "high" {
		t.Errorf("expected priority 'high', got %s", respBody.Priority)
	}
}

func TestPatchTaskHandler_UpdateTitleAndStatus(t *testing.T) {
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
		Title:       "initial title",
		Description: "desc",
		Status:      domain.StatusTodo,
		Priority:    domain.PriorityMedium,
		Now:         now,
	})
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	updateTime := now.Add(1 * time.Hour)
	handler := httpiface.NewTaskHandler(createUC, listUC, updateUC, func() time.Time { return updateTime })

	// title と status を同時更新
	body := map[string]string{
		"title":  "x",
		"status": "done",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/tasks/task-1", bytes.NewReader(b))
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

	var respBody struct {
		Title  string `json:"title"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(res.Body).Decode(&respBody); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if respBody.Title != "x" {
		t.Errorf("expected title 'x', got %s", respBody.Title)
	}
	if respBody.Status != string(domain.StatusDone) {
		t.Errorf("expected status 'done', got %s", respBody.Status)
	}
}

func TestPatchTaskHandler_UpdateStatusInProgress(t *testing.T) {
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
		Title:       "initial title",
		Description: "desc",
		Status:      domain.StatusTodo,
		Priority:    domain.PriorityMedium,
		Now:         now,
	})
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	updateTime := now.Add(1 * time.Hour)
	handler := httpiface.NewTaskHandler(createUC, listUC, updateUC, func() time.Time { return updateTime })

	// status を "in_progress" で更新
	body := map[string]string{
		"status": "in_progress",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/tasks/task-1", bytes.NewReader(b))
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

	var respBody struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(res.Body).Decode(&respBody); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// status が "in_progress" に更新されていることを確認（domain の値）
	if respBody.Status != string(domain.StatusInProgress) {
		t.Errorf("expected status 'in_progress', got %s", respBody.Status)
	}
}

func TestPatchTaskHandler_InvalidStatus(t *testing.T) {
	repo := taskinfra.NewMemoryTaskRepository()
	createUC := &usecase.CreateTaskUsecase{Repo: repo}
	updateUC := &usecase.UpdateTaskUsecase{Repo: repo}
	listUC := &usecase.ListTasksByProjectUsecase{Repo: repo}

	handler := httpiface.NewTaskHandler(createUC, listUC, updateUC, fixedNow)

	// 無効な status
	body := map[string]string{
		"status": "in-progress",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/tasks/task-1", bytes.NewReader(b))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", res.StatusCode)
	}
}

func TestPatchTaskHandler_InvalidPriority(t *testing.T) {
	repo := taskinfra.NewMemoryTaskRepository()
	createUC := &usecase.CreateTaskUsecase{Repo: repo}
	updateUC := &usecase.UpdateTaskUsecase{Repo: repo}
	listUC := &usecase.ListTasksByProjectUsecase{Repo: repo}

	handler := httpiface.NewTaskHandler(createUC, listUC, updateUC, fixedNow)

	// 無効な priority
	body := map[string]string{
		"priority": "urgent",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/tasks/task-1", bytes.NewReader(b))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", res.StatusCode)
	}
}

func TestPatchTaskHandler_UpdateDescription(t *testing.T) {
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
		Title:       "initial title",
		Description: "initial description",
		Status:      domain.StatusTodo,
		Priority:    domain.PriorityMedium,
		Now:         now,
	})
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	updateTime := now.Add(1 * time.Hour)
	handler := httpiface.NewTaskHandler(createUC, listUC, updateUC, func() time.Time { return updateTime })

	// description のみを更新
	body := map[string]string{
		"description": "updated description",
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
		ID          string    `json:"id"`
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

	if respBody.Description != "updated description" {
		t.Errorf("expected description 'updated description', got %s", respBody.Description)
	}
	// 他のフィールドは変更されない
	if respBody.Title != "initial title" {
		t.Errorf("expected title to be unchanged, got %s", respBody.Title)
	}
	if respBody.Status != string(domain.StatusTodo) {
		t.Errorf("expected status to be unchanged, got %s", respBody.Status)
	}
}

func TestPatchTaskHandler_UpdateDescriptionToNull(t *testing.T) {
	repo := taskinfra.NewMemoryTaskRepository()
	createUC := &usecase.CreateTaskUsecase{Repo: repo}
	updateUC := &usecase.UpdateTaskUsecase{Repo: repo}
	listUC := &usecase.ListTasksByProjectUsecase{Repo: repo}

	now := fixedNow()
	ctx := context.Background()

	// 事前にタスク作成（description あり）
	_, err := createUC.Execute(ctx, usecase.CreateTaskInput{
		ID:          "task-1",
		ProjectID:   "proj-1",
		Title:       "initial title",
		Description: "initial description",
		Status:      domain.StatusTodo,
		Priority:    domain.PriorityMedium,
		Now:         now,
	})
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	updateTime := now.Add(1 * time.Hour)
	handler := httpiface.NewTaskHandler(createUC, listUC, updateUC, func() time.Time { return updateTime })

	// description を null で更新（説明を消す）
	body := map[string]interface{}{
		"description": nil,
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
		ID          string    `json:"id"`
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

	// description が空文字列になっていることを確認（null で消された）
	if respBody.Description != "" {
		t.Errorf("expected description to be empty string, got %s", respBody.Description)
	}
	// 他のフィールドは変更されない
	if respBody.Title != "initial title" {
		t.Errorf("expected title to be unchanged, got %s", respBody.Title)
	}
}


func TestPatchTaskHandler_UpdateAssigneeID(t *testing.T) {
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
		Title:       "initial title",
		Description: "desc",
		Status:      domain.StatusTodo,
		Priority:    domain.PriorityMedium,
		Now:         now,
	})
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	updateTime := now.Add(1 * time.Hour)
	handler := httpiface.NewTaskHandler(createUC, listUC, updateUC, func() time.Time { return updateTime })

	// assigneeId のみを更新
	validUUID := "12345678-1234-1234-1234-123456789abc"
	body := map[string]interface{}{
		"assigneeId": validUUID,
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/tasks/task-1", bytes.NewReader(b))
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

	var respBody struct {
		AssigneeID *string `json:"assigneeId"`
	}
	if err := json.NewDecoder(res.Body).Decode(&respBody); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if respBody.AssigneeID == nil {
		t.Errorf("expected assigneeId to be set, got nil")
	} else if *respBody.AssigneeID != validUUID {
		t.Errorf("expected assigneeId '%s', got '%s'", validUUID, *respBody.AssigneeID)
	}
}


func TestPatchTaskHandler_UpdateAssigneeIDNull(t *testing.T) {
	repo := taskinfra.NewMemoryTaskRepository()
	createUC := &usecase.CreateTaskUsecase{Repo: repo}
	updateUC := &usecase.UpdateTaskUsecase{Repo: repo}
	listUC := &usecase.ListTasksByProjectUsecase{Repo: repo}

	now := fixedNow()
	ctx := context.Background()

	// 事前にタスク作成（assigneeId を設定済み）
	initialAssigneeID := "12345678-1234-1234-1234-123456789abc"
	_, err := createUC.Execute(ctx, usecase.CreateTaskInput{
		ID:          "task-1",
		ProjectID:   "proj-1",
		Title:       "initial title",
		Description: "desc",
		Status:      domain.StatusTodo,
		Priority:    domain.PriorityMedium,
		Now:         now,
	})
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// まず assigneeId を設定
	updateTime1 := now.Add(1 * time.Hour)
	handler1 := httpiface.NewTaskHandler(createUC, listUC, updateUC, func() time.Time { return updateTime1 })
	body1 := map[string]interface{}{
		"assigneeId": initialAssigneeID,
	}
	b1, _ := json.Marshal(body1)
	req1 := httptest.NewRequest(http.MethodPatch, "/tasks/task-1", bytes.NewReader(b1))
	w1 := httptest.NewRecorder()
	handler1.ServeHTTP(w1, req1)
	if w1.Result().StatusCode != http.StatusOK {
		t.Fatalf("failed to set initial assigneeId")
	}

	// 次に assigneeId を null で外す
	updateTime2 := now.Add(2 * time.Hour)
	handler2 := httpiface.NewTaskHandler(createUC, listUC, updateUC, func() time.Time { return updateTime2 })
	body2 := map[string]interface{}{
		"assigneeId": nil,
	}
	b2, _ := json.Marshal(body2)

	req2 := httptest.NewRequest(http.MethodPatch, "/tasks/task-1", bytes.NewReader(b2))
	w2 := httptest.NewRecorder()

	handler2.ServeHTTP(w2, req2)

	res := w2.Result()
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

	var respBody struct {
		AssigneeID *string `json:"assigneeId"`
	}
	if err := json.NewDecoder(res.Body).Decode(&respBody); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if respBody.AssigneeID != nil {
		t.Errorf("expected assigneeId to be nil, got '%s'", *respBody.AssigneeID)
	}
}


func TestPatchTaskHandler_InvalidAssigneeID(t *testing.T) {
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
		Title:       "initial title",
		Description: "desc",
		Status:      domain.StatusTodo,
		Priority:    domain.PriorityMedium,
		Now:         now,
	})
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	handler := httpiface.NewTaskHandler(createUC, listUC, updateUC, fixedNow)

	// 無効な UUID 形式
	body := map[string]string{
		"assigneeId": "not-uuid",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/tasks/task-1", bytes.NewReader(b))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", res.StatusCode)
	}

	var errorResp struct {
		Error  string `json:"error"`
		Detail string `json:"detail"`
	}
	if err := json.NewDecoder(res.Body).Decode(&errorResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errorResp.Error != "validation error" {
		t.Errorf("expected error 'validation error', got '%s'", errorResp.Error)
	}
}
