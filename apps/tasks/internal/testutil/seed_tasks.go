//go:build integration
// +build integration

package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SeedTask represents a task to be inserted for testing.
type SeedTask struct {
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

// InsertTasks inserts tasks into the database for testing.
func InsertTasks(t *testing.T, db *pgxpool.Pool, tasks []SeedTask) {
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

// DateYMD creates a time.Time at midnight UTC for a given date (for DATE fields).
func DateYMD(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
