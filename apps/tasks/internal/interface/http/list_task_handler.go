package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	domain "teamflow-tasks/internal/domain/task"
	usecase "teamflow-tasks/internal/usecase/task"
)

// ListTaskHandler は GET /tasks と GET /projects/{projectId}/tasks を処理する HTTP ハンドラ。
//
// 責務:
//   - GET /api/tasks?projectId=xxx エンドポイントのリクエストを受け付ける（旧API、後方互換性のため）
//   - GET /api/projects/{projectId}/tasks エンドポイントのリクエストを受け付ける（新API）
//   - クエリパラメータ（status, priority, assigneeId, dueDateFrom, dueDateTo, q, sort, cursor, limit）をパースし、TaskQueryを構築する
//   - ListTasksByProjectUsecaseを呼び出してタスク一覧を取得する
//   - カーソルページネーションの場合はnextCursorを計算してレスポンスに含める
//   - 取得したタスク一覧をJSONレスポンスとして返す
type ListTaskHandler struct {
	listUC       *usecase.ListTasksByProjectUsecase
	nowFunc      func() time.Time
	cursorSecret []byte
}

// NewListTaskHandler は ListTaskHandler を生成する。
func NewListTaskHandler(
	listUC *usecase.ListTasksByProjectUsecase,
	nowFunc func() time.Time,
	cursorSecret []byte,
) http.Handler {
	return &ListTaskHandler{
		listUC:       listUC,
		nowFunc:      nowFunc,
		cursorSecret: cursorSecret,
	}
}

func (h *ListTaskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// /api/projects/{projectId}/tasks の処理
	if strings.HasPrefix(r.URL.Path, "/api/projects/") && strings.HasSuffix(r.URL.Path, "/tasks") {
		// /api/projects/{projectId}/tasks から projectId を抽出
		path := strings.TrimPrefix(r.URL.Path, "/api/projects/")
		path = strings.TrimSuffix(path, "/tasks")
		projectID := path
		h.handleListByProjectWithQuery(w, r, projectID)
		return
	}

	// /api/tasks?projectId=xxx の処理（旧API、後方互換性のため残す）
	// /tasks も後方互換性のためサポート
	if r.URL.Path == "/api/tasks" || r.URL.Path == "/tasks" {
		h.handleListByProject(w, r)
		return
	}

	w.WriteHeader(http.StatusNotFound)
}

func (h *ListTaskHandler) handleListByProject(w http.ResponseWriter, r *http.Request) {
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
func (h *ListTaskHandler) handleListByProjectWithQuery(w http.ResponseWriter, r *http.Request, projectID string) {
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

	// cursor と sort の併用チェック（cursor がある場合、sort は指定不可）
	cursor := r.URL.Query().Get("cursor")
	sortStr := r.URL.Query().Get("sort")
	if cursor != "" && sortStr != "" {
		rejected := sortStr
		issue := ValidationIssue{
			Location:      "query",
			Field:         "sort",
			Code:          "INCOMPATIBLE_WITH_CURSOR",
			Message:       "cursor を使用する場合、sort は指定できません。",
			RejectedValue: &rejected,
		}
		resp := NewValidationErrorResponse(issue)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	// sort（cursor がない場合のみ）
	if sortStr != "" {
		opts = append(opts, domain.WithSort(sortStr))
	}

	// cursor（cursor がある場合）
	if cursor != "" {
		opts = append(opts, domain.WithCursor(cursor, projectID, h.cursorSecret, h.nowFunc()))
	}

	// limit の default=200 を HTTP 層で明示
	limit := 200
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		v, err := ParseLimit(limitStr)
		if err != nil {
			issue := toValidationIssue(err)
			resp := NewValidationErrorResponse(issue)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		// ParseLimit 成功時は v>0 のはず
		limit = v
	}
	opts = append(opts, domain.WithLimit(limit))

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

	// レスポンス形式: { "tasks": [...], "page": {...} } (OpenAPI仕様に準拠)
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

	// nextCursor の計算
	var nextCursor *string
	// repository 層で limit + 1 件取得している
	// limit + 1 件取得できた場合（次ページが存在する場合）、limit 件目を使って nextCursor を生成し、limit 件だけ返す
	// 1ページ目（cursor なし）でも次ページがあれば nextCursor を返す
	if len(tasks) > query.Limit {
		// limit 件目（インデックス query.Limit-1）を使って nextCursor を生成
		lastTask := tasks[query.Limit-1]
		payload := domain.CursorPayload{
			V:         1,
			CreatedAt: domain.FormatCursorCreatedAt(lastTask.CreatedAt),
			ID:        lastTask.ID,
			ProjectID: projectID,
			QHash:     query.ComputeQHash(projectID),
			IssuedAt:  h.nowFunc().Unix(),
		}
		cursor, err := domain.EncodeCursor(payload, h.cursorSecret)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		nextCursor = &cursor
		// レスポンスから limit + 1 件目を除外（limit 件だけ返す）
		responses = responses[:query.Limit]
	}

	// page を返す
	page := &pageInfo{
		NextCursor: nextCursor,
		Limit:      query.Limit,
	}

	// 検索結果が 0 件でも 200 + tasks: [] を返す
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(listTasksResponse{
		Tasks: responses,
		Page:  page,
	})
}
