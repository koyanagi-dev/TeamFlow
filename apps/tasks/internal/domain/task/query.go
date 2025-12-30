package task

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// TaskQuery はタスク検索条件を表すQuery Object。
// フィルタ、ソート、リミットの正規化ロジックを含む。
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

// PrioritySortValue はpriorityをソート用の数値に変換する（high > medium > low）。
// SQLのCASE文と同等のロジック。
func PrioritySortValue(p TaskPriority) int {
	switch p {
	case PriorityHigh:
		return 3
	case PriorityMedium:
		return 2
	case PriorityLow:
		return 1
	default:
		return 0
	}
}

// CompareTasks はTaskQueryのソート条件に従ってタスクを比較する。
// sort.Slice の比較関数として使用可能。
// 注意: メモリリポジトリ用。SQL実装ではORDER BY句に変換される。
func (q *TaskQuery) CompareTasks(t1, t2 *Task) bool {
	if len(q.SortOrders) == 0 {
		// デフォルトソート: createdAt ASC
		return t1.CreatedAt.Before(t2.CreatedAt)
	}

	for _, order := range q.SortOrders {
		cmp := q.compareByKey(t1, t2, order.Key, order.Direction)
		if cmp != 0 {
			if order.Direction == SortDirectionDESC {
				return cmp > 0
			}
			return cmp < 0
		}
	}

	// すべてのソートキーで等しい場合はIDで安定ソート
	return t1.ID < t2.ID
}

// compareByKey は指定されたキーで2つのタスクを比較する。
// direction はASC/DESCを指定し、dueDateのnull値処理に使用される。
// 戻り値: <0 (t1 < t2), 0 (t1 == t2), >0 (t1 > t2)
func (q *TaskQuery) compareByKey(t1, t2 *Task, key string, direction string) int {
	switch key {
	case "sortOrder":
		// sortOrder は現在Taskエンティティにないため、0を返す（将来対応）
		return 0

	case "createdAt":
		if t1.CreatedAt.Before(t2.CreatedAt) {
			return -1
		}
		if t1.CreatedAt.After(t2.CreatedAt) {
			return 1
		}
		return 0

	case "updatedAt":
		if t1.UpdatedAt.Before(t2.UpdatedAt) {
			return -1
		}
		if t1.UpdatedAt.After(t2.UpdatedAt) {
			return 1
		}
		return 0

	case "dueDate":
		// null値の処理:
		// ASC時はnullを最後（nullを最大値として扱う）
		// DESC時はnullを最初（nullを最小値として扱う）
		if t1.DueDate == nil && t2.DueDate == nil {
			return 0
		}
		if t1.DueDate == nil {
			if direction == SortDirectionDESC {
				return -1 // DESC時はnullを先頭（t1 < t2）
			}
			return 1 // ASC時はnullを最後（t1 > t2）
		}
		if t2.DueDate == nil {
			if direction == SortDirectionDESC {
				return 1 // DESC時はnullを先頭（t1 > t2）
			}
			return -1 // ASC時はnullを最後（t1 < t2）
		}
		if t1.DueDate.Before(*t2.DueDate) {
			return -1
		}
		if t1.DueDate.After(*t2.DueDate) {
			return 1
		}
		return 0

	case "priority":
		v1 := PrioritySortValue(t1.Priority)
		v2 := PrioritySortValue(t2.Priority)
		return v1 - v2

	default:
		return 0
	}
}

// FilterTasks はタスクのスライスをフィルタする（メモリリポジトリ用）。
func (q *TaskQuery) FilterTasks(tasks []*Task) []*Task {
	var result []*Task

	for _, t := range tasks {
		if !q.matches(t) {
			continue
		}
		result = append(result, t)
	}

	return result
}

// matches はタスクがフィルタ条件に一致するかチェックする。
func (q *TaskQuery) matches(t *Task) bool {
	// Status filter
	if len(q.Statuses) > 0 {
		found := false
		for _, status := range q.Statuses {
			if t.Status == status {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// AssigneeID filter
	if q.AssigneeID != nil {
		if t.AssigneeID == nil || *t.AssigneeID != *q.AssigneeID {
			return false
		}
	}

	// Priority filter
	if len(q.Priorities) > 0 {
		found := false
		for _, priority := range q.Priorities {
			if t.Priority == priority {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// DueDate range filter
	if q.DueDateFrom != nil {
		if t.DueDate == nil || t.DueDate.Before(*q.DueDateFrom) {
			return false
		}
	}
	if q.DueDateTo != nil {
		if t.DueDate == nil || t.DueDate.After(*q.DueDateTo) {
			return false
		}
	}

	// Query filter (title search)
	if q.Query != nil {
		if !strings.Contains(strings.ToLower(t.Title), strings.ToLower(*q.Query)) {
			return false
		}
	}

	return true
}

// SortTasks はタスクのスライスをソートする（メモリリポジトリ用）。
func (q *TaskQuery) SortTasks(tasks []*Task) {
	sort.Slice(tasks, func(i, j int) bool {
		return q.CompareTasks(tasks[i], tasks[j])
	})
}

// ApplyLimit はタスクのスライスをリミットする。
func (q *TaskQuery) ApplyLimit(tasks []*Task) []*Task {
	if len(tasks) <= q.Limit {
		return tasks
	}
	return tasks[:q.Limit]
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

	return nil
}

