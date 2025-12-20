package task

import (
	"errors"
	"fmt"
	"time"
)

// TaskStatus はタスクの状態を表す型。
type TaskStatus string

const (
	StatusTodo       TaskStatus = "todo"
	StatusInProgress TaskStatus = "in_progress"
	StatusDone       TaskStatus = "done"
)

// ParseStatus 正規の TaskStatus か検証し、型付きで返す。
// "doing" は "in_progress" に正規化される。
func ParseStatus(s string) (TaskStatus, error) {
	input := s
	if s == "doing" {
		s = "in_progress"
	}
	switch TaskStatus(s) {
	case StatusTodo, StatusInProgress, StatusDone:
		return TaskStatus(s), nil
	default:
		return "", fmt.Errorf("invalid task status: %s", input)
	}
}

// TaskPriority はタスクの優先度を表す型。
type TaskPriority string

const (
	PriorityLow    TaskPriority = "low"
	PriorityMedium TaskPriority = "medium"
	PriorityHigh   TaskPriority = "high"
)

// ParsePriority 正規の TaskPriority か検証し、型付きで返す。
func ParsePriority(p string) (TaskPriority, error) {
	switch TaskPriority(p) {
	case PriorityLow, PriorityMedium, PriorityHigh:
		return TaskPriority(p), nil
	default:
		return "", fmt.Errorf("invalid task priority: %s", p)
	}
}

// Task は TeamFlow におけるタスクのドメインモデル。
type Task struct {
	ID          string
	ProjectID   string
	Title       string
	Description string
	Status      TaskStatus
	Priority    TaskPriority
	AssigneeID  *string
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

	if err := validateStatus(status); err != nil {
		return nil, err
	}

	if err := validatePriority(priority); err != nil {
		return nil, err
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
	priority *TaskPriority,
	assigneeID *string,
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
		if err := validateStatus(*status); err != nil {
			return err
		}
		t.Status = *status
	}

	if priority != nil {
		if err := validatePriority(*priority); err != nil {
			return err
		}
		t.Priority = *priority
	}

	// assigneeID は nil かどうかに関わらず設定する
	// nil が渡された場合は明示的に nil を設定する（担当者を外す）
	// ただし、これは呼び出し側で「未指定」と「null」を区別できることが前提
	// 実際には、呼び出し側（usecase層）で「未指定」の場合は nil を渡さないようにする必要がある
	// ここでは単純に、渡された値をそのまま設定する
	// 「未指定」の場合は、usecase層で nil を渡さないようにする
	t.AssigneeID = assigneeID

	t.UpdatedAt = now
	return nil
}

func validateStatus(s TaskStatus) error {
	if _, err := ParseStatus(string(s)); err != nil {
		return errors.New("invalid task status")
	}
	return nil
}

func validatePriority(p TaskPriority) error {
	if _, err := ParsePriority(string(p)); err != nil {
		return errors.New("invalid task priority")
	}
	return nil
}

func (t *Task) TouchUpdatedAt() {
	t.UpdatedAt = time.Now()
}
