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

// UpdateTaskHandler は PATCH /tasks/{id} を処理する HTTP ハンドラ。
//
// 責務:
//   - PATCH /api/tasks/{id} エンドポイントのリクエストを受け付ける
//   - パスパラメータからタスクIDを抽出する
//   - リクエストボディのJSONをパースし、部分更新用のPatch型に変換する
//   - 各フィールドのバリデーションを行う（titleの空文字チェック、assigneeIdのUUID形式チェック、dueDateのRFC3339形式チェックなど）
//   - UpdateTaskUsecaseを呼び出してタスクを更新する
//   - 更新されたタスクをJSONレスポンスとして返す
type UpdateTaskHandler struct {
	updateUC *usecase.UpdateTaskUsecase
}

// NewUpdateTaskHandler は UpdateTaskHandler を生成する。
func NewUpdateTaskHandler(
	updateUC *usecase.UpdateTaskUsecase,
) http.Handler {
	return &UpdateTaskHandler{
		updateUC: updateUC,
	}
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

func (h *UpdateTaskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// /api/tasks/{id} または /tasks/{id} から id を抽出
	var path string
	if strings.HasPrefix(r.URL.Path, "/api/tasks/") {
		path = strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	} else if strings.HasPrefix(r.URL.Path, "/tasks/") {
		path = strings.TrimPrefix(r.URL.Path, "/tasks/")
	} else {
		writeErrorResponse(w, http.StatusBadRequest, "validation error", "invalid task id")
		return
	}

	if path == "" || strings.Contains(path, "/") {
		writeErrorResponse(w, http.StatusBadRequest, "validation error", "invalid task id")
		return
	}

	h.handleUpdate(w, r, path)
}

func (h *UpdateTaskHandler) handleUpdate(w http.ResponseWriter, r *http.Request, id string) {
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
