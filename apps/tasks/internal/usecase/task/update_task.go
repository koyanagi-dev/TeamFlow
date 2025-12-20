package task

import (
	"context"
	"errors"
	"fmt"
	"time"

	domain "teamflow-tasks/internal/domain/task"
)

// UpdateTaskInput はタスク更新ユースケースの入力。
// HTTP 層から受け取った情報を TaskPatch に変換する。
type UpdateTaskInput struct {
	ID          string
	Title       domain.Patch[string]
	Description domain.Patch[string]
	StatusStr   *string
	PriorityStr *string
	AssigneeID  domain.Patch[string]
	DueDate     domain.Patch[time.Time]
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

	// TaskPatch を組み立てる
	patch := domain.TaskPatch{}

	// Title
	patch.Title = in.Title

	// Description
	patch.Description = in.Description

	// Status (Usecase 層で Parse)
	if in.StatusStr != nil {
		parsed, err := domain.ParseStatus(*in.StatusStr)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
		}
		patch.Status = domain.Set(parsed)
	}

	// Priority (Usecase 層で Parse)
	if in.PriorityStr != nil {
		parsed, err := domain.ParsePriority(*in.PriorityStr)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
		}
		patch.Priority = domain.Set(parsed)
	}

	// AssigneeID
	patch.AssigneeID = in.AssigneeID

	// DueDate
	patch.DueDate = in.DueDate

	if err := existing.ApplyPatch(patch); err != nil {
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
