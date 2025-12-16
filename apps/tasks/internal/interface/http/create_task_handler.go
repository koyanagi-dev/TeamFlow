package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	domain "teamflow-tasks/internal/domain/task"
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

	status, err := parseStatusInput(req.Status)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	priority, err := parsePriorityInput(req.Priority)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	in := usecase.CreateTaskInput{
		ID:          req.ID,
		ProjectID:   req.ProjectID,
		Title:       req.Title,
		Description: req.Description,
		Status:      status,
		Priority:    priority,
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

	status := r.URL.Query().Get("status")
	assigneeId := r.URL.Query().Get("assigneeId")
	projectID := r.URL.Query().Get("projectId")
	if projectID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tasks, err := h.listUC.Execute(r.Context(), usecase.ListTasksByProjectInput{
		ProjectID:  projectID,
		Status:     status,
		AssigneeID: assigneeId,
	})
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
	AssigneeID  *string `json:"assigneeId"`
	Priority    *string `json:"priority"`
	DueDate     *string `json:"dueDate"`
}

// PatchTaskRequest は PATCH /api/tasks/{id} のリクエストボディ。
type PatchTaskRequest struct {
	Title    *string `json:"title"`
	Status   *string `json:"status"`
	Priority *string `json:"priority"`
}

func (h *TaskHandler) handleUpdate(w http.ResponseWriter, r *http.Request, id string) {
	if h.updateUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var req PatchTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 全部 nil チェック
	if req.Title == nil && req.Status == nil && req.Priority == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var trimmedTitle *string
	if req.Title != nil {
		// title が空文字または空白のみの場合は 400
		trimmed := strings.TrimSpace(*req.Title)
		if trimmed == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		trimmedTitle = &trimmed
	}

	var status *domain.TaskStatus
	if req.Status != nil {
		parsed, err := parseStatusInput(*req.Status)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		status = &parsed
	}

	var priority *domain.TaskPriority
	if req.Priority != nil {
		parsed, err := parsePriorityInput(*req.Priority)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		priority = &parsed
	}

	in := usecase.UpdateTaskInput{
		ID:       id,
		Title:    trimmedTitle,
		Status:   status,
		Priority: priority,
		Now:      h.nowFunc(),
	}

	t, err := h.updateUC.Execute(r.Context(), in)
	if err != nil {
		if errors.Is(err, usecase.ErrTaskNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if errors.Is(err, usecase.ErrInvalidInput) {
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

func parseStatusInput(raw string) (domain.TaskStatus, error) {
	normalized := normalizeStatus(raw)
	return domain.ParseStatus(normalized)
}

func parsePriorityInput(raw string) (domain.TaskPriority, error) {
	return domain.ParsePriority(raw)
}

func normalizeStatus(raw string) string {
	if raw == "doing" {
		return string(domain.StatusInProgress)
	}
	return raw
}
