package task

import (
	"errors"
	"time"
)

// TaskStatus はタスクの状態を表す型。
type TaskStatus string

const (
	StatusTodo       TaskStatus = "todo"
	StatusInProgress TaskStatus = "in_progress"
	StatusDone       TaskStatus = "done"
)

// TaskPriority はタスクの優先度を表す型。
type TaskPriority string

const (
	PriorityLow    TaskPriority = "low"
	PriorityMedium TaskPriority = "medium"
	PriorityHigh   TaskPriority = "high"
)

// Task は TeamFlow におけるタスクのドメインモデル。
type Task struct {
	ID          string
	ProjectID   string
	Title       string
	Description string
	Status      TaskStatus
	Priority    TaskPriority
	DueDate     *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewTask は新しいタスクを生成する。
func NewTask(
	id string,
	projectID string,
	title string,
	description string,
	status TaskStatus,
	priority TaskPriority,
	dueDate *time.Time,
	now time.Time,
) (*Task, error) {
	if title == "" {
		return nil, errors.New("task title must not be empty")
	}

	if !isValidStatus(status) {
		return nil, errors.New("invalid task status")
	}

	if !isValidPriority(priority) {
		return nil, errors.New("invalid task priority")
	}

	return &Task{
		ID:          id,
		ProjectID:   projectID,
		Title:       title,
		Description: description,
		Status:      status,
		Priority:    priority,
		DueDate:     dueDate,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// Update はタスクの一部または全てのフィールドを更新する。
// nil のフィールドは変更しない。
func (t *Task) Update(
	title *string,
	description *string,
	status *TaskStatus,
	dueDate *time.Time,
	now time.Time,
) error {
	if title != nil {
		if *title == "" {
			return errors.New("task title must not be empty")
		}
		t.Title = *title
	}

	if description != nil {
		t.Description = *description
	}

	if status != nil {
		if !isValidStatus(*status) {
			return errors.New("invalid task status")
		}
		t.Status = *status
	}

	if dueDate != nil {
		t.DueDate = dueDate
	}

	t.UpdatedAt = now
	return nil
}

func isValidStatus(s TaskStatus) bool {
	switch s {
	case StatusTodo, StatusInProgress, StatusDone:
		return true
	default:
		return false
	}
}

func isValidPriority(p TaskPriority) bool {
	switch p {
	case PriorityLow, PriorityMedium, PriorityHigh:
		return true
	default:
		return false
	}
}
