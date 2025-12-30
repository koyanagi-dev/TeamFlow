package taskinfra

import (
	"context"
	"sort"

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
	filtered := query.FilterTasks(candidates)

	// Query Object のソートを適用
	query.SortTasks(filtered)

	// Query Object のリミットを適用
	result := query.ApplyLimit(filtered)

	return result, nil
}
