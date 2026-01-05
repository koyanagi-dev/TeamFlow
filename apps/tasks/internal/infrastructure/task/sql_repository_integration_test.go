//go:build integration
// +build integration

package taskinfra

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	domain "teamflow-tasks/internal/domain/task"
)

// testPool is initialized in integration_test.go (TestMain).
// We keep it in this package scope so integration tests can share a single DB pool.
var testPool *pgxpool.Pool

// setupTestDB returns the integration-test pool.
// It fails fast if TestMain didn't initialize the pool.
func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testPool == nil {
		t.Fatalf("testPool is nil: ensure TestMain initialized it (go test -tags=integration ./... with DB_TEST_DSN)")
	}
	return testPool
}

func resetTasksTable(t *testing.T, db *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	_, err := db.Exec(ctx, "TRUNCATE TABLE tasks")
	if err != nil {
		t.Fatalf("failed to truncate tasks: %v", err)
	}
}

type seedTask struct {
	ID         string
	ProjectID  string
	Title      string
	Desc       *string
	Status     string
	Priority   string
	AssigneeID *string
	DueDate    *time.Time // DATE in DB: pass time at midnight; nil for NULL
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func insertTasks(t *testing.T, db *pgxpool.Pool, tasks []seedTask) {
	t.Helper()
	ctx := context.Background()

	const q = `
		INSERT INTO tasks (
			id, project_id, title, description, status, priority, assignee_id, due_date, created_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10
		)
	`
	for _, tt := range tasks {
		_, err := db.Exec(ctx, q,
			tt.ID, tt.ProjectID, tt.Title, tt.Desc, tt.Status, tt.Priority, tt.AssigneeID, tt.DueDate, tt.CreatedAt, tt.UpdatedAt,
		)
		if err != nil {
			t.Fatalf("failed to insert seed task id=%s: %v", tt.ID, err)
		}
	}
}

// helper to build a DATE (no time precision required; we care about NULL ordering / filtering)
func dateYMD(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// TestSQLTaskRepository_FindByProjectID_SortByPriority はpriorityソートを検証する。
func TestSQLTaskRepository_FindByProjectID_SortByPriority(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLTaskRepository(db)
	resetTasksTable(t, db)

	now := time.Now().UTC()

	insertTasks(t, db, []seedTask{
		{ID: "task-high", ProjectID: "proj-1", Title: "T1", Status: "todo", Priority: "high", CreatedAt: now.Add(-3 * time.Hour), UpdatedAt: now.Add(-3 * time.Hour)},
		{ID: "task-medium", ProjectID: "proj-1", Title: "T2", Status: "todo", Priority: "medium", CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)},
		{ID: "task-low", ProjectID: "proj-1", Title: "T3", Status: "todo", Priority: "low", CreatedAt: now.Add(-1 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)},
	})

	// priority DESC でソート（high > medium > low）
	query, err := domain.NewTaskQuery(domain.WithSort("-priority"))
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}

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

// TestSQLTaskRepository_FindByProjectID_SortByDueDate_NullHandling はdueDateのnull順を検証する。
func TestSQLTaskRepository_FindByProjectID_SortByDueDate_NullHandling(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLTaskRepository(db)
	resetTasksTable(t, db)

	now := time.Now().UTC()
	d1 := dateYMD(2026, 1, 10)
	d2 := dateYMD(2026, 1, 20)

	insertTasks(t, db, []seedTask{
		{ID: "task-null", ProjectID: "proj-1", Title: "NULL due", Status: "todo", Priority: "medium", DueDate: nil, CreatedAt: now.Add(-3 * time.Hour), UpdatedAt: now.Add(-3 * time.Hour)},
		{ID: "task-d1", ProjectID: "proj-1", Title: "due 1", Status: "todo", Priority: "medium", DueDate: &d1, CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)},
		{ID: "task-d2", ProjectID: "proj-1", Title: "due 2", Status: "todo", Priority: "medium", DueDate: &d2, CreatedAt: now.Add(-1 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)},
	})

	// dueDate ASC は NULLS LAST が期待（ASC のとき null は last）
	qAsc, err := domain.NewTaskQuery(domain.WithSort("dueDate"))
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}

	ascTasks, err := repo.FindByProjectID(context.Background(), "proj-1", qAsc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ascTasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(ascTasks))
	}
	if ascTasks[2].DueDate != nil {
		t.Errorf("expected NULL dueDate at last index for ASC, got %v", ascTasks[2].DueDate)
	}

	// dueDate DESC は NULLS FIRST が期待（DESC のとき null は first）
	qDesc, err := domain.NewTaskQuery(domain.WithSort("-dueDate"))
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}

	descTasks, err := repo.FindByProjectID(context.Background(), "proj-1", qDesc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(descTasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(descTasks))
	}
	if descTasks[0].DueDate != nil {
		t.Errorf("expected NULL dueDate at first index for DESC, got %v", descTasks[0].DueDate)
	}
}

// TestSQLTaskRepository_FindByProjectID_SortByCreatedAt はcreatedAtソートを検証する。
func TestSQLTaskRepository_FindByProjectID_SortByCreatedAt(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLTaskRepository(db)
	resetTasksTable(t, db)

	base := time.Now().UTC().Add(-10 * time.Minute)

	insertTasks(t, db, []seedTask{
		{ID: "task-1", ProjectID: "proj-1", Title: "old", Status: "todo", Priority: "low", CreatedAt: base.Add(-2 * time.Minute), UpdatedAt: base.Add(-2 * time.Minute)},
		{ID: "task-2", ProjectID: "proj-1", Title: "mid", Status: "todo", Priority: "low", CreatedAt: base.Add(-1 * time.Minute), UpdatedAt: base.Add(-1 * time.Minute)},
		{ID: "task-3", ProjectID: "proj-1", Title: "new", Status: "todo", Priority: "low", CreatedAt: base, UpdatedAt: base},
	})

	qDesc, err := domain.NewTaskQuery(domain.WithSort("-createdAt"))
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}

	tasks, err := repo.FindByProjectID(context.Background(), "proj-1", qDesc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}
	if tasks[0].ID != "task-3" || tasks[2].ID != "task-1" {
		t.Errorf("unexpected order: got [%s,%s,%s]", tasks[0].ID, tasks[1].ID, tasks[2].ID)
	}
}

// TestSQLTaskRepository_FindByProjectID_MultipleSortKeys は複数ソートキーを検証する。
func TestSQLTaskRepository_FindByProjectID_MultipleSortKeys(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLTaskRepository(db)
	resetTasksTable(t, db)

	base := time.Now().UTC().Add(-1 * time.Hour)

	// 同 priority の中で createdAt で並ぶことを確認する（-priority,createdAt）
	insertTasks(t, db, []seedTask{
		{ID: "task-a", ProjectID: "proj-1", Title: "A", Status: "todo", Priority: "high", CreatedAt: base.Add(10 * time.Minute), UpdatedAt: base.Add(10 * time.Minute)},
		{ID: "task-b", ProjectID: "proj-1", Title: "B", Status: "todo", Priority: "high", CreatedAt: base.Add(0 * time.Minute), UpdatedAt: base.Add(0 * time.Minute)},
		{ID: "task-c", ProjectID: "proj-1", Title: "C", Status: "todo", Priority: "medium", CreatedAt: base.Add(5 * time.Minute), UpdatedAt: base.Add(5 * time.Minute)},
	})

	q, err := domain.NewTaskQuery(domain.WithSort("-priority,createdAt"))
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}

	tasks, err := repo.FindByProjectID(context.Background(), "proj-1", q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}

	// 1) priority high が先、2) high の中では createdAt ASC（古い→新しい）
	if tasks[0].ID != "task-b" || tasks[1].ID != "task-a" {
		t.Errorf("unexpected order for high-priority subgroup: got [%s,%s]", tasks[0].ID, tasks[1].ID)
	}
	if tasks[2].ID != "task-c" {
		t.Errorf("expected task-c at index 2, got %s", tasks[2].ID)
	}
}
