//go:build integration
// +build integration

package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestPool is initialized in TestMain.
// We keep it in this package scope so integration tests can share a single DB pool.
var TestPool *pgxpool.Pool

// SetupTestDB returns the integration-test pool.
// It fails fast if TestMain didn't initialize the pool.
func SetupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if TestPool == nil {
		t.Fatalf("TestPool is nil: ensure TestMain initialized it (go test -tags=integration ./... with DB_TEST_DSN)")
	}
	return TestPool
}

// ResetTasksTable truncates the tasks table.
func ResetTasksTable(t *testing.T, db *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	_, err := db.Exec(ctx, "TRUNCATE TABLE tasks")
	if err != nil {
		t.Fatalf("failed to truncate tasks: %v", err)
	}
}

// WaitForDB waits for the database to be ready.
func WaitForDB(ctx context.Context, dsn string, timeout time.Duration) (*pgxpool.Pool, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		pool, err := pgxpool.New(ctx, dsn)
		if err == nil {
			pctx, cancel := context.WithTimeout(ctx, 2*time.Second)
			err = pool.Ping(pctx)
			cancel()
			if err == nil {
				return pool, nil
			}
			pool.Close()
		}
		time.Sleep(300 * time.Millisecond)
	}
	return nil, fmt.Errorf("timeout waiting for db")
}

// ApplySchema applies the database schema from sql/schema.sql.
func ApplySchema(ctx context.Context, pool *pgxpool.Pool) error {
	// Resolve schema.sql path based on this source file location (robust against CWD differences)
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("runtime.Caller failed")
	}
	baseDir := filepath.Dir(thisFile) // .../internal/testutil
	// Go up to internal, then to infrastructure/task/sql
	schemaPath := filepath.Join(baseDir, "..", "infrastructure", "task", "sql", "schema.sql")

	b, err := os.ReadFile(schemaPath)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, string(b))
	return err
}

// TestMain initializes the test database pool.
// This should be called from a TestMain function in the test package.
func InitTestDB(m *testing.M) int {
	dsn := os.Getenv("DB_TEST_DSN")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "DB_TEST_DSN is required")
		return 2
	}

	ctx := context.Background()

	pool, err := WaitForDB(ctx, dsn, 30*time.Second)
	if err != nil {
		fmt.Fprintln(os.Stderr, "db not ready:", err)
		return 1
	}
	TestPool = pool

	if err := ApplySchema(ctx, TestPool); err != nil {
		// スキーマが既に存在する場合はエラーを無視（複数のテストパッケージが同じDBを使う場合）
		if !strings.Contains(err.Error(), "already exists") {
			fmt.Fprintln(os.Stderr, "apply schema failed:", err)
			TestPool.Close()
			return 1
		}
	}

	code := m.Run()

	TestPool.Close()
	return code
}
