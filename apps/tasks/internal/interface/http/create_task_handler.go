package http

import (
	"encoding/json"
	"net/http"
	"time"

	domain "teamflow-tasks/internal/domain/task"
	usecase "teamflow-tasks/internal/usecase/task"
)

// CreateTaskHandler は POST /tasks を処理する HTTP ハンドラ。
//
// 責務:
//   - POST /api/tasks エンドポイントのリクエストを受け付ける
//   - リクエストボディのJSONをパースし、バリデーションを行う
//   - CreateTaskUsecaseを呼び出してタスクを作成する
//   - 作成されたタスクをJSONレスポンスとして返す
type CreateTaskHandler struct {
	createUC *usecase.CreateTaskUsecase
	nowFunc  func() time.Time
}

// NewCreateTaskHandler は CreateTaskHandler を生成する。
func NewCreateTaskHandler(
	createUC *usecase.CreateTaskUsecase,
	nowFunc func() time.Time,
) http.Handler {
	return &CreateTaskHandler{
		createUC: createUC,
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

func (h *CreateTaskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	h.handleCreate(w, r)
}

func (h *CreateTaskHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req createTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid json", err.Error())
		return
	}

	status, err := domain.ParseStatus(req.Status)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid status", err.Error())
		return
	}
	priority, err := domain.ParsePriority(req.Priority)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid priority", err.Error())
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
		writeErrorResponse(w, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	resp := taskResponse{
		ID:          t.ID,
		ProjectID:   t.ProjectID,
		Title:       t.Title,
		Description: t.Description,
		Status:      string(t.Status),   // ★ TaskStatus → string
		Priority:    string(t.Priority), // ★ TaskPriority → string
		AssigneeID:  t.AssigneeID,
		DueDate:     t.DueDate,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}
