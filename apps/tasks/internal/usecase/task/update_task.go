package task

import (
	"context"
	"errors"
	"fmt"
	"time"

	domain "teamflow-tasks/internal/domain/task"
)

// UpdateTaskInput はタスク更新ユースケースの入力。
// フィールドは PATCH を想定し、nil の場合は更新しない。
type UpdateTaskInput struct {
	ID          string
	Title       *string
	Description *string
	Status      *domain.TaskStatus
	AssigneeID  *string
	Priority    *domain.TaskPriority
	DueDate     *string
	Now         time.Time
}

// UpdateTaskUsecase はタスク更新ユースケースを表す。
type UpdateTaskUsecase struct {
	Repo TaskRepository
}

// Execute は既存タスクを取得し、指定されたフィールドを更新する。
func (uc *UpdateTaskUsecase) Execute(ctx context.Context, in UpdateTaskInput) (*domain.Task, error) {
	existing, err := uc.Repo.FindByID(ctx, in.ID)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			return nil, fmt.Errorf("%w: %v", ErrTaskNotFound, err)
		}
		return nil, err
	}

	var dueDate *time.Time
	if in.DueDate != nil {
		if *in.DueDate == "" {
			return nil, fmt.Errorf("%w: dueDate must be RFC3339 when provided", ErrInvalidInput)
		}
		parsed, err := time.Parse(time.RFC3339, *in.DueDate)
		if err != nil {
			return nil, fmt.Errorf("%w: dueDate must be RFC3339: %v", ErrInvalidInput, err)
		}
		dueDate = &parsed
	}

	if err := existing.Update(in.Title, in.Description, in.Status, in.Priority, dueDate, in.Now); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	if err := uc.Repo.Update(ctx, existing); err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			return existing, fmt.Errorf("%w: %v", ErrTaskNotFound, err)
		}
		return existing, err
	}

	return existing, nil
}
