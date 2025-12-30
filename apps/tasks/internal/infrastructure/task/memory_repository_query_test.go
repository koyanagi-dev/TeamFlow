package taskinfra

import (
	"context"
	"fmt"
	"testing"
	"time"

	domain "teamflow-tasks/internal/domain/task"
)

func TestMemoryTaskRepository_FindByProjectID_StatusFilter(t *testing.T) {
	repo := NewMemoryTaskRepository()
	now := time.Now()

	// テストデータ作成
	t1, _ := domain.NewTask("task-1", "proj-1", "T1", "", domain.StatusTodo, domain.PriorityMedium, nil, now)
	t2, _ := domain.NewTask("task-2", "proj-1", "T2", "", domain.StatusInProgress, domain.PriorityMedium, nil, now)
	t3, _ := domain.NewTask("task-3", "proj-1", "T3", "", domain.StatusDone, domain.PriorityMedium, nil, now)

	repo.Save(context.Background(), t1)
	repo.Save(context.Background(), t2)
	repo.Save(context.Background(), t3)

	// status=todo でフィルタ
	query, _ := domain.NewTaskQuery(domain.WithStatusFilter("todo"))
	tasks, err := repo.FindByProjectID(context.Background(), "proj-1", query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	if tasks[0].ID != "task-1" {
		t.Errorf("expected task-1, got %s", tasks[0].ID)
	}
}

func TestMemoryTaskRepository_FindByProjectID_MultipleStatusFilter(t *testing.T) {
	repo := NewMemoryTaskRepository()
	now := time.Now()

	t1, _ := domain.NewTask("task-1", "proj-1", "T1", "", domain.StatusTodo, domain.PriorityMedium, nil, now)
	t2, _ := domain.NewTask("task-2", "proj-1", "T2", "", domain.StatusInProgress, domain.PriorityMedium, nil, now)
	t3, _ := domain.NewTask("task-3", "proj-1", "T3", "", domain.StatusDone, domain.PriorityMedium, nil, now)

	repo.Save(context.Background(), t1)
	repo.Save(context.Background(), t2)
	repo.Save(context.Background(), t3)

	// status=todo,in_progress でフィルタ
	query, _ := domain.NewTaskQuery(domain.WithStatusFilter("todo,in_progress"))
	tasks, err := repo.FindByProjectID(context.Background(), "proj-1", query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
}

func TestMemoryTaskRepository_FindByProjectID_StatusFilterWithDoing(t *testing.T) {
	repo := NewMemoryTaskRepository()
	now := time.Now()

	t1, _ := domain.NewTask("task-1", "proj-1", "T1", "", domain.StatusInProgress, domain.PriorityMedium, nil, now)
	repo.Save(context.Background(), t1)

	// doing は in_progress に正規化される
	query, _ := domain.NewTaskQuery(domain.WithStatusFilter("doing"))
	tasks, err := repo.FindByProjectID(context.Background(), "proj-1", query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
}

func TestMemoryTaskRepository_FindByProjectID_PriorityFilter(t *testing.T) {
	repo := NewMemoryTaskRepository()
	now := time.Now()

	t1, _ := domain.NewTask("task-1", "proj-1", "T1", "", domain.StatusTodo, domain.PriorityHigh, nil, now)
	t2, _ := domain.NewTask("task-2", "proj-1", "T2", "", domain.StatusTodo, domain.PriorityMedium, nil, now)
	t3, _ := domain.NewTask("task-3", "proj-1", "T3", "", domain.StatusTodo, domain.PriorityLow, nil, now)

	repo.Save(context.Background(), t1)
	repo.Save(context.Background(), t2)
	repo.Save(context.Background(), t3)

	// priority=high でフィルタ
	query, _ := domain.NewTaskQuery(domain.WithPriorityFilter("high"))
	tasks, err := repo.FindByProjectID(context.Background(), "proj-1", query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	if tasks[0].Priority != domain.PriorityHigh {
		t.Errorf("expected PriorityHigh, got %v", tasks[0].Priority)
	}
}

func TestMemoryTaskRepository_FindByProjectID_AssigneeIDFilter(t *testing.T) {
	repo := NewMemoryTaskRepository()
	now := time.Now()

	assignee1 := "user-1"
	assignee2 := "user-2"

	t1, _ := domain.NewTask("task-1", "proj-1", "T1", "", domain.StatusTodo, domain.PriorityMedium, nil, now)
	t1.AssigneeID = &assignee1
	t2, _ := domain.NewTask("task-2", "proj-1", "T2", "", domain.StatusTodo, domain.PriorityMedium, nil, now)
	t2.AssigneeID = &assignee2
	t3, _ := domain.NewTask("task-3", "proj-1", "T3", "", domain.StatusTodo, domain.PriorityMedium, nil, now)

	repo.Save(context.Background(), t1)
	repo.Save(context.Background(), t2)
	repo.Save(context.Background(), t3)

	// assigneeId=user-1 でフィルタ
	query, _ := domain.NewTaskQuery(domain.WithAssigneeIDFilter("user-1"))
	tasks, err := repo.FindByProjectID(context.Background(), "proj-1", query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	if tasks[0].ID != "task-1" {
		t.Errorf("expected task-1, got %s", tasks[0].ID)
	}
}

func TestMemoryTaskRepository_FindByProjectID_SortByPriority(t *testing.T) {
	repo := NewMemoryTaskRepository()
	now := time.Now()

	t1, _ := domain.NewTask("task-1", "proj-1", "T1", "", domain.StatusTodo, domain.PriorityLow, nil, now)
	t2, _ := domain.NewTask("task-2", "proj-1", "T2", "", domain.StatusTodo, domain.PriorityHigh, nil, now)
	t3, _ := domain.NewTask("task-3", "proj-1", "T3", "", domain.StatusTodo, domain.PriorityMedium, nil, now)

	repo.Save(context.Background(), t1)
	repo.Save(context.Background(), t2)
	repo.Save(context.Background(), t3)

	// priority DESC でソート（high > medium > low）
	query, _ := domain.NewTaskQuery(domain.WithSort("-priority"))
	tasks, err := repo.FindByProjectID(context.Background(), "proj-1", query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}

	// high > medium > low の順序を確認
	if tasks[0].Priority != domain.PriorityHigh {
		t.Errorf("expected PriorityHigh at index 0, got %v", tasks[0].Priority)
	}
	if tasks[1].Priority != domain.PriorityMedium {
		t.Errorf("expected PriorityMedium at index 1, got %v", tasks[1].Priority)
	}
	if tasks[2].Priority != domain.PriorityLow {
		t.Errorf("expected PriorityLow at index 2, got %v", tasks[2].Priority)
	}
}

func TestMemoryTaskRepository_FindByProjectID_SortByCreatedAt(t *testing.T) {
	repo := NewMemoryTaskRepository()
	baseTime := time.Now()

	t1, _ := domain.NewTask("task-1", "proj-1", "T1", "", domain.StatusTodo, domain.PriorityMedium, nil, baseTime.Add(-2*time.Hour))
	t2, _ := domain.NewTask("task-2", "proj-1", "T2", "", domain.StatusTodo, domain.PriorityMedium, nil, baseTime.Add(-1*time.Hour))
	t3, _ := domain.NewTask("task-3", "proj-1", "T3", "", domain.StatusTodo, domain.PriorityMedium, nil, baseTime)

	repo.Save(context.Background(), t1)
	repo.Save(context.Background(), t2)
	repo.Save(context.Background(), t3)

	// createdAt ASC でソート
	query, _ := domain.NewTaskQuery(domain.WithSort("createdAt"))
	tasks, err := repo.FindByProjectID(context.Background(), "proj-1", query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}

	// 古い順（t1, t2, t3）
	if tasks[0].ID != "task-1" {
		t.Errorf("expected task-1 at index 0, got %s", tasks[0].ID)
	}
	if tasks[1].ID != "task-2" {
		t.Errorf("expected task-2 at index 1, got %s", tasks[1].ID)
	}
	if tasks[2].ID != "task-3" {
		t.Errorf("expected task-3 at index 2, got %s", tasks[2].ID)
	}
}

func TestMemoryTaskRepository_FindByProjectID_Limit(t *testing.T) {
	repo := NewMemoryTaskRepository()
	now := time.Now()

	// 10個のタスクを作成
	for i := 1; i <= 10; i++ {
		id := fmt.Sprintf("task-%d", i)
		title := fmt.Sprintf("T%d", i)
		task, _ := domain.NewTask(
			id,
			"proj-1",
			title,
			"",
			domain.StatusTodo,
			domain.PriorityMedium,
			nil,
			now,
		)
		repo.Save(context.Background(), task)
	}

	// limit=5 でリミット
	query, _ := domain.NewTaskQuery(domain.WithLimit(5))
	tasks, err := repo.FindByProjectID(context.Background(), "proj-1", query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tasks) != 5 {
		t.Fatalf("expected 5 tasks, got %d", len(tasks))
	}
}

func TestMemoryTaskRepository_FindByProjectID_QueryFilter(t *testing.T) {
	repo := NewMemoryTaskRepository()
	now := time.Now()

	t1, _ := domain.NewTask("task-1", "proj-1", "Task Alpha", "", domain.StatusTodo, domain.PriorityMedium, nil, now)
	t2, _ := domain.NewTask("task-2", "proj-1", "Task Beta", "", domain.StatusTodo, domain.PriorityMedium, nil, now)
	t3, _ := domain.NewTask("task-3", "proj-1", "Alpha Task", "", domain.StatusTodo, domain.PriorityMedium, nil, now)

	repo.Save(context.Background(), t1)
	repo.Save(context.Background(), t2)
	repo.Save(context.Background(), t3)

	// q=Alpha でタイトル検索（部分一致、大文字小文字無視）
	query, _ := domain.NewTaskQuery(domain.WithQueryFilter("Alpha"))
	tasks, err := repo.FindByProjectID(context.Background(), "proj-1", query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}

	// "Task Alpha" と "Alpha Task" が含まれることを確認
	found := make(map[string]bool)
	for _, task := range tasks {
		found[task.ID] = true
	}
	if !found["task-1"] || !found["task-3"] {
		t.Errorf("expected task-1 and task-3, got %v", found)
	}
}

func TestMemoryTaskRepository_FindByProjectID_MultipleFilters(t *testing.T) {
	repo := NewMemoryTaskRepository()
	now := time.Now()

	assignee1 := "user-1"

	t1, _ := domain.NewTask("task-1", "proj-1", "T1", "", domain.StatusTodo, domain.PriorityHigh, nil, now)
	t1.AssigneeID = &assignee1
	t2, _ := domain.NewTask("task-2", "proj-1", "T2", "", domain.StatusTodo, domain.PriorityLow, nil, now)
	t2.AssigneeID = &assignee1
	t3, _ := domain.NewTask("task-3", "proj-1", "T3", "", domain.StatusInProgress, domain.PriorityHigh, nil, now)
	t3.AssigneeID = &assignee1

	repo.Save(context.Background(), t1)
	repo.Save(context.Background(), t2)
	repo.Save(context.Background(), t3)

	// status=todo AND priority=high AND assigneeId=user-1 でフィルタ
	query, _ := domain.NewTaskQuery(
		domain.WithStatusFilter("todo"),
		domain.WithPriorityFilter("high"),
		domain.WithAssigneeIDFilter("user-1"),
	)
	tasks, err := repo.FindByProjectID(context.Background(), "proj-1", query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	if tasks[0].ID != "task-1" {
		t.Errorf("expected task-1, got %s", tasks[0].ID)
	}
}

