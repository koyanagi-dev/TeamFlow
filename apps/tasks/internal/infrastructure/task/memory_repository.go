package taskinfra

import (
	"context"
	"sort"
	"strings"

	domain "teamflow-tasks/internal/domain/task"
	usecase "teamflow-tasks/internal/usecase/task"
)

// MemoryTaskRepository はメモリ上にタスクを保持するシンプルな実装。
type MemoryTaskRepository struct {
	tasks map[string]*domain.Task
}

// コンパイル時にインターフェース実装を保証する。
var _ usecase.TaskRepository = (*MemoryTaskRepository)(nil)

// ErrTaskNotFound は指定 ID のタスクが存在しない場合に返す。
var ErrTaskNotFound = usecase.ErrTaskNotFound

// NewMemoryTaskRepository は空のインメモリリポジトリを生成する。
func NewMemoryTaskRepository() *MemoryTaskRepository {
	return &MemoryTaskRepository{
		tasks: make(map[string]*domain.Task),
	}
}

// Save はタスクを保存する。
// タスク ID をキーにして複数タスクを独立して保存できる状態にする。
func (r *MemoryTaskRepository) Save(_ context.Context, t *domain.Task) error {
	if r.tasks == nil {
		r.tasks = make(map[string]*domain.Task)
	}
	r.tasks[t.ID] = t // ★ これが非常に重要（taskID をキーにする）
	return nil
}

// Update は既存タスクを上書き保存する。
func (r *MemoryTaskRepository) Update(_ context.Context, t *domain.Task) error {
	if r.tasks == nil {
		return ErrTaskNotFound
	}
	if _, ok := r.tasks[t.ID]; !ok {
		return ErrTaskNotFound
	}
	r.tasks[t.ID] = t
	return nil
}

// FindByID は ID を指定してタスクを取得する。
func (r *MemoryTaskRepository) FindByID(_ context.Context, id string) (*domain.Task, error) {
	if r.tasks == nil {
		return nil, ErrTaskNotFound
	}
	task, ok := r.tasks[id]
	if !ok {
		return nil, ErrTaskNotFound
	}
	return task, nil
}

// ListByProject は指定された projectID のタスク一覧を返す（後方互換性のため残す）。
func (r *MemoryTaskRepository) ListByProject(_ context.Context, projectID string) ([]*domain.Task, error) {
	if r.tasks == nil {
		return []*domain.Task{}, nil
	}

	out := make([]*domain.Task, 0)
	for _, t := range r.tasks {
		if t.ProjectID == projectID {
			out = append(out, t)
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out, nil
}

// FindByProjectID は指定された projectID と Query Object に基づいてタスクを取得する。
func (r *MemoryTaskRepository) FindByProjectID(_ context.Context, projectID string, query *domain.TaskQuery) ([]*domain.Task, error) {
	if r.tasks == nil {
		return []*domain.Task{}, nil
	}

	// まず projectID でフィルタ
	candidates := make([]*domain.Task, 0)
	for _, t := range r.tasks {
		if t.ProjectID == projectID {
			candidates = append(candidates, t)
		}
	}

	// Query Object のフィルタを適用
	filtered := r.filterTasks(candidates, query)

	// Query Object のソートを適用
	r.sortTasks(filtered, query)

	// Query Object のリミットを適用
	result := r.applyLimit(filtered, query)

	return result, nil
}

// filterTasks はタスクのスライスをフィルタする（メモリリポジトリ用）。
func (r *MemoryTaskRepository) filterTasks(tasks []*domain.Task, query *domain.TaskQuery) []*domain.Task {
	var result []*domain.Task

	for _, t := range tasks {
		if r.matches(t, query) {
			result = append(result, t)
		}
	}

	return result
}

// matches はタスクがフィルタ条件に一致するかチェックする。
func (r *MemoryTaskRepository) matches(t *domain.Task, query *domain.TaskQuery) bool {
	// Status filter
	if len(query.Statuses) > 0 {
		found := false
		for _, status := range query.Statuses {
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
	if query.AssigneeID != nil {
		if t.AssigneeID == nil || *t.AssigneeID != *query.AssigneeID {
			return false
		}
	}

	// Priority filter
	if len(query.Priorities) > 0 {
		found := false
		for _, priority := range query.Priorities {
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
	if query.DueDateFrom != nil {
		if t.DueDate == nil || t.DueDate.Before(*query.DueDateFrom) {
			return false
		}
	}
	if query.DueDateTo != nil {
		if t.DueDate == nil || t.DueDate.After(*query.DueDateTo) {
			return false
		}
	}

	// Query filter (title search)
	if query.Query != nil {
		if !strings.Contains(strings.ToLower(t.Title), strings.ToLower(*query.Query)) {
			return false
		}
	}

	return true
}

// sortTasks はタスクのスライスをソートする（メモリリポジトリ用）。
func (r *MemoryTaskRepository) sortTasks(tasks []*domain.Task, query *domain.TaskQuery) {
	sort.Slice(tasks, func(i, j int) bool {
		return r.compareTasks(tasks[i], tasks[j], query)
	})
}

// compareTasks はTaskQueryのソート条件に従ってタスクを比較する。
// sort.Slice の比較関数として使用可能。
func (r *MemoryTaskRepository) compareTasks(t1, t2 *domain.Task, query *domain.TaskQuery) bool {
	if len(query.SortOrders) == 0 {
		// デフォルトソート: createdAt ASC
		return t1.CreatedAt.Before(t2.CreatedAt)
	}

	for _, order := range query.SortOrders {
		cmp := r.compareByKey(t1, t2, order.Key, order.Direction)
		if cmp != 0 {
			if order.Direction == domain.SortDirectionDESC {
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
// dueDate の null は以下の順序で扱う:
// ASC: null last (SQL: NULLS LAST)
// DESC: null first (SQL: NULLS FIRST)
func (r *MemoryTaskRepository) compareByKey(t1, t2 *domain.Task, key string, direction string) int {
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
		// dueDate の null は以下の順序で扱う:
		// ASC: null last (SQL: NULLS LAST)
		// DESC: null first (SQL: NULLS FIRST)
		if t1.DueDate == nil && t2.DueDate == nil {
			return 0
		}
		if t1.DueDate == nil {
			if direction == domain.SortDirectionDESC {
				return -1 // DESC時はnullを先頭（t1 < t2）
			}
			return 1 // ASC時はnullを最後（t1 > t2）
		}
		if t2.DueDate == nil {
			if direction == domain.SortDirectionDESC {
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
		return t1.Priority.CompareTo(t2.Priority)

	default:
		return 0
	}
}

// applyLimit はタスクのスライスをリミットする。
func (r *MemoryTaskRepository) applyLimit(tasks []*domain.Task, query *domain.TaskQuery) []*domain.Task {
	if len(tasks) <= query.Limit {
		return tasks
	}
	return tasks[:query.Limit]
}
