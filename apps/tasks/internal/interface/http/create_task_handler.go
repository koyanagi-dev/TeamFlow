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

// OptionalString は JSON で null と未指定を区別するための型。
// - 未指定: nil
// - null: &OptionalString{Value: nil, IsSet: true}
// - 値あり: &OptionalString{Value: &str, IsSet: true}
type OptionalString struct {
	Value *string
	IsSet bool
}

// UnmarshalJSON は JSON を Unmarshal し、null と未指定を区別する。
func (o *OptionalString) UnmarshalJSON(data []byte) error {
	o.IsSet = true
	if string(data) == "null" {
		o.Value = nil
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	o.Value = &s
	return nil
}

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
	AssigneeID  *string    `json:"assigneeId"`
	DueDate     *time.Time `json:"dueDate"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

func (h *TaskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// /projects/{projectId}/tasks の処理
	if strings.HasPrefix(r.URL.Path, "/projects/") && strings.HasSuffix(r.URL.Path, "/tasks") {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		// /projects/{projectId}/tasks から projectId を抽出
		path := strings.TrimPrefix(r.URL.Path, "/projects/")
		path = strings.TrimSuffix(path, "/tasks")
		projectID := path
		h.handleListByProjectWithQuery(w, r, projectID)
		return
	}

	// /tasks/{id} の処理
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

	// /tasks の処理（既存API、後方互換性のため残す）
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
		AssigneeID:  t.AssigneeID,
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
			AssigneeID:  t.AssigneeID,
			DueDate:     t.DueDate,
			CreatedAt:   t.CreatedAt,
			UpdatedAt:   t.UpdatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(responses)
}

// handleListByProjectWithQuery は /projects/{projectId}/tasks を処理する（Query Objectを使用）。
func (h *TaskHandler) handleListByProjectWithQuery(w http.ResponseWriter, r *http.Request, projectID string) {
	if h.listUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if projectID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "validation error", "projectId is required")
		return
	}

	// Query Object を構築
	opts := []domain.TaskQueryOption{}

	// status フィルタ（カンマ区切り）
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		opts = append(opts, domain.WithStatusFilter(statusStr))
	}

	// priority フィルタ（カンマ区切り）
	if priorityStr := r.URL.Query().Get("priority"); priorityStr != "" {
		opts = append(opts, domain.WithPriorityFilter(priorityStr))
	}

	// assigneeId フィルタ
	if assigneeID := r.URL.Query().Get("assigneeId"); assigneeID != "" {
		if !isValidUUID(assigneeID) {
			writeErrorResponse(w, http.StatusBadRequest, "validation error", "assigneeId must be a valid UUID")
			return
		}
		opts = append(opts, domain.WithAssigneeIDFilter(assigneeID))
	}

	// dueDateFrom / dueDateTo フィルタ
	dueDateFrom := r.URL.Query().Get("dueDateFrom")
	dueDateTo := r.URL.Query().Get("dueDateTo")
	if dueDateFrom != "" || dueDateTo != "" {
		opts = append(opts, domain.WithDueDateRangeFilter(dueDateFrom, dueDateTo))
	}

	// q フィルタ（タイトル検索）
	if queryStr := r.URL.Query().Get("q"); queryStr != "" {
		opts = append(opts, domain.WithQueryFilter(queryStr))
	}

	// sort
	if sortStr := r.URL.Query().Get("sort"); sortStr != "" {
		opts = append(opts, domain.WithSort(sortStr))
	}

	// limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, err := ParseLimit(limitStr)
		if err != nil {
			issue := toValidationIssue(err)
			resp := NewValidationErrorResponse(issue)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		if limit > 0 {
			opts = append(opts, domain.WithLimit(limit))
		}
	}

	// cursor パラメータは予約席なので受け取るが無視
	_ = r.URL.Query().Get("cursor")

	// Query Object を作成
	query, err := domain.NewTaskQuery(opts...)
	if err != nil {
		issue := toValidationIssue(err)
		resp := NewValidationErrorResponse(issue)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	// Query Object のバリデーション
	if err := query.Validate(); err != nil {
		issue := toValidationIssue(err)
		resp := NewValidationErrorResponse(issue)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	// Usecase を実行
	tasks, err := h.listUC.ExecuteWithQuery(r.Context(), usecase.ListTasksByProjectWithQueryInput{
		ProjectID: projectID,
		Query:     query,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// レスポンス形式: { "tasks": [...], "page": null } (OpenAPI仕様に準拠)
	type pageInfo struct {
		NextCursor *string `json:"nextCursor,omitempty"`
		Limit      int     `json:"limit,omitempty"`
	}

	type listTasksResponse struct {
		Tasks []taskResponse `json:"tasks"`
		Page  *pageInfo      `json:"page,omitempty"`
	}

	responses := make([]taskResponse, 0, len(tasks))
	for _, t := range tasks {
		responses = append(responses, taskResponse{
			ID:          t.ID,
			ProjectID:   t.ProjectID,
			Title:       t.Title,
			Description: t.Description,
			Status:      string(t.Status),
			Priority:    string(t.Priority),
			AssigneeID:  t.AssigneeID,
			DueDate:     t.DueDate,
			CreatedAt:   t.CreatedAt,
			UpdatedAt:   t.UpdatedAt,
		})
	}

	// page は今は nil でOK（cursor導入時に利用）
	// 検索結果が 0 件でも 200 + tasks: [] を返す
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(listTasksResponse{
		Tasks: responses,
		Page:  nil,
	})
}

type updateTaskRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	AssigneeID  *string `json:"assigneeId"`
	Priority    *string `json:"priority"`
	DueDate     *string `json:"dueDate"`
}

// nullableString は JSON で null を受け取ることができる文字列型。
// UnmarshalJSON で null と未指定を区別するため、null の場合は存在フラグを立てる。
type nullableString struct {
	value   *string
	isNull  bool
	present bool
}

func (ns *nullableString) UnmarshalJSON(data []byte) error {
	ns.present = true
	var s *string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s == nil {
		ns.isNull = true
		ns.value = nil
	} else {
		ns.isNull = false
		ns.value = s
	}
	return nil
}

func (ns *nullableString) toPtr() *string {
	if !ns.present {
		return nil // 未指定
	}
	if ns.isNull {
		empty := ""
		return &empty // null の場合は空文字列を返す
	}
	return ns.value // 文字列が指定された場合
}

// PatchTaskRequest は PATCH /api/tasks/{id} のリクエストボディ。
type PatchTaskRequest struct {
	Title       *string        `json:"title"`
	Description nullableString `json:"description"`
	Status      *string        `json:"status"`
	Priority    *string        `json:"priority"`
	AssigneeID  OptionalString `json:"assigneeId"`
	DueDate     nullableString `json:"dueDate"`
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

	// 全部 nil チェック
	if req.Title == nil &&
		req.Status == nil &&
		req.Priority == nil &&
		!req.Description.present &&
		!req.AssigneeID.IsSet &&
		!req.DueDate.present {
		writeErrorResponse(w, http.StatusBadRequest, "validation error", "at least one field must be provided")
		return
	}

	// Title
	var titlePatch domain.Patch[string]
	if req.Title != nil {
		trimmed := strings.TrimSpace(*req.Title)
		if trimmed == "" {
			writeErrorResponse(w, http.StatusBadRequest, "validation error", "task title must not be empty")
			return
		}
		titlePatch = domain.Set(trimmed)
	}

	// Description
	var descriptionPatch domain.Patch[string]
	if req.Description.present {
		if req.Description.isNull {
			descriptionPatch = domain.Null[string]()
		} else {
			descriptionPatch = domain.Set(*req.Description.value)
		}
	}

	// Status (Usecase 層で Parse するため、文字列のまま渡す)
	var statusStr *string
	if req.Status != nil {
		statusStr = req.Status
	}

	// Priority (Usecase 層で Parse するため、文字列のまま渡す)
	var priorityStr *string
	if req.Priority != nil {
		priorityStr = req.Priority
	}

	// AssigneeID
	var assigneeIDPatch domain.Patch[string]
	if req.AssigneeID.IsSet {
		if req.AssigneeID.Value != nil {
			// UUID 形式のバリデーション
			uuidStr := *req.AssigneeID.Value
			if !isValidUUID(uuidStr) {
				writeErrorResponse(w, http.StatusBadRequest, "validation error", "assigneeId must be a valid UUID")
				return
			}
			assigneeIDPatch = domain.Set(uuidStr)
		} else {
			assigneeIDPatch = domain.Null[string]()
		}
	}

	// DueDate
	var dueDatePatch domain.Patch[time.Time]
	if req.DueDate.present {
		if req.DueDate.isNull {
			dueDatePatch = domain.Null[time.Time]()
		} else {
			parsed, err := time.Parse(time.RFC3339, *req.DueDate.value)
			if err != nil {
				writeErrorResponse(w, http.StatusBadRequest, "validation error", "dueDate must be RFC3339")
				return
			}
			dueDatePatch = domain.Set(parsed)
		}
	}

	in := usecase.UpdateTaskInput{
		ID:          id,
		Title:       titlePatch,
		Description: descriptionPatch,
		StatusStr:   statusStr,
		PriorityStr: priorityStr,
		AssigneeID:  assigneeIDPatch,
		DueDate:     dueDatePatch,
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
		AssigneeID:  t.AssigneeID,
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

// isValidUUID は文字列が有効な UUID 形式かどうかをチェックする。
func isValidUUID(s string) bool {
	// UUID 形式: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx (36文字)
	if len(s) != 36 {
		return false
	}
	parts := strings.Split(s, "-")
	if len(parts) != 5 {
		return false
	}
	expectedLengths := []int{8, 4, 4, 4, 12}
	for i, part := range parts {
		if len(part) != expectedLengths[i] {
			return false
		}
		for _, r := range part {
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
				return false
			}
		}
	}
	return true
}
