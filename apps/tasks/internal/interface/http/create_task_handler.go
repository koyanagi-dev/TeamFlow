package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	infra "teamflow-tasks/internal/infrastructure/task"
	usecase "teamflow-tasks/internal/usecase/task"
)

// TaskHandler は /tasks を処理する HTTP ハンドラ。
// - POST: タスク作成
// - GET : projectId ごとのタスク一覧取得
type TaskHandler struct {
	createUC *usecase.CreateTaskUsecase
	listUC   *usecase.ListTasksByProjectUsecase
	updateUC *usecase.UpdateTaskUsecase
	nowFunc  func() time.Time
}

func NewTaskHandler(
	createUC *usecase.CreateTaskUsecase,
	listUC *usecase.ListTasksByProjectUsecase,
	updateUC *usecase.UpdateTaskUsecase,
	nowFunc func() time.Time,
) http.Handler {
	return &TaskHandler{
		createUC: createUC,
		listUC:   listUC,
		updateUC: updateUC,
		nowFunc:  nowFunc,
	}
}

type createTaskRequest struct {
	ID          string `json:"id"`
	ProjectID   string `json:"projectId"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
}

type taskResponse struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"projectId"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority"`
	DueDate     *time.Time `json:"dueDate"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

func (h *TaskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/tasks/") {
		if r.Method != http.MethodPatch {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/tasks/")
		if id == "" || strings.Contains(id, "/") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		h.handleUpdate(w, r, id)
		return
	}

	if r.URL.Path != "/tasks" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodPost:
		h.handleCreate(w, r)
	case http.MethodGet:
		h.handleListByProject(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *TaskHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req createTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	in := usecase.CreateTaskInput{
		ID:          req.ID,
		ProjectID:   req.ProjectID,
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		Priority:    req.Priority,
		Now:         h.nowFunc(),
	}

	t, err := h.createUC.Execute(r.Context(), in)
	if err != nil {
		// バリデーションエラーなどは 400 として扱う（簡易実装）
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	resp := taskResponse{
		ID:          t.ID,
		ProjectID:   t.ProjectID,
		Title:       t.Title,
		Description: t.Description,
		Status:      string(t.Status),   // ★ TaskStatus → string
		Priority:    string(t.Priority), // ★ TaskPriority → string
		DueDate:     t.DueDate,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *TaskHandler) handleListByProject(w http.ResponseWriter, r *http.Request) {
	if h.listUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	projectID := r.URL.Query().Get("projectId")
	if projectID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tasks, err := h.listUC.Execute(r.Context(), projectID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	responses := make([]taskResponse, 0, len(tasks))
	for _, t := range tasks {
		responses = append(responses, taskResponse{
			ID:          t.ID,
			ProjectID:   t.ProjectID,
			Title:       t.Title,
			Description: t.Description,
			Status:      string(t.Status),   // ★ ここも string に変換
			Priority:    string(t.Priority), // ★
			DueDate:     t.DueDate,
			CreatedAt:   t.CreatedAt,
			UpdatedAt:   t.UpdatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(responses)
}

type updateTaskRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	DueDate     *string `json:"dueDate"`
}

func (h *TaskHandler) handleUpdate(w http.ResponseWriter, r *http.Request, id string) {
	if h.updateUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var req updateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	in := usecase.UpdateTaskInput{
		ID:          id,
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		DueDate:     req.DueDate,
		Now:         h.nowFunc(),
	}

	t, err := h.updateUC.Execute(r.Context(), in)
	if err != nil {
		if errors.Is(err, infra.ErrTaskNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// ドメインバリデーションや入力エラーは 400
		if strings.Contains(err.Error(), "must not be empty") ||
			strings.Contains(err.Error(), "invalid task status") ||
			strings.Contains(err.Error(), "dueDate") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp := taskResponse{
		ID:          t.ID,
		ProjectID:   t.ProjectID,
		Title:       t.Title,
		Description: t.Description,
		Status:      string(t.Status),
		Priority:    string(t.Priority),
		DueDate:     t.DueDate,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
