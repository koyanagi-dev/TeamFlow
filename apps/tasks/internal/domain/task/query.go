package task

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// TaskQuery はタスク検索条件を表すQuery Object。
// 条件定義のみを担当し、実装詳細（フィルタリング・ソート・リミット処理）はリポジトリ層に委譲する。
type TaskQuery struct {
	// Filters
	Statuses    []TaskStatus  // status フィルタ（doing -> in_progress 正規化済み）
	AssigneeID  *string       // assigneeId フィルタ
	Priorities  []TaskPriority // priority フィルタ
	DueDateFrom *time.Time    // dueDateFrom
	DueDateTo   *time.Time    // dueDateTo
	Query       *string       // q (title検索)

	// Sorting
	SortOrders []SortOrder // sort パラメータからパース済み

	// Limit
	Limit int // limit (default 200, max 200, min 1)

	// Cursor
	Cursor *TaskCursor // cursor デコード結果
}

// TaskCursor は cursor のデコード結果を保持する。
type TaskCursor struct {
	CreatedAt time.Time
	ID        string
	ProjectID string
	QHash     string
	IssuedAt  int64
}

// SortOrder はソート順を表す。
type SortOrder struct {
	Key       string // sortOrder, createdAt, updatedAt, dueDate, priority
	Direction string // "ASC" or "DESC"
}

const (
	SortDirectionASC  = "ASC"
	SortDirectionDESC = "DESC"
)

// NewTaskQuery はQuery Objectを構築し、正規化を行う。
// エラーはバリデーションエラーの場合のみ返す。
func NewTaskQuery(opts ...TaskQueryOption) (*TaskQuery, error) {
	q := &TaskQuery{
		Limit: 200, // default
	}

	for _, opt := range opts {
		if err := opt(q); err != nil {
			return nil, err
		}
	}

	// Limit の正規化（1-200にクランプ）
	if q.Limit < 1 {
		q.Limit = 200
	}
	if q.Limit > 200 {
		q.Limit = 200
	}

	return q, nil
}

// TaskQueryOption はQuery Objectの構築オプション。
type TaskQueryOption func(*TaskQuery) error

// WithStatusFilter はstatusフィルタを設定する（カンマ区切り文字列を受け取り、doing -> in_progress を正規化）。
func WithStatusFilter(statusStr string) TaskQueryOption {
	return func(q *TaskQuery) error {
		if statusStr == "" {
			return nil
		}

		parts := strings.Split(statusStr, ",")
		statuses := make([]TaskStatus, 0, len(parts))
		seen := make(map[TaskStatus]bool)

		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			// doing -> in_progress 正規化
			status, err := ParseStatus(part)
			if err != nil {
				return fmt.Errorf("invalid status in filter: %w", err)
			}

			// 重複排除
			if !seen[status] {
				statuses = append(statuses, status)
				seen[status] = true
			}
		}

		q.Statuses = statuses
		return nil
	}
}

// WithPriorityFilter はpriorityフィルタを設定する（カンマ区切り文字列）。
func WithPriorityFilter(priorityStr string) TaskQueryOption {
	return func(q *TaskQuery) error {
		if priorityStr == "" {
			return nil
		}

		parts := strings.Split(priorityStr, ",")
		priorities := make([]TaskPriority, 0, len(parts))
		seen := make(map[TaskPriority]bool)

		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			priority, err := ParsePriority(part)
			if err != nil {
				return fmt.Errorf("invalid priority in filter: %w", err)
			}

			// 重複排除
			if !seen[priority] {
				priorities = append(priorities, priority)
				seen[priority] = true
			}
		}

		q.Priorities = priorities
		return nil
	}
}

// WithAssigneeIDFilter はassigneeIdフィルタを設定する。
func WithAssigneeIDFilter(assigneeID string) TaskQueryOption {
	return func(q *TaskQuery) error {
		if assigneeID == "" {
			return nil
		}
		// UUID形式のバリデーションは簡易的に行う（実際はhandler側でより厳密に）
		q.AssigneeID = &assigneeID
		return nil
	}
}

// WithDueDateRangeFilter はdueDateFrom/Toフィルタを設定する（YYYY-MM-DD形式）。
func WithDueDateRangeFilter(dueDateFromStr, dueDateToStr string) TaskQueryOption {
	return func(q *TaskQuery) error {
		if dueDateFromStr != "" {
			t, err := time.Parse("2006-01-02", dueDateFromStr)
			if err != nil {
				return fmt.Errorf("invalid dueDateFrom format (expected YYYY-MM-DD): %w", err)
			}
			// 日付のみなので時刻は00:00:00に正規化
			from := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
			q.DueDateFrom = &from
		}

		if dueDateToStr != "" {
			t, err := time.Parse("2006-01-02", dueDateToStr)
			if err != nil {
				return fmt.Errorf("invalid dueDateTo format (expected YYYY-MM-DD): %w", err)
			}
			// 日付のみなので時刻は23:59:59に正規化（その日を含むため）
			to := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, time.UTC)
			q.DueDateTo = &to
		}

		return nil
	}
}

// WithQueryFilter はq（タイトル検索）フィルタを設定する。
func WithQueryFilter(queryStr string) TaskQueryOption {
	return func(q *TaskQuery) error {
		if queryStr == "" {
			return nil
		}
		trimmed := strings.TrimSpace(queryStr)
		if trimmed == "" {
			return nil
		}
		q.Query = &trimmed
		return nil
	}
}

// WithSort はsortパラメータをパースして設定する。
// 形式: "-priority,createdAt" (- はDESC、無印はASC)
// 対応キー: sortOrder, createdAt, updatedAt, dueDate, priority
func WithSort(sortStr string) TaskQueryOption {
	return func(q *TaskQuery) error {
		if sortStr == "" {
			return nil
		}

		parts := strings.Split(sortStr, ",")
		orders := make([]SortOrder, 0, len(parts))
		validKeys := map[string]bool{
			"sortOrder": true,
			"createdAt": true,
			"updatedAt": true,
			"dueDate":   true,
			"priority":  true,
		}

		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			key := part
			direction := SortDirectionASC

			if strings.HasPrefix(part, "-") {
				key = strings.TrimPrefix(part, "-")
				direction = SortDirectionDESC
			}

			if !validKeys[key] {
				return fmt.Errorf("invalid sort key: %s (valid keys: sortOrder, createdAt, updatedAt, dueDate, priority)", key)
			}

			orders = append(orders, SortOrder{
				Key:       key,
				Direction: direction,
			})
		}

		q.SortOrders = orders
		return nil
	}
}

// WithLimit はlimitを設定する（正規化はNewTaskQuery内で行われる）。
func WithLimit(limit int) TaskQueryOption {
	return func(q *TaskQuery) error {
		q.Limit = limit
		return nil
	}
}

// Validate はQuery Objectの整合性をチェックする。
func (q *TaskQuery) Validate() error {
	if q.Limit < 1 || q.Limit > 200 {
		return errors.New("limit must be between 1 and 200")
	}

	if q.DueDateFrom != nil && q.DueDateTo != nil {
		if q.DueDateFrom.After(*q.DueDateTo) {
			return errors.New("dueDateFrom must not be after dueDateTo")
		}
	}

	// cursor + sort 併用禁止
	if q.Cursor != nil && len(q.SortOrders) > 0 {
		return errors.New("sort is incompatible with cursor")
	}

	return nil
}

// ComputeQHash はクエリ条件から qhash を計算する。
// projectId と filter/search 等のパラメータを正規化してハッシュ化した短い文字列を返す。
func (q *TaskQuery) ComputeQHash(projectID string) string {
	// 正規化: 複数値（status/priority 等）はソートして join（順序差を吸収）
	parts := []string{}

	// projectID
	parts = append(parts, "projectId:"+projectID)

	// statuses（ソート済み）
	if len(q.Statuses) > 0 {
		statusStrs := make([]string, len(q.Statuses))
		for i, s := range q.Statuses {
			statusStrs[i] = string(s)
		}
		sort.Strings(statusStrs)
		parts = append(parts, "status:"+strings.Join(statusStrs, ","))
	}

	// priorities（ソート済み）
	if len(q.Priorities) > 0 {
		priorityStrs := make([]string, len(q.Priorities))
		for i, p := range q.Priorities {
			priorityStrs[i] = string(p)
		}
		sort.Strings(priorityStrs)
		parts = append(parts, "priority:"+strings.Join(priorityStrs, ","))
	}

	// assigneeId
	if q.AssigneeID != nil {
		parts = append(parts, "assigneeId:"+*q.AssigneeID)
	}

	// dueDateFrom
	if q.DueDateFrom != nil {
		parts = append(parts, "dueDateFrom:"+q.DueDateFrom.Format("2006-01-02"))
	}

	// dueDateTo
	if q.DueDateTo != nil {
		parts = append(parts, "dueDateTo:"+q.DueDateTo.Format("2006-01-02"))
	}

	// q (title検索)
	if q.Query != nil {
		parts = append(parts, "q:"+*q.Query)
	}

	// ソート済みの parts を join
	normalized := strings.Join(parts, "|")

	// sha256 の先頭 8byte を Base64URL でエンコード
	hash := sha256.Sum256([]byte(normalized))
	return base64.RawURLEncoding.EncodeToString(hash[:8])
}

// WithCursor は cursor をデコードし、検証して設定する。
func WithCursor(cursorStr string, projectID string, secret []byte, now time.Time) TaskQueryOption {
	return func(q *TaskQuery) error {
		if cursorStr == "" {
			return nil
		}

		// cursor をデコード
		payload, err := DecodeCursor(cursorStr, secret)
		if err != nil {
			// エラーメッセージをそのまま返す（validation_error.go で判定）
			return err
		}

		// createdAt をパース（micro秒丸め）
		createdAt, err := ParseCursorCreatedAt(payload.CreatedAt)
		if err != nil {
			return errors.New("invalid cursor format")
		}

		// 有効期限チェック
		if err := ValidateCursorExpiry(payload, now); err != nil {
			return err
		}

		// projectID の一致確認
		if payload.ProjectID != projectID {
			return errors.New("cursor query mismatch")
		}

		// qhash の一致確認
		computedQHash := q.ComputeQHash(projectID)
		if computedQHash != payload.QHash {
			return errors.New("cursor query mismatch")
		}

		// TaskCursor を設定
		q.Cursor = &TaskCursor{
			CreatedAt: createdAt,
			ID:        payload.ID,
			ProjectID: payload.ProjectID,
			QHash:     payload.QHash,
			IssuedAt:  payload.IssuedAt,
		}

		return nil
	}
}

