package taskinfra

import (
	"context"
	"errors"

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
var ErrTaskNotFound = errors.New("task not found")

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

// ListByProject は指定された projectID のタスク一覧を返す。
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
	return out, nil
}
