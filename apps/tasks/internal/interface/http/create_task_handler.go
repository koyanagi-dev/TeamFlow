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
			writeErrorResponse(w, http.StatusBadRequest, "validation error", "invalid task id")
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
		writeErrorResponse(w, http.StatusBadRequest, "validation error", "projectId is required")
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

// NullableString は null と未指定を区別するためのカスタム型。
// null を受け取った場合は IsNull が true になり、未指定の場合は nil のまま。
type NullableString struct {
	Value   *string
	IsNull  bool
}

// UnmarshalJSON は JSON の null と未指定を区別する。
func (n *NullableString) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		n.IsNull = true
		n.Value = nil
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	n.Value = &s
	n.IsNull = false
	return nil
}

// PatchTaskRequest は PATCH /api/tasks/{id} のリクエストボディ。
type PatchTaskRequest struct {
	Title    *string        `json:"title"`
	Status   *string        `json:"status"`
	Priority *string        `json:"priority"`
	DueDate  NullableString `json:"dueDate,omitempty"`
}

// UnmarshalJSON は JSON をパースし、dueDate の null と未指定を区別する。
func (p *PatchTaskRequest) UnmarshalJSON(data []byte) error {
	// 一時的な構造体を使用して JSON をパース
	type Alias PatchTaskRequest
	aux := &struct {
		DueDate json.RawMessage `json:"dueDate"`
		*Alias
	}{
		Alias: (*Alias)(p),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// dueDate が指定されている場合のみ処理
	// aux.DueDate が nil の場合は未指定、null の場合は []byte("null")
	if len(aux.DueDate) > 0 {
		if err := p.DueDate.UnmarshalJSON(aux.DueDate); err != nil {
			return err
		}
	} else {
		// 未指定の場合は IsNull = false, Value = nil のまま
		p.DueDate = NullableString{IsNull: false, Value: nil}
	}

	return nil
}

func (h *TaskHandler) handleUpdate(w http.ResponseWriter, r *http.Request, id string) {
	if h.updateUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var req PatchTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid json", err.Error())
		return
	}

	// 全部 nil チェック（dueDate は IsNull または Value が nil でない場合に指定されているとみなす）
	dueDateProvided := req.DueDate.IsNull || req.DueDate.Value != nil
	if req.Title == nil && req.Status == nil && req.Priority == nil && !dueDateProvided {
		writeErrorResponse(w, http.StatusBadRequest, "validation error", "at least one field must be provided")
		return
	}

	var trimmedTitle *string
	if req.Title != nil {
		// title が空文字または空白のみの場合は 400
		trimmed := strings.TrimSpace(*req.Title)
		if trimmed == "" {
			writeErrorResponse(w, http.StatusBadRequest, "validation error", "task title must not be empty")
			return
		}
		trimmedTitle = &trimmed
	}

	var status *domain.TaskStatus
	if req.Status != nil {
		parsed, err := domain.ParseStatus(*req.Status)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "invalid status", err.Error())
			return
		}
		status = &parsed
	}

	var priority *domain.TaskPriority
	if req.Priority != nil {
		parsed, err := domain.ParsePriority(*req.Priority)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "invalid priority", err.Error())
			return
		}
		priority = &parsed
	}

	var dueDate *string
	if dueDateProvided {
		if req.DueDate.IsNull {
			// null の場合は空文字列を渡して削除を指示（usecase で処理）
			empty := ""
			dueDate = &empty
		} else if req.DueDate.Value != nil {
			// RFC3339 形式の検証
			_, err := time.Parse(time.RFC3339, *req.DueDate.Value)
			if err != nil {
				writeErrorResponse(w, http.StatusBadRequest, "validation error", "dueDate must be RFC3339 format")
				return
			}
			dueDate = req.DueDate.Value
		}
	}

	in := usecase.UpdateTaskInput{
		ID:       id,
		Title:    trimmedTitle,
		Status:   status,
		Priority: priority,
		DueDate:  dueDate,
		Now:      h.nowFunc(),
	}

	t, err := h.updateUC.Execute(r.Context(), in)
	if err != nil {
		if errors.Is(err, usecase.ErrTaskNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if errors.Is(err, usecase.ErrInvalidInput) {
			writeErrorResponse(w, http.StatusBadRequest, "validation error", err.Error())
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

type errorResponse struct {
	Error  string `json:"error"`
	Detail string `json:"detail"`
}

func writeErrorResponse(w http.ResponseWriter, statusCode int, errorMsg, detail string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	resp := errorResponse{
		Error:  errorMsg,
		Detail: detail,
	}
	_ = json.NewEncoder(w).Encode(resp)
}
