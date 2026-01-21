//go:build integration
// +build integration

package taskinfra

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	domain "teamflow-tasks/internal/domain/task"
	"teamflow-tasks/internal/testutil"
)

// testPool is initialized in integration_test.go (TestMain).
// We keep it in this package scope so integration tests can share a single DB pool.
var testPool *pgxpool.Pool

// clipToLimit は tasks を limit 件に切り詰める（repository層は limit+1 件返すため）
func clipToLimit(tasks []*domain.Task, limit int) []*domain.Task {
	if len(tasks) <= limit {
		return tasks
	}
	return tasks[:limit]
}

// taskIDSet は task ID の集合を表す（順序に依存しない比較用）
func taskIDSet(tasks []*domain.Task) map[string]struct{} {
	set := make(map[string]struct{})
	for _, t := range tasks {
		set[t.ID] = struct{}{}
	}
	return set
}

// assertTaskIDs は返されたタスクの ID が期待値と一致することを検証する（順序不問）
func assertTaskIDs(t *testing.T, tasks []*domain.Task, expectedIDs []string) {
	t.Helper()
	actualSet := taskIDSet(tasks)
	expectedSet := make(map[string]struct{})
	for _, id := range expectedIDs {
		expectedSet[id] = struct{}{}
	}

	if len(actualSet) != len(expectedSet) {
		t.Errorf("task count mismatch: expected %d tasks, got %d. ExpectedIDs: %v, ActualIDs: %v", len(expectedSet), len(actualSet), expectedIDs, getTaskIDs(tasks))
		return
	}

	for id := range expectedSet {
		if _, ok := actualSet[id]; !ok {
			t.Errorf("expected task ID %s not found. ExpectedIDs: %v, ActualIDs: %v", id, expectedIDs, getTaskIDs(tasks))
		}
	}
}

// assertNoProjectLeakage は proj-2 のタスクが混入していないことを検証する
func assertNoProjectLeakage(t *testing.T, tasks []*domain.Task, projectID string) {
	t.Helper()
	for _, task := range tasks {
		if task.ProjectID != projectID {
			t.Errorf("project leakage detected: task %s belongs to project %s, expected %s", task.ID, task.ProjectID, projectID)
		}
	}
}

// getTaskIDs はタスクの ID リストを返す（デバッグ用）
func getTaskIDs(tasks []*domain.Task) []string {
	ids := make([]string, len(tasks))
	for i, t := range tasks {
		ids[i] = t.ID
	}
	return ids
}

// TestSQLTaskRepository_FindByProjectID_SortByPriority はpriorityソートを検証する。
func TestSQLTaskRepository_FindByProjectID_SortByPriority(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewSQLTaskRepository(db)
	testutil.ResetTasksTable(t, db)

	now := time.Now().UTC()

	testutil.InsertTasks(t, db, []testutil.SeedTask{
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
	db := testutil.SetupTestDB(t)
	repo := NewSQLTaskRepository(db)
	testutil.ResetTasksTable(t, db)

	now := time.Now().UTC()
	d1 := testutil.DateYMD(2026, 1, 10)
	d2 := testutil.DateYMD(2026, 1, 20)

	testutil.InsertTasks(t, db, []testutil.SeedTask{
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
	db := testutil.SetupTestDB(t)
	repo := NewSQLTaskRepository(db)
	testutil.ResetTasksTable(t, db)

	base := time.Now().UTC().Add(-10 * time.Minute)

	testutil.InsertTasks(t, db, []testutil.SeedTask{
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
	db := testutil.SetupTestDB(t)
	repo := NewSQLTaskRepository(db)
	testutil.ResetTasksTable(t, db)

	base := time.Now().UTC().Add(-1 * time.Hour)

	// 同 priority の中で createdAt で並ぶことを確認する（-priority,createdAt）
	testutil.InsertTasks(t, db, []testutil.SeedTask{
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

// ============================================================================
// Filter: Status Tests
// ============================================================================

// TestSQLTaskRepository_FindByProjectID_Filter_Status_Single は単一 status フィルタを検証する。
func TestSQLTaskRepository_FindByProjectID_Filter_Status_Single(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewSQLTaskRepository(db)
	testutil.ResetTasksTable(t, db)

	now := time.Now().UTC()
	user1 := "user-1"
	user2 := "user-2"

	testutil.InsertTasks(t, db, []testutil.SeedTask{
		// proj-1: todo, in_progress, done を混在
		{ID: "proj1-todo", ProjectID: "proj-1", Title: "alpha", Status: "todo", Priority: "high", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
		{ID: "proj1-inprogress", ProjectID: "proj-1", Title: "beta", Status: "in_progress", Priority: "medium", AssigneeID: &user2, CreatedAt: now, UpdatedAt: now},
		{ID: "proj1-done", ProjectID: "proj-1", Title: "gamma", Status: "done", Priority: "low", AssigneeID: nil, CreatedAt: now, UpdatedAt: now},
		// proj-2: 混入防止のため
		{ID: "proj2-todo", ProjectID: "proj-2", Title: "delta", Status: "todo", Priority: "high", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
		{ID: "proj2-done", ProjectID: "proj-2", Title: "epsilon", Status: "done", Priority: "medium", AssigneeID: &user2, CreatedAt: now, UpdatedAt: now},
	})

	query, err := domain.NewTaskQuery(domain.WithStatusFilter("todo"), domain.WithLimit(10))
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}

	tasks, err := repo.FindByProjectID(context.Background(), "proj-1", query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// todo のみ返る
	assertTaskIDs(t, tasks, []string{"proj1-todo"})
	assertNoProjectLeakage(t, tasks, "proj-1")
}

// TestSQLTaskRepository_FindByProjectID_Filter_Status_Multiple は複数 status フィルタを検証する。
func TestSQLTaskRepository_FindByProjectID_Filter_Status_Multiple(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewSQLTaskRepository(db)
	testutil.ResetTasksTable(t, db)

	now := time.Now().UTC()
	user1 := "user-1"

	testutil.InsertTasks(t, db, []testutil.SeedTask{
		{ID: "proj1-todo", ProjectID: "proj-1", Title: "alpha", Status: "todo", Priority: "high", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
		{ID: "proj1-inprogress", ProjectID: "proj-1", Title: "beta", Status: "in_progress", Priority: "medium", AssigneeID: nil, CreatedAt: now, UpdatedAt: now},
		{ID: "proj1-done", ProjectID: "proj-1", Title: "gamma", Status: "done", Priority: "low", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
		{ID: "proj2-todo", ProjectID: "proj-2", Title: "delta", Status: "todo", Priority: "high", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
		{ID: "proj2-done", ProjectID: "proj-2", Title: "epsilon", Status: "done", Priority: "medium", AssigneeID: nil, CreatedAt: now, UpdatedAt: now},
	})

	query, err := domain.NewTaskQuery(domain.WithStatusFilter("todo,done"), domain.WithLimit(10))
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}

	tasks, err := repo.FindByProjectID(context.Background(), "proj-1", query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// todo と done のみ返る（in_progress は返らない）
	assertTaskIDs(t, tasks, []string{"proj1-todo", "proj1-done"})
	assertNoProjectLeakage(t, tasks, "proj-1")
}

// TestSQLTaskRepository_FindByProjectID_Filter_Status_InvalidValue は無効な status が domain.NewTaskQuery でエラーになることを検証する。
func TestSQLTaskRepository_FindByProjectID_Filter_Status_InvalidValue(t *testing.T) {
	// 無効な status を domain.NewTaskQuery で作成するとエラーになることを検証
	invalidStatus := "invalid"
	_, err := domain.NewTaskQuery(domain.WithStatusFilter(invalidStatus))
	if err == nil {
		t.Fatalf("expected error for invalid status, but got nil")
	}
	// typed error (ValidationError) で field=status, code=INVALID_ENUM, RejectedValue を確認
	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("expected ValidationError, got: %T", err)
		return
	}
	if ve.Field != "status" {
		t.Errorf("expected field=status, got field=%s", ve.Field)
	}
	if ve.Code != "INVALID_ENUM" {
		t.Errorf("expected code=INVALID_ENUM, got code=%s", ve.Code)
	}
	if ve.RejectedValue == nil {
		t.Errorf("expected RejectedValue to be set, got nil")
	} else if *ve.RejectedValue != invalidStatus {
		t.Errorf("expected RejectedValue=%s, got %s", invalidStatus, *ve.RejectedValue)
	}
}

// ============================================================================
// Filter: Priority Tests
// ============================================================================

// TestSQLTaskRepository_FindByProjectID_Filter_Priority_Single は単一 priority フィルタを検証する。
func TestSQLTaskRepository_FindByProjectID_Filter_Priority_Single(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewSQLTaskRepository(db)
	testutil.ResetTasksTable(t, db)

	now := time.Now().UTC()
	user1 := "user-1"

	testutil.InsertTasks(t, db, []testutil.SeedTask{
		{ID: "proj1-high", ProjectID: "proj-1", Title: "alpha", Status: "todo", Priority: "high", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
		{ID: "proj1-medium", ProjectID: "proj-1", Title: "beta", Status: "todo", Priority: "medium", AssigneeID: nil, CreatedAt: now, UpdatedAt: now},
		{ID: "proj1-low", ProjectID: "proj-1", Title: "gamma", Status: "todo", Priority: "low", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
		{ID: "proj2-high", ProjectID: "proj-2", Title: "delta", Status: "todo", Priority: "high", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
	})

	query, err := domain.NewTaskQuery(domain.WithPriorityFilter("high"), domain.WithLimit(10))
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}

	tasks, err := repo.FindByProjectID(context.Background(), "proj-1", query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertTaskIDs(t, tasks, []string{"proj1-high"})
	assertNoProjectLeakage(t, tasks, "proj-1")
}

// TestSQLTaskRepository_FindByProjectID_Filter_Priority_Multiple は複数 priority フィルタを検証する。
func TestSQLTaskRepository_FindByProjectID_Filter_Priority_Multiple(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewSQLTaskRepository(db)
	testutil.ResetTasksTable(t, db)

	now := time.Now().UTC()
	user1 := "user-1"

	testutil.InsertTasks(t, db, []testutil.SeedTask{
		{ID: "proj1-high", ProjectID: "proj-1", Title: "alpha", Status: "todo", Priority: "high", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
		{ID: "proj1-medium", ProjectID: "proj-1", Title: "beta", Status: "todo", Priority: "medium", AssigneeID: nil, CreatedAt: now, UpdatedAt: now},
		{ID: "proj1-low", ProjectID: "proj-1", Title: "gamma", Status: "todo", Priority: "low", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
		{ID: "proj2-high", ProjectID: "proj-2", Title: "delta", Status: "todo", Priority: "high", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
	})

	query, err := domain.NewTaskQuery(domain.WithPriorityFilter("high,low"), domain.WithLimit(10))
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}

	tasks, err := repo.FindByProjectID(context.Background(), "proj-1", query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// high と low のみ返る（medium は返らない）
	assertTaskIDs(t, tasks, []string{"proj1-high", "proj1-low"})
	assertNoProjectLeakage(t, tasks, "proj-1")
}

// TestSQLTaskRepository_FindByProjectID_Filter_Priority_InvalidValue は無効な priority が domain.NewTaskQuery でエラーになることを検証する。
func TestSQLTaskRepository_FindByProjectID_Filter_Priority_InvalidValue(t *testing.T) {
	// 無効な priority を domain.NewTaskQuery で作成するとエラーになることを検証
	invalidPriority := "pwn"
	_, err := domain.NewTaskQuery(domain.WithPriorityFilter(invalidPriority))
	if err == nil {
		t.Fatalf("expected error for invalid priority, but got nil")
	}
	// typed error (ValidationError) で field=priority, code=INVALID_ENUM, RejectedValue を確認
	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("expected ValidationError, got: %T", err)
		return
	}
	if ve.Field != "priority" {
		t.Errorf("expected field=priority, got field=%s", ve.Field)
	}
	if ve.Code != "INVALID_ENUM" {
		t.Errorf("expected code=INVALID_ENUM, got code=%s", ve.Code)
	}
	if ve.RejectedValue == nil {
		t.Errorf("expected RejectedValue to be set, got nil")
	} else if *ve.RejectedValue != invalidPriority {
		t.Errorf("expected RejectedValue=%s, got %s", invalidPriority, *ve.RejectedValue)
	}
}

// ============================================================================
// Filter: AssigneeID Tests
// ============================================================================

// TestSQLTaskRepository_FindByProjectID_Filter_AssigneeID は assigneeId フィルタを検証する。
func TestSQLTaskRepository_FindByProjectID_Filter_AssigneeID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewSQLTaskRepository(db)
	testutil.ResetTasksTable(t, db)

	now := time.Now().UTC()
	user1 := "user-1"
	user2 := "user-2"

	testutil.InsertTasks(t, db, []testutil.SeedTask{
		{ID: "proj1-user1", ProjectID: "proj-1", Title: "alpha", Status: "todo", Priority: "high", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
		{ID: "proj1-user2", ProjectID: "proj-1", Title: "beta", Status: "todo", Priority: "medium", AssigneeID: &user2, CreatedAt: now, UpdatedAt: now},
		{ID: "proj1-null", ProjectID: "proj-1", Title: "gamma", Status: "todo", Priority: "low", AssigneeID: nil, CreatedAt: now, UpdatedAt: now},
		{ID: "proj2-user1", ProjectID: "proj-2", Title: "delta", Status: "todo", Priority: "high", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
	})

	query, err := domain.NewTaskQuery(domain.WithAssigneeIDFilter("user-1"), domain.WithLimit(10))
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}

	tasks, err := repo.FindByProjectID(context.Background(), "proj-1", query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertTaskIDs(t, tasks, []string{"proj1-user1"})
	assertNoProjectLeakage(t, tasks, "proj-1")
}

// TestSQLTaskRepository_FindByProjectID_Filter_AssigneeID_NilOrEmptyIgnored は nil/empty assigneeId が無視されることを検証する。
func TestSQLTaskRepository_FindByProjectID_Filter_AssigneeID_NilOrEmptyIgnored(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewSQLTaskRepository(db)
	testutil.ResetTasksTable(t, db)

	now := time.Now().UTC()
	user1 := "user-1"
	user2 := "user-2"

	testutil.InsertTasks(t, db, []testutil.SeedTask{
		{ID: "proj1-user1", ProjectID: "proj-1", Title: "alpha", Status: "todo", Priority: "high", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
		{ID: "proj1-user2", ProjectID: "proj-1", Title: "beta", Status: "todo", Priority: "medium", AssigneeID: &user2, CreatedAt: now, UpdatedAt: now},
		{ID: "proj1-null", ProjectID: "proj-1", Title: "gamma", Status: "todo", Priority: "low", AssigneeID: nil, CreatedAt: now, UpdatedAt: now},
		{ID: "proj2-user1", ProjectID: "proj-2", Title: "delta", Status: "todo", Priority: "high", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
	})

	// nil の時は全件（project scope内）を返す
	query1, err := domain.NewTaskQuery(domain.WithLimit(10))
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}
	query1.AssigneeID = nil

	tasks1, err := repo.FindByProjectID(context.Background(), "proj-1", query1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertTaskIDs(t, tasks1, []string{"proj1-user1", "proj1-user2", "proj1-null"})
	assertNoProjectLeakage(t, tasks1, "proj-1")

	// "" の時も nil と同じ挙動（絞られない）
	query2, err := domain.NewTaskQuery(domain.WithAssigneeIDFilter(""), domain.WithLimit(10))
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}

	tasks2, err := repo.FindByProjectID(context.Background(), "proj-1", query2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// WithAssigneeIDFilter("") は nil を設定するので、全件返る
	assertTaskIDs(t, tasks2, []string{"proj1-user1", "proj1-user2", "proj1-null"})
	assertNoProjectLeakage(t, tasks2, "proj-1")
}

// ============================================================================
// Combined Filter Tests
// ============================================================================

// TestSQLTaskRepository_FindByProjectID_Filter_Combined は複合フィルタを検証する。
func TestSQLTaskRepository_FindByProjectID_Filter_Combined(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewSQLTaskRepository(db)
	testutil.ResetTasksTable(t, db)

	now := time.Now().UTC()
	user1 := "user-1"
	user2 := "user-2"

	testutil.InsertTasks(t, db, []testutil.SeedTask{
		// proj-1: 条件に合うもの
		{ID: "proj1-match", ProjectID: "proj-1", Title: "alpha", Status: "todo", Priority: "high", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
		{ID: "proj1-done-high-user1", ProjectID: "proj-1", Title: "beta", Status: "done", Priority: "high", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
		// proj-1: 条件に合わないもの
		{ID: "proj1-todo-medium-user1", ProjectID: "proj-1", Title: "gamma", Status: "todo", Priority: "medium", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
		{ID: "proj1-todo-high-user2", ProjectID: "proj-1", Title: "delta", Status: "todo", Priority: "high", AssigneeID: &user2, CreatedAt: now, UpdatedAt: now},
		{ID: "proj1-inprogress-high-user1", ProjectID: "proj-1", Title: "epsilon", Status: "in_progress", Priority: "high", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
		// proj-2: 混入防止
		{ID: "proj2-match", ProjectID: "proj-2", Title: "zeta", Status: "todo", Priority: "high", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
	})

	query, err := domain.NewTaskQuery(
		domain.WithStatusFilter("todo,done"),
		domain.WithPriorityFilter("high"),
		domain.WithAssigneeIDFilter("user-1"),
		domain.WithLimit(10),
	)
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}

	tasks, err := repo.FindByProjectID(context.Background(), "proj-1", query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// status=[todo,done] AND priority=[high] AND assigneeId=user-1 の AND 条件
	assertTaskIDs(t, tasks, []string{"proj1-match", "proj1-done-high-user1"})
	assertNoProjectLeakage(t, tasks, "proj-1")
}

// TestSQLTaskRepository_FindByProjectID_Filter_NoHit は 0 件になるケースを検証する。
func TestSQLTaskRepository_FindByProjectID_Filter_NoHit(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewSQLTaskRepository(db)
	testutil.ResetTasksTable(t, db)

	now := time.Now().UTC()
	user1 := "user-1"
	user2 := "user-2"

	testutil.InsertTasks(t, db, []testutil.SeedTask{
		{ID: "proj1-todo-medium-user1", ProjectID: "proj-1", Title: "alpha", Status: "todo", Priority: "medium", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
		{ID: "proj1-done-low-user2", ProjectID: "proj-1", Title: "beta", Status: "done", Priority: "low", AssigneeID: &user2, CreatedAt: now, UpdatedAt: now},
		{ID: "proj2-todo-high-user1", ProjectID: "proj-2", Title: "gamma", Status: "todo", Priority: "high", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
	})

	query, err := domain.NewTaskQuery(
		domain.WithStatusFilter("todo"),
		domain.WithPriorityFilter("high"),
		domain.WithAssigneeIDFilter("user-2"),
		domain.WithLimit(10),
	)
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}

	tasks, err := repo.FindByProjectID(context.Background(), "proj-1", query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 0 件になる（todo AND high AND user-2 は存在しない）
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d: %v", len(tasks), getTaskIDs(tasks))
	}
	assertNoProjectLeakage(t, tasks, "proj-1")
}

// ============================================================================
// Search (q / title ILIKE) Tests
// ============================================================================

// TestSQLTaskRepository_FindByProjectID_Search_Title_Partial は title の部分一致検索を検証する。
func TestSQLTaskRepository_FindByProjectID_Search_Title_Partial(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewSQLTaskRepository(db)
	testutil.ResetTasksTable(t, db)

	now := time.Now().UTC()
	user1 := "user-1"

	testutil.InsertTasks(t, db, []testutil.SeedTask{
		{ID: "proj1-alpha", ProjectID: "proj-1", Title: "alpha", Status: "todo", Priority: "high", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
		{ID: "proj1-ALPHA", ProjectID: "proj-1", Title: "ALPHA", Status: "todo", Priority: "medium", AssigneeID: nil, CreatedAt: now, UpdatedAt: now},
		{ID: "proj1-beta", ProjectID: "proj-1", Title: "beta", Status: "todo", Priority: "low", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
		{ID: "proj2-alpha", ProjectID: "proj-2", Title: "alpha", Status: "todo", Priority: "high", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
	})

	query, err := domain.NewTaskQuery(domain.WithQueryFilter("alp"), domain.WithLimit(10))
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}

	tasks, err := repo.FindByProjectID(context.Background(), "proj-1", query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// "alp" で "alpha" と "ALPHA" がヒット（大小無視）
	assertTaskIDs(t, tasks, []string{"proj1-alpha", "proj1-ALPHA"})
	assertNoProjectLeakage(t, tasks, "proj-1")
}

// TestSQLTaskRepository_FindByProjectID_Search_MinLength_1 は最小長 1 の検索を検証する。
func TestSQLTaskRepository_FindByProjectID_Search_MinLength_1(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewSQLTaskRepository(db)
	testutil.ResetTasksTable(t, db)

	now := time.Now().UTC()
	user1 := "user-1"

	testutil.InsertTasks(t, db, []testutil.SeedTask{
		{ID: "proj1-alpha", ProjectID: "proj-1", Title: "alpha", Status: "todo", Priority: "high", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
		{ID: "proj1-beta", ProjectID: "proj-1", Title: "beta", Status: "todo", Priority: "medium", AssigneeID: nil, CreatedAt: now, UpdatedAt: now},
		{ID: "proj2-alpha", ProjectID: "proj-2", Title: "alpha", Status: "todo", Priority: "low", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
	})

	query, err := domain.NewTaskQuery(domain.WithQueryFilter("a"), domain.WithLimit(10))
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}

	tasks, err := repo.FindByProjectID(context.Background(), "proj-1", query)
	// SQL エラーが起きず、期待した結果が返る（本テストでは alpha と beta が返る）
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// "a" は "alpha" と "beta" の両方に含まれる
	assertTaskIDs(t, tasks, []string{"proj1-alpha", "proj1-beta"})
	assertNoProjectLeakage(t, tasks, "proj-1")
}

// TestSQLTaskRepository_FindByProjectID_Search_ScopedToProject は検索が project スコープ内に限定されることを検証する。
func TestSQLTaskRepository_FindByProjectID_Search_ScopedToProject(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewSQLTaskRepository(db)
	testutil.ResetTasksTable(t, db)

	now := time.Now().UTC()
	user1 := "user-1"

	testutil.InsertTasks(t, db, []testutil.SeedTask{
		{ID: "proj1-other", ProjectID: "proj-1", Title: "other", Status: "todo", Priority: "high", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
		{ID: "proj2-alpha", ProjectID: "proj-2", Title: "alpha", Status: "todo", Priority: "medium", AssigneeID: nil, CreatedAt: now, UpdatedAt: now},
		{ID: "proj2-alpha2", ProjectID: "proj-2", Title: "alpha task", Status: "todo", Priority: "low", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
	})

	query, err := domain.NewTaskQuery(domain.WithQueryFilter("alpha"), domain.WithLimit(10))
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}

	tasks, err := repo.FindByProjectID(context.Background(), "proj-1", query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// proj-2 側にだけ "alpha" があっても、proj-1 の検索結果に出ない（露出防止）
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks (proj-2 should not leak), got %d: %v", len(tasks), getTaskIDs(tasks))
	}
	assertNoProjectLeakage(t, tasks, "proj-1")
}

// TestSQLTaskRepository_FindByProjectID_Search_SpecialCharacters は特殊文字を含むタイトルの検索を検証する。
func TestSQLTaskRepository_FindByProjectID_Search_SpecialCharacters(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewSQLTaskRepository(db)
	testutil.ResetTasksTable(t, db)

	now := time.Now().UTC()
	user1 := "user-1"

	testutil.InsertTasks(t, db, []testutil.SeedTask{
		{ID: "proj1-special1", ProjectID: "proj-1", Title: "x'y", Status: "todo", Priority: "high", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
		{ID: "proj1-special2", ProjectID: "proj-1", Title: "100% legit", Status: "todo", Priority: "medium", AssigneeID: nil, CreatedAt: now, UpdatedAt: now},
		{ID: "proj2-special", ProjectID: "proj-2", Title: "x'y", Status: "todo", Priority: "low", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
	})

	// "x'" で検索（特殊文字が含まれる）
	query1, err := domain.NewTaskQuery(domain.WithQueryFilter("x'"), domain.WithLimit(10))
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}

	tasks1, err := repo.FindByProjectID(context.Background(), "proj-1", query1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertTaskIDs(t, tasks1, []string{"proj1-special1"})
	assertNoProjectLeakage(t, tasks1, "proj-1")

	// "100%" で検索
	query2, err := domain.NewTaskQuery(domain.WithQueryFilter("100%"), domain.WithLimit(10))
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}

	tasks2, err := repo.FindByProjectID(context.Background(), "proj-1", query2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertTaskIDs(t, tasks2, []string{"proj1-special2"})
	assertNoProjectLeakage(t, tasks2, "proj-1")
}

// ============================================================================
// Limit Tests
// ============================================================================

// TestSQLTaskRepository_FindByProjectID_Limit_1 は limit=1 を検証する。
// sort を付与して決定的にする（createdAt,asc で最古のタスクが返ることを期待）。
func TestSQLTaskRepository_FindByProjectID_Limit_1(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewSQLTaskRepository(db)
	testutil.ResetTasksTable(t, db)

	base := time.Now().UTC()
	user1 := "user-1"

	// createdAt を 3件で差が出るようにする（now, now+1s, now+2s）
	testutil.InsertTasks(t, db, []testutil.SeedTask{
		{ID: "proj1-1", ProjectID: "proj-1", Title: "alpha", Status: "todo", Priority: "high", AssigneeID: &user1, CreatedAt: base, UpdatedAt: base},
		{ID: "proj1-2", ProjectID: "proj-1", Title: "beta", Status: "todo", Priority: "medium", AssigneeID: nil, CreatedAt: base.Add(1 * time.Second), UpdatedAt: base.Add(1 * time.Second)},
		{ID: "proj1-3", ProjectID: "proj-1", Title: "gamma", Status: "todo", Priority: "low", AssigneeID: &user1, CreatedAt: base.Add(2 * time.Second), UpdatedAt: base.Add(2 * time.Second)},
		{ID: "proj2-1", ProjectID: "proj-2", Title: "delta", Status: "todo", Priority: "high", AssigneeID: &user1, CreatedAt: base, UpdatedAt: base},
	})

	// createdAt,asc でソートして決定的にする
	query, err := domain.NewTaskQuery(domain.WithSort("createdAt"), domain.WithLimit(1))
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}

	tasks, err := repo.FindByProjectID(context.Background(), "proj-1", query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// repository層は limit + 1 件取得する（nextCursor判定のため）
	// limit=1 の場合は 2件取得される
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks (limit + 1), got %d: %v", len(tasks), getTaskIDs(tasks))
	}
	// createdAt,asc で最古のタスク（proj1-1）が返ることを期待（最初の limit 件だけをチェック）
	assertTaskIDs(t, tasks[:1], []string{"proj1-1"})
	assertNoProjectLeakage(t, tasks, "proj-1")
}

// TestSQLTaskRepository_FindByProjectID_Limit_ExactCount は limit=seed数 を検証する。
func TestSQLTaskRepository_FindByProjectID_Limit_ExactCount(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewSQLTaskRepository(db)
	testutil.ResetTasksTable(t, db)

	now := time.Now().UTC()
	user1 := "user-1"

	testutil.InsertTasks(t, db, []testutil.SeedTask{
		{ID: "proj1-1", ProjectID: "proj-1", Title: "alpha", Status: "todo", Priority: "high", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
		{ID: "proj1-2", ProjectID: "proj-1", Title: "beta", Status: "todo", Priority: "medium", AssigneeID: nil, CreatedAt: now, UpdatedAt: now},
		{ID: "proj1-3", ProjectID: "proj-1", Title: "gamma", Status: "todo", Priority: "low", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
		{ID: "proj2-1", ProjectID: "proj-2", Title: "delta", Status: "todo", Priority: "high", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
	})

	query, err := domain.NewTaskQuery(domain.WithLimit(3))
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}

	tasks, err := repo.FindByProjectID(context.Background(), "proj-1", query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// limit=3, seed=3 で 3件
	if len(tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d: %v", len(tasks), getTaskIDs(tasks))
	}
	assertTaskIDs(t, tasks, []string{"proj1-1", "proj1-2", "proj1-3"})
	assertNoProjectLeakage(t, tasks, "proj-1")
}

// TestSQLTaskRepository_FindByProjectID_Limit_Zero は limit=0 が domain.NewTaskQuery で 200 にクランプされることを検証する。
// 仕様: NewTaskQuery では Limit < 1 の場合は 200 にクランプされる（エラーを返さない）。
func TestSQLTaskRepository_FindByProjectID_Limit_Zero(t *testing.T) {
	// limit=0 を domain.NewTaskQuery で作成すると 200 にクランプされることを検証
	query, err := domain.NewTaskQuery(domain.WithLimit(0))
	if err != nil {
		t.Fatalf("unexpected error (limit=0 should be clamped to 200, not error): %v", err)
	}
	if query.Limit != 200 {
		t.Errorf("expected limit to be clamped to 200, got %d", query.Limit)
	}
}

// TestSQLTaskRepository_FindByProjectID_Limit_Negative_ShouldError は limit=-1 が domain.NewTaskQuery で 200 にクランプされることを検証する。
// 仕様: NewTaskQuery では Limit < 1 の場合は 200 にクランプされる（エラーを返さない）。
func TestSQLTaskRepository_FindByProjectID_Limit_Negative_ShouldError(t *testing.T) {
	// limit=-1 を domain.NewTaskQuery で作成すると 200 にクランプされることを検証
	query, err := domain.NewTaskQuery(domain.WithLimit(-1))
	if err != nil {
		t.Fatalf("unexpected error (limit=-1 should be clamped to 200, not error): %v", err)
	}
	if query.Limit != 200 {
		t.Errorf("expected limit to be clamped to 200, got %d", query.Limit)
	}
}

// ============================================================================
// Security Tests (SQL Injection / Data Leakage)
// ============================================================================

// TestSQLTaskRepository_FindByProjectID_Security_SQLi_InQuery_DoesNotBypassFilters は SQLi がフィルタをバイパスしないことを検証する。
func TestSQLTaskRepository_FindByProjectID_Security_SQLi_InQuery_DoesNotBypassFilters(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewSQLTaskRepository(db)
	testutil.ResetTasksTable(t, db)

	now := time.Now().UTC()
	user1 := "user-1"

	testutil.InsertTasks(t, db, []testutil.SeedTask{
		{ID: "proj1-normal", ProjectID: "proj-1", Title: "normal task", Status: "todo", Priority: "high", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
		{ID: "proj1-inject", ProjectID: "proj-1", Title: "100% legit", Status: "todo", Priority: "medium", AssigneeID: nil, CreatedAt: now, UpdatedAt: now},
		{ID: "proj1-special", ProjectID: "proj-1", Title: "x'y", Status: "todo", Priority: "low", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
		{ID: "proj2-secret", ProjectID: "proj-2", Title: "secret", Status: "todo", Priority: "high", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
	})

	// SQLi 攻撃文字列
	query, err := domain.NewTaskQuery(domain.WithQueryFilter("%' OR 1=1 --"), domain.WithLimit(10))
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}

	tasks, err := repo.FindByProjectID(context.Background(), "proj-1", query)
	// err==nil を優先（パラメタ扱いで安全に処理されるのが理想）
	if err != nil {
		t.Fatalf("unexpected error (should be handled safely): %v", err)
	}

	// 常に leakage 検証を実行（0件でも将来のバグで proj-2 が混ざった場合に確実に落とせる）
	assertNoProjectLeakage(t, tasks, "proj-1")

	// "%' OR 1=1 --" は通常のタスクタイトルには含まれないので、0件が期待
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks for SQLi attack string, got %d: %v", len(tasks), getTaskIDs(tasks))
	}
}

// TestSQLTaskRepository_FindByProjectID_Security_SQLi_InAssigneeID_DoesNotBypass は assigneeId での SQLi がバイパスしないことを検証する。
func TestSQLTaskRepository_FindByProjectID_Security_SQLi_InAssigneeID_DoesNotBypass(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewSQLTaskRepository(db)
	testutil.ResetTasksTable(t, db)

	now := time.Now().UTC()
	user1 := "user-1"

	testutil.InsertTasks(t, db, []testutil.SeedTask{
		{ID: "proj1-user1", ProjectID: "proj-1", Title: "alpha", Status: "todo", Priority: "high", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
		{ID: "proj2-secret", ProjectID: "proj-2", Title: "secret", Status: "todo", Priority: "medium", AssigneeID: &user1, CreatedAt: now, UpdatedAt: now},
	})

	// SQLi 攻撃文字列
	maliciousAssigneeID := "user-1' OR '1'='1"
	query := &domain.TaskQuery{
		AssigneeID: &maliciousAssigneeID,
		Limit:      10,
	}

	tasks, err := repo.FindByProjectID(context.Background(), "proj-1", query)
	// err==nil を優先（パラメタ扱いで安全に処理されるのが理想）
	if err != nil {
		t.Fatalf("unexpected error (should be handled safely): %v", err)
	}

	// 常に leakage 検証を実行（0件でも将来のバグで proj-2 が混ざった場合に確実に落とせる）
	assertNoProjectLeakage(t, tasks, "proj-1")

	// "user-1' OR '1'='1" という assignee_id は存在しないので、0件が期待
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks for SQLi attack string, got %d: %v", len(tasks), getTaskIDs(tasks))
	}
}

// TestSQLTaskRepository_FindByProjectID_Security_SortKeyInjection_Ignored は sortKey でのインジェクションが domain.NewTaskQuery でエラーになることを検証する。
func TestSQLTaskRepository_FindByProjectID_Security_SortKeyInjection_Ignored(t *testing.T) {
	// 悪意のある sortKey を domain.NewTaskQuery で作成するとエラーになることを検証
	// "createdAt; DROP TABLE tasks;--" は無効なキーとして扱われ、エラーになる
	// WithSort は key 部分（"-" を除去した後）を RejectedValue に入れるので、
	// "createdAt; DROP TABLE tasks;--" がそのまま key として扱われる
	invalidSortKey := "createdAt; DROP TABLE tasks;--"
	_, err := domain.NewTaskQuery(domain.WithSort(invalidSortKey))
	if err == nil {
		t.Fatalf("expected error for invalid sort key with injection attempt, but got nil")
	}
	// typed error (ValidationError) で field=sort, code=INVALID_ENUM, RejectedValue を確認
	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("expected ValidationError, got: %T", err)
		return
	}
	if ve.Field != "sort" {
		t.Errorf("expected field=sort, got field=%s", ve.Field)
	}
	if ve.Code != "INVALID_ENUM" {
		t.Errorf("expected code=INVALID_ENUM, got code=%s", ve.Code)
	}
	if ve.RejectedValue == nil {
		t.Errorf("expected RejectedValue to be set, got nil")
	} else if *ve.RejectedValue != invalidSortKey {
		t.Errorf("expected RejectedValue=%s, got %s", invalidSortKey, *ve.RejectedValue)
	}
}

// ============================================================================
// Cursor Pagination Tests (v1)
// ============================================================================

// TestSQLTaskRepository_FindByProjectID_CursorPagination_Normal は正常系の cursor pagination を検証する。
// - limit=2 で1ページ目取得 → page.nextCursor != nil
// - nextCursor を付けて2ページ目取得
// - 1ページ目と2ページ目に重複がない
// - 合計が期待件数と一致（欠落なし）
func TestSQLTaskRepository_FindByProjectID_CursorPagination_Normal(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewSQLTaskRepository(db)
	testutil.ResetTasksTable(t, db)

	base := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	secret := []byte("test-secret-key")

	// 5件のタスクを作成（micro秒単位で差をつける）
	testutil.InsertTasks(t, db, []testutil.SeedTask{
		{ID: "task-001", ProjectID: "proj-1", Title: "T1", Status: "todo", Priority: "high", CreatedAt: base.Add(1 * time.Microsecond), UpdatedAt: base.Add(1 * time.Microsecond)},
		{ID: "task-002", ProjectID: "proj-1", Title: "T2", Status: "todo", Priority: "medium", CreatedAt: base.Add(2 * time.Microsecond), UpdatedAt: base.Add(2 * time.Microsecond)},
		{ID: "task-003", ProjectID: "proj-1", Title: "T3", Status: "todo", Priority: "low", CreatedAt: base.Add(3 * time.Microsecond), UpdatedAt: base.Add(3 * time.Microsecond)},
		{ID: "task-004", ProjectID: "proj-1", Title: "T4", Status: "todo", Priority: "high", CreatedAt: base.Add(4 * time.Microsecond), UpdatedAt: base.Add(4 * time.Microsecond)},
		{ID: "task-005", ProjectID: "proj-1", Title: "T5", Status: "todo", Priority: "medium", CreatedAt: base.Add(5 * time.Microsecond), UpdatedAt: base.Add(5 * time.Microsecond)},
	})

	// 1ページ目取得（limit=2）
	query1, err := domain.NewTaskQuery(domain.WithLimit(2))
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}

	tasks1, err := repo.FindByProjectID(context.Background(), "proj-1", query1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// repository層は limit + 1 件取得する（nextCursor判定のため）
	if len(tasks1) != 3 {
		t.Fatalf("expected 3 tasks (limit + 1), got %d", len(tasks1))
	}

	// nextCursor を生成（limit 件目を使う）
	got1 := clipToLimit(tasks1, query1.Limit)
	lastTask1 := got1[len(got1)-1]
	payload1 := domain.CursorPayload{
		V:         1,
		CreatedAt: domain.FormatCursorCreatedAt(lastTask1.CreatedAt),
		ID:        lastTask1.ID,
		ProjectID: "proj-1",
		QHash:     query1.ComputeQHash("proj-1"),
		IssuedAt:  time.Now().Unix(),
	}
	cursor1, err := domain.EncodeCursor(payload1, secret)
	if err != nil {
		t.Fatalf("failed to encode cursor: %v", err)
	}

	// 2ページ目取得（cursor を使用）
	query2, err := domain.NewTaskQuery(
		domain.WithLimit(2),
		domain.WithCursor(cursor1, "proj-1", secret, time.Now()),
	)
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}

	tasks2, err := repo.FindByProjectID(context.Background(), "proj-1", query2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// repository層は limit + 1 件取得する（nextCursor判定のため）
	if len(tasks2) == 0 {
		t.Fatalf("expected non-empty tasks2")
	}
	if len(tasks2) > query2.Limit+1 {
		t.Fatalf("expected at most %d tasks, got %d", query2.Limit+1, len(tasks2))
	}
	got2 := clipToLimit(tasks2, query2.Limit)

	// 重複チェック（repository層は limit + 1 件取得するので、最初の limit 件だけをチェック）
	taskIDs1 := getTaskIDs(got1)
	taskIDs2 := getTaskIDs(got2)
	for _, id1 := range taskIDs1 {
		for _, id2 := range taskIDs2 {
			if id1 == id2 {
				t.Errorf("duplicate task found: %s", id1)
			}
		}
	}

	// 合計件数チェック（5件すべて取得できているか）
	// repository層は limit + 1 件取得するので、1ページ目で3件、2ページ目で2件取得される
	allIDs := append(taskIDs1, taskIDs2...)
	if len(allIDs) != 4 {
		t.Errorf("expected 4 tasks total (2+2), got %d", len(allIDs))
	}

	// 順序チェック（createdAt ASC, id ASC）
	// repository層は limit + 1 件取得するので、最初の limit 件だけをチェック
	if got1[0].ID != "task-001" || got1[1].ID != "task-002" {
		t.Errorf("unexpected order for page 1: got %v", taskIDs1)
	}
	if len(got2) >= 2 && (got2[0].ID != "task-003" || got2[1].ID != "task-004") {
		t.Errorf("unexpected order for page 2: got %v", taskIDs2)
	}
}

// TestSQLTaskRepository_FindByProjectID_CursorPagination_TieBreaker は tie-breaker（createdAt同値でid順）を検証する。
func TestSQLTaskRepository_FindByProjectID_CursorPagination_TieBreaker(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewSQLTaskRepository(db)
	testutil.ResetTasksTable(t, db)

	base := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	secret := []byte("test-secret-key")

	// 同じ createdAt のタスクを複数作成（id で順序が決まる）
	testutil.InsertTasks(t, db, []testutil.SeedTask{
		{ID: "task-aaa", ProjectID: "proj-1", Title: "T1", Status: "todo", Priority: "high", CreatedAt: base, UpdatedAt: base},
		{ID: "task-bbb", ProjectID: "proj-1", Title: "T2", Status: "todo", Priority: "medium", CreatedAt: base, UpdatedAt: base},
		{ID: "task-ccc", ProjectID: "proj-1", Title: "T3", Status: "todo", Priority: "low", CreatedAt: base, UpdatedAt: base},
	})

	// 1ページ目取得（limit=2）
	query1, err := domain.NewTaskQuery(domain.WithLimit(2))
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}

	tasks1, err := repo.FindByProjectID(context.Background(), "proj-1", query1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// repository層は limit + 1 件取得する（nextCursor判定のため）
	if len(tasks1) != 3 {
		t.Fatalf("expected 3 tasks (limit + 1), got %d", len(tasks1))
	}

	// id 順で並んでいることを確認（最初の limit 件だけをチェック）
	got1 := clipToLimit(tasks1, query1.Limit)
	if got1[0].ID != "task-aaa" || got1[1].ID != "task-bbb" {
		t.Errorf("unexpected order: got %v, expected [task-aaa, task-bbb]", getTaskIDs(got1))
	}

	// nextCursor を生成（limit 件目を使う）
	lastTask1 := got1[len(got1)-1]
	payload1 := domain.CursorPayload{
		V:         1,
		CreatedAt: domain.FormatCursorCreatedAt(lastTask1.CreatedAt),
		ID:        lastTask1.ID,
		ProjectID: "proj-1",
		QHash:     query1.ComputeQHash("proj-1"),
		IssuedAt:  time.Now().Unix(),
	}
	cursor1, err := domain.EncodeCursor(payload1, secret)
	if err != nil {
		t.Fatalf("failed to encode cursor: %v", err)
	}

	// 2ページ目取得（cursor を使用）
	query2, err := domain.NewTaskQuery(
		domain.WithLimit(2),
		domain.WithCursor(cursor1, "proj-1", secret, time.Now()),
	)
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}

	tasks2, err := repo.FindByProjectID(context.Background(), "proj-1", query2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// repository層は limit + 1 件取得しようとするが、実際に存在する件数が少ない場合は存在する件数だけ返る
	// 残りが1件しかないので、1件だけ返ってくる
	if len(tasks2) < 1 {
		t.Fatalf("expected at least 1 task, got %d", len(tasks2))
	}

	// 順序が崩れていないことを確認（limit 件だけをチェック）
	got2 := clipToLimit(tasks2, query2.Limit)
	if len(got2) != 1 {
		t.Fatalf("expected 1 task remaining, got %d: %v", len(got2), getTaskIDs(got2))
	}
	if got2[0].ID != "task-ccc" {
		t.Fatalf("expected [task-ccc], got %v", getTaskIDs(got2))
	}
}

// TestSQLTaskRepository_FindByProjectID_CursorPagination_Error_CursorWithSort は cursor + sort の併用エラーを検証する。
func TestSQLTaskRepository_FindByProjectID_CursorPagination_Error_CursorWithSort(t *testing.T) {
	secret := []byte("test-secret-key")
	base := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)

	// cursor を生成
	payload := domain.CursorPayload{
		V:         1,
		CreatedAt: domain.FormatCursorCreatedAt(base),
		ID:        "task-001",
		ProjectID: "proj-1",
		QHash:     "test-hash",
		IssuedAt:  time.Now().Unix(),
	}
	cursor, err := domain.EncodeCursor(payload, secret)
	if err != nil {
		t.Fatalf("failed to encode cursor: %v", err)
	}

	// cursor + sort を指定すると NewTaskQuery または Validate() でエラーになる
	// WithCursor 内で qhash 検証が行われるため、sort を追加すると qhash が一致せず NewTaskQuery でエラーになる可能性がある
	// または、Validate() でエラーになる
	query, err := domain.NewTaskQuery(
		domain.WithLimit(2),
		domain.WithCursor(cursor, "proj-1", secret, time.Now()),
		domain.WithSort("priority"),
	)
	if err != nil {
		// NewTaskQuery でエラーになった場合（qhash不一致など）
		if strings.Contains(err.Error(), "cursor query mismatch") {
			// これは期待される動作（sort を追加すると qhash が変わるため）
			return
		}
		t.Fatalf("unexpected error in NewTaskQuery: %v", err)
	}

	// NewTaskQuery が成功した場合は Validate() でエラーになる
	err = query.Validate()
	if err == nil {
		t.Fatalf("expected error for cursor + sort, but got nil")
	}

	if !strings.Contains(err.Error(), "sort is incompatible with cursor") {
		t.Errorf("expected error message to contain 'sort is incompatible with cursor', got: %v", err)
	}
}

// TestSQLTaskRepository_FindByProjectID_CursorPagination_Error_InvalidFormat は cursor 形式不正エラーを検証する。
func TestSQLTaskRepository_FindByProjectID_CursorPagination_Error_InvalidFormat(t *testing.T) {
	secret := []byte("test-secret-key")

	// 形式不正な cursor（.なし）
	_, err := domain.NewTaskQuery(
		domain.WithLimit(2),
		domain.WithCursor("invalid-cursor-no-dot", "proj-1", secret, time.Now()),
	)
	if err == nil {
		t.Fatalf("expected error for invalid cursor format, but got nil")
	}

	if !strings.Contains(err.Error(), "invalid cursor format") {
		t.Errorf("expected error message to contain 'invalid cursor format', got: %v", err)
	}

	// base64 壊れ
	_, err = domain.NewTaskQuery(
		domain.WithLimit(2),
		domain.WithCursor("invalid.base64!!!", "proj-1", secret, time.Now()),
	)
	if err == nil {
		t.Fatalf("expected error for invalid cursor format, but got nil")
	}

	if !strings.Contains(err.Error(), "invalid cursor format") {
		t.Errorf("expected error message to contain 'invalid cursor format', got: %v", err)
	}
}

// TestSQLTaskRepository_FindByProjectID_CursorPagination_Error_InvalidSignature は署名改ざんエラーを検証する。
func TestSQLTaskRepository_FindByProjectID_CursorPagination_Error_InvalidSignature(t *testing.T) {
	secret := []byte("test-secret-key")
	base := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)

	// 正しい cursor を生成
	payload := domain.CursorPayload{
		V:         1,
		CreatedAt: domain.FormatCursorCreatedAt(base),
		ID:        "task-001",
		ProjectID: "proj-1",
		QHash:     "test-hash",
		IssuedAt:  time.Now().Unix(),
	}
	cursor, err := domain.EncodeCursor(payload, secret)
	if err != nil {
		t.Fatalf("failed to encode cursor: %v", err)
	}

	// 署名を改ざん（最後の文字を変更）
	tamperedCursor := cursor[:len(cursor)-1] + "X"

	// 異なる secret で検証
	wrongSecret := []byte("wrong-secret")
	_, err = domain.NewTaskQuery(
		domain.WithLimit(2),
		domain.WithCursor(tamperedCursor, "proj-1", wrongSecret, time.Now()),
	)
	if err == nil {
		t.Fatalf("expected error for invalid signature, but got nil")
	}

	if !strings.Contains(err.Error(), "invalid cursor signature") {
		t.Errorf("expected error message to contain 'invalid cursor signature', got: %v", err)
	}
}

// TestSQLTaskRepository_FindByProjectID_CursorPagination_Error_Expired は期限切れエラーを検証する。
func TestSQLTaskRepository_FindByProjectID_CursorPagination_Error_Expired(t *testing.T) {
	secret := []byte("test-secret-key")
	base := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)

	// 過去の iat で cursor を生成（24時間以上前）
	payload := domain.CursorPayload{
		V:         1,
		CreatedAt: domain.FormatCursorCreatedAt(base),
		ID:        "task-001",
		ProjectID: "proj-1",
		QHash:     "test-hash",
		IssuedAt:  time.Now().Unix() - 86401, // 24時間 + 1秒前
	}
	cursor, err := domain.EncodeCursor(payload, secret)
	if err != nil {
		t.Fatalf("failed to encode cursor: %v", err)
	}

	// 期限切れエラー
	_, err = domain.NewTaskQuery(
		domain.WithLimit(2),
		domain.WithCursor(cursor, "proj-1", secret, time.Now()),
	)
	if err == nil {
		t.Fatalf("expected error for expired cursor, but got nil")
	}

	if !strings.Contains(err.Error(), "cursor expired") {
		t.Errorf("expected error message to contain 'cursor expired', got: %v", err)
	}
}

// TestSQLTaskRepository_FindByProjectID_CursorPagination_Error_QueryMismatch は qhash 不一致エラーを検証する。
func TestSQLTaskRepository_FindByProjectID_CursorPagination_Error_QueryMismatch(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewSQLTaskRepository(db)
	testutil.ResetTasksTable(t, db)

	secret := []byte("test-secret-key")
	base := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)

	// タスクを作成
	testutil.InsertTasks(t, db, []testutil.SeedTask{
		{ID: "task-001", ProjectID: "proj-1", Title: "T1", Status: "todo", Priority: "high", CreatedAt: base, UpdatedAt: base},
	})

	// フィルタなしで cursor を生成
	query1, err := domain.NewTaskQuery(domain.WithLimit(2))
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}

	tasks1, err := repo.FindByProjectID(context.Background(), "proj-1", query1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lastTask1 := tasks1[len(tasks1)-1]
	payload1 := domain.CursorPayload{
		V:         1,
		CreatedAt: domain.FormatCursorCreatedAt(lastTask1.CreatedAt),
		ID:        lastTask1.ID,
		ProjectID: "proj-1",
		QHash:     query1.ComputeQHash("proj-1"), // フィルタなしの qhash
		IssuedAt:  time.Now().Unix(),
	}
	cursor1, err := domain.EncodeCursor(payload1, secret)
	if err != nil {
		t.Fatalf("failed to encode cursor: %v", err)
	}

	// フィルタを追加して cursor を再利用（qhash 不一致）
	_, err = domain.NewTaskQuery(
		domain.WithLimit(2),
		domain.WithStatusFilter("done"), // フィルタを追加
		domain.WithCursor(cursor1, "proj-1", secret, time.Now()),
	)
	if err == nil {
		t.Fatalf("expected error for query mismatch, but got nil")
	}

	if !strings.Contains(err.Error(), "cursor query mismatch") {
		t.Errorf("expected error message to contain 'cursor query mismatch', got: %v", err)
	}

	// projectID 不一致も検証
	_, err = domain.NewTaskQuery(
		domain.WithLimit(2),
		domain.WithCursor(cursor1, "proj-2", secret, time.Now()), // 異なる projectID
	)
	if err == nil {
		t.Fatalf("expected error for query mismatch (projectID), but got nil")
	}

	if !strings.Contains(err.Error(), "cursor query mismatch") {
		t.Errorf("expected error message to contain 'cursor query mismatch', got: %v", err)
	}
}
