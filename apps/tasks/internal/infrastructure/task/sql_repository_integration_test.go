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

// setupTestDB はテスト用のPostgreSQL接続をセットアップする。
// 環境変数から接続文字列を取得するか、デフォルト値を使用する。
// テスト環境で実際のDBを使用する場合は、この関数を実装する。
func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	// TODO: 実際のDB接続を設定する
	// 例: DATABASE_URL環境変数から取得、またはtestcontainersを使用
	t.Skip("database connection not configured for tests")
	return nil
}

// TestSQLTaskRepository_FindByProjectID_SortByPriority はpriorityソートを検証する。
func TestSQLTaskRepository_FindByProjectID_SortByPriority(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLTaskRepository(db)
	ctx := context.Background()
	now := time.Now()

	// テストデータ作成（実際のDBに保存する必要がある）
	// TODO: テストデータをDBに保存する処理を追加

	// priority DESC でソート（high > medium > low）
	query, err := domain.NewTaskQuery(domain.WithSort("-priority"))
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}

	tasks, err := repo.FindByProjectID(ctx, "proj-1", query)
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
	ctx := context.Background()

	// テストデータ作成
	// - task-1: dueDate = nil
	// - task-2: dueDate = 2024-01-15
	// - task-3: dueDate = 2024-01-10
	// TODO: テストデータをDBに保存する処理を追加

	// dueDate ASC でソート（NULLS LAST）
	query, err := domain.NewTaskQuery(domain.WithSort("dueDate"))
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}

	tasks, err := repo.FindByProjectID(ctx, "proj-1", query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}

	// ASC: 有効な日付が先、nullが最後
	// task-2 (2024-01-10) が最初
	// task-3 (2024-01-15) が次
	// task-1 (null) が最後
	// TODO: 実際のデータと比較して検証

	// dueDate DESC でソート（NULLS FIRST）
	queryDesc, err := domain.NewTaskQuery(domain.WithSort("-dueDate"))
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}

	tasksDesc, err := repo.FindByProjectID(ctx, "proj-1", queryDesc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// DESC: nullが最初、有効な日付が後
	// task-1 (null) が最初
	// task-3 (2024-01-15) が次
	// task-2 (2024-01-10) が最後
	// TODO: 実際のデータと比較して検証
}

// TestSQLTaskRepository_FindByProjectID_SortByCreatedAt はcreatedAtソートを検証する。
func TestSQLTaskRepository_FindByProjectID_SortByCreatedAt(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLTaskRepository(db)
	ctx := context.Background()
	baseTime := time.Now()

	// テストデータ作成
	// - task-1: createdAt = baseTime - 2h（最も古い）
	// - task-2: createdAt = baseTime - 1h
	// - task-3: createdAt = baseTime（最も新しい）
	// TODO: テストデータをDBに保存する処理を追加

	// createdAt ASC でソート
	query, err := domain.NewTaskQuery(domain.WithSort("createdAt"))
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}

	tasks, err := repo.FindByProjectID(ctx, "proj-1", query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}

	// 古い順（task-1, task-2, task-3）
	// TODO: 実際のデータと比較して検証
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

// TestSQLTaskRepository_FindByProjectID_MultipleSortKeys は複数キーでのソートを検証する。
func TestSQLTaskRepository_FindByProjectID_MultipleSortKeys(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLTaskRepository(db)
	ctx := context.Background()

	// テストデータ作成
	// - task-1: priority=high, createdAt=old
	// - task-2: priority=high, createdAt=new
	// - task-3: priority=low, createdAt=old
	// TODO: テストデータをDBに保存する処理を追加

	// priority DESC, createdAt ASC でソート
	query, err := domain.NewTaskQuery(domain.WithSort("-priority,createdAt"))
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}

	tasks, err := repo.FindByProjectID(ctx, "proj-1", query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// priorityが同じ場合はcreatedAtでソートされる
	// task-1 (high, old) が最初
	// task-2 (high, new) が次
	// task-3 (low, old) が最後
	// TODO: 実際のデータと比較して検証
}

