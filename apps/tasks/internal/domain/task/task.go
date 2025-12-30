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

// CompareTo は優先度を比較する（high > medium > low）。
// 戻り値: <0 (p < other), 0 (p == other), >0 (p > other)
func (p TaskPriority) CompareTo(other TaskPriority) int {
	value := func(pr TaskPriority) int {
		switch pr {
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
	return value(p) - value(other)
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
