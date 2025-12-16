package task

import (
	"testing"
	"time"
)

func TestNewTask_Success(t *testing.T) {
	now := time.Now()

	task, err := NewTask(
		"task-1",
		"proj-1",
		"画面設計",
		"プロジェクト一覧画面のUIを設計する",
		StatusTodo,
		PriorityMedium,
		nil,
		now,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if task.ID != "task-1" {
		t.Errorf("expected ID=task-1, got=%s", task.ID)
	}

	if task.ProjectID != "proj-1" {
		t.Errorf("expected ProjectID=proj-1, got=%s", task.ProjectID)
	}

	if task.Title != "画面設計" {
		t.Errorf("expected Title=画面設計, got=%s", task.Title)
	}

	if task.Status != StatusTodo {
		t.Errorf("expected Status=StatusTodo, got=%s", task.Status)
	}

	if task.Priority != PriorityMedium {
		t.Errorf("expected Priority=PriorityMedium, got=%s", task.Priority)
	}

	if !task.CreatedAt.Equal(now) || !task.UpdatedAt.Equal(now) {
		t.Errorf("expected CreatedAt/UpdatedAt to equal now, got=%v/%v", task.CreatedAt, task.UpdatedAt)
	}
}

func TestNewTask_EmptyTitle(t *testing.T) {
	now := time.Now()

	_, err := NewTask(
		"task-1",
		"proj-1",
		"",
		"説明",
		StatusTodo,
		PriorityMedium,
		nil,
		now,
	)
	if err == nil {
		t.Fatalf("expected error for empty title, got nil")
	}
}

func TestNewTask_InvalidStatus(t *testing.T) {
	now := time.Now()

	_, err := NewTask(
		"task-1",
		"proj-1",
		"タイトル",
		"説明",
		"invalid-status",
		PriorityMedium,
		nil,
		now,
	)
	if err == nil {
		t.Fatalf("expected error for invalid status, got nil")
	}
}

func TestParseStatus(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		for _, s := range []string{"todo", "in_progress", "done"} {
			got, err := ParseStatus(s)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", s, err)
			}
			if string(got) != s {
				t.Fatalf("expected %s, got %s", s, got)
			}
		}
	})

	t.Run("invalid", func(t *testing.T) {
		if _, err := ParseStatus("doing"); err == nil {
			t.Fatalf("expected error for invalid status")
		}
	})
}

func TestParsePriority(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		for _, p := range []string{"low", "medium", "high"} {
			got, err := ParsePriority(p)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", p, err)
			}
			if string(got) != p {
				t.Fatalf("expected %s, got %s", p, got)
			}
		}
	})

	t.Run("invalid", func(t *testing.T) {
		if _, err := ParsePriority("urgent"); err == nil {
			t.Fatalf("expected error for invalid priority")
		}
	})
}
