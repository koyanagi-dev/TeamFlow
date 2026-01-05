//go:build integration

package taskinfra

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMain(m *testing.M) {
	dsn := os.Getenv("DB_TEST_DSN")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "DB_TEST_DSN is required")
		os.Exit(2)
	}

	ctx := context.Background()

	pool, err := waitForDB(ctx, dsn, 30*time.Second)
	if err != nil {
		fmt.Fprintln(os.Stderr, "db not ready:", err)
		os.Exit(1)
	}
	testPool = pool

	if err := applySchema(ctx, testPool); err != nil {
		fmt.Fprintln(os.Stderr, "apply schema failed:", err)
		testPool.Close()
		os.Exit(1)
	}

	code := m.Run()

	testPool.Close()
	os.Exit(code)
}

func waitForDB(ctx context.Context, dsn string, timeout time.Duration) (*pgxpool.Pool, error) {
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

func applySchema(ctx context.Context, pool *pgxpool.Pool) error {
	// Resolve schema.sql path based on this source file location (robust against CWD differences)
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("runtime.Caller failed")
	}
	baseDir := filepath.Dir(thisFile) // .../internal/infrastructure/task
	schemaPath := filepath.Join(baseDir, "sql", "schema.sql")

	b, err := os.ReadFile(schemaPath)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, string(b))
	return err
}
