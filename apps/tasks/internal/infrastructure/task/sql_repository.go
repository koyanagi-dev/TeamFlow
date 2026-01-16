package taskinfra

import (
	"context"
	"fmt"
	"database/sql"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	domain "teamflow-tasks/internal/domain/task"
	usecase "teamflow-tasks/internal/usecase/task"
)

// SQLTaskRepository はPostgreSQLを使用したTaskRepository実装。
type SQLTaskRepository struct {
	db *pgxpool.Pool
}

// コンパイル時にインターフェース実装を保証する。
var _ usecase.TaskRepository = (*SQLTaskRepository)(nil)

// NewSQLTaskRepository は新しいSQLTaskRepositoryを生成する。
func NewSQLTaskRepository(db *pgxpool.Pool) *SQLTaskRepository {
	return &SQLTaskRepository{
		db: db,
	}
}

// Save はタスクを保存する（後回し）。
func (r *SQLTaskRepository) Save(_ context.Context, _ *domain.Task) error {
	return fmt.Errorf("not implemented yet")
}

// Update は既存タスクを更新する（後回し）。
func (r *SQLTaskRepository) Update(_ context.Context, _ *domain.Task) error {
	return fmt.Errorf("not implemented yet")
}

// FindByID はIDを指定してタスクを取得する（後回し）。
func (r *SQLTaskRepository) FindByID(_ context.Context, _ string) (*domain.Task, error) {
	return nil, fmt.Errorf("not implemented yet")
}

// ListByProject は指定されたprojectIDのタスク一覧を返す（後方互換性のため残す、後回し）。
func (r *SQLTaskRepository) ListByProject(_ context.Context, _ string) ([]*domain.Task, error) {
	return nil, fmt.Errorf("not implemented yet")
}

// FindByProjectID は指定されたprojectIDとQuery Objectに基づいてタスクを取得する。
func (r *SQLTaskRepository) FindByProjectID(ctx context.Context, projectID string, query *domain.TaskQuery) ([]*domain.Task, error) {
	// SQLクエリを動的に構築
	querySQL, args := r.buildQuery(projectID, query)

	rows, err := r.db.Query(ctx, querySQL, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*domain.Task
	for rows.Next() {
		var t domain.Task
		var assigneeID *string
		var dueDate *time.Time
		var description sql.NullString // ← ここは database/sql を使う

		err := rows.Scan(
			&t.ID,
			&t.ProjectID,
			&t.Title,
			&description,
			&t.Status,
			&t.Priority,
			&assigneeID,
			&dueDate,
			&t.CreatedAt,
			&t.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		t.AssigneeID = assigneeID
		t.DueDate = dueDate
		if description.Valid {
			t.Description = description.String
		}

		tasks = append(tasks, &t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return tasks, nil
}

// buildQuery はFindByProjectID用のSQLクエリを構築する。
// 戻り値: (SQL文字列, パラメータ配列)
func (r *SQLTaskRepository) buildQuery(projectID string, query *domain.TaskQuery) (string, []interface{}) {
	var whereParts []string
	var args []interface{}
	argIndex := 1

	// projectIDは必ず絞る
	whereParts = append(whereParts, fmt.Sprintf("project_id = $%d", argIndex))
	args = append(args, projectID)
	argIndex++

	// Status filter
	if len(query.Statuses) > 0 {
		placeholders := make([]string, len(query.Statuses))
		for i, status := range query.Statuses {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, string(status))
			argIndex++
		}
		whereParts = append(whereParts, fmt.Sprintf("status IN (%s)", strings.Join(placeholders, ", ")))
	}

	// Priority filter
	if len(query.Priorities) > 0 {
		placeholders := make([]string, len(query.Priorities))
		for i, priority := range query.Priorities {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, string(priority))
			argIndex++
		}
		whereParts = append(whereParts, fmt.Sprintf("priority IN (%s)", strings.Join(placeholders, ", ")))
	}

	// AssigneeID filter
	if query.AssigneeID != nil && *query.AssigneeID != "" {
		whereParts = append(whereParts, fmt.Sprintf("assignee_id = $%d", argIndex))
		args = append(args, *query.AssigneeID)
		argIndex++
	}

	// DueDate range filter
	if query.DueDateFrom != nil {
		whereParts = append(whereParts, fmt.Sprintf("due_date >= $%d::date", argIndex))
		args = append(args, query.DueDateFrom.Format("2006-01-02"))
		argIndex++
	}
	if query.DueDateTo != nil {
		whereParts = append(whereParts, fmt.Sprintf("due_date <= $%d::date", argIndex))
		args = append(args, query.DueDateTo.Format("2006-01-02"))
		argIndex++
	}

	// Query filter (title ILIKE)
	if query.Query != nil {
		whereParts = append(whereParts, fmt.Sprintf("title ILIKE $%d", argIndex))
		args = append(args, "%"+*query.Query+"%")
		argIndex++
	}

	// Cursor がある場合の seek 条件
	if query.Cursor != nil {
		// WHERE: (created_at > $X) OR (created_at = $X AND id > $Y)
		seekCondition := fmt.Sprintf("(created_at > $%d) OR (created_at = $%d AND id > $%d)", argIndex, argIndex, argIndex+1)
		whereParts = append(whereParts, seekCondition)
		args = append(args, query.Cursor.CreatedAt, query.Cursor.ID)
		argIndex += 2
	}

	// WHERE句を組み立て
	whereClause := ""
	if len(whereParts) > 0 {
		whereClause = "WHERE " + strings.Join(whereParts, " AND ")
	}

	// ORDER BY句を組み立て
	// cursor がある場合は created_at ASC, id ASC に固定（v1 の制限）
	var orderByClause string
	if query.Cursor != nil {
		// cursor 使用時は created_at ASC, id ASC に固定
		orderByClause = "ORDER BY created_at ASC, id ASC"
	} else {
		// cursor がない場合は既存のロジック
		orderByParts := r.buildOrderBy(query)
		if len(orderByParts) > 0 {
			orderByClause = "ORDER BY " + strings.Join(orderByParts, ", ")
		} else {
			// デフォルトソート: createdAt ASC
			orderByClause = "ORDER BY created_at ASC"
		}
		// 安定化のため、最後にid ASCを追加
		orderByClause += ", id ASC"
	}

	// LIMIT句（nextCursor 判定のため limit + 1 件取得）
	// 1ページ目（cursor が nil）でも limit + 1 件取得して nextCursor 判定を行う
	limitValue := query.Limit + 1
	limitClause := fmt.Sprintf("LIMIT $%d", argIndex)
	args = append(args, limitValue)

	// 最終的なSQL
	sql := fmt.Sprintf(`
		SELECT
			id,
			project_id,
			title,
			description,
			status,
			priority,
			assignee_id,
			due_date,
			created_at,
			updated_at
		FROM tasks
		%s
		%s
		%s
	`, whereClause, orderByClause, limitClause)

	return sql, args
}

// buildOrderBy はORDER BY句を構築する（ホワイトリストで安全に）。
func (r *SQLTaskRepository) buildOrderBy(query *domain.TaskQuery) []string {
	if len(query.SortOrders) == 0 {
		return nil
	}

	var orderByParts []string
	validKeys := map[string]bool{
		"sortOrder": true,
		"createdAt": true,
		"updatedAt": true,
		"dueDate":   true,
		"priority":  true,
	}

	for _, order := range query.SortOrders {
		// ホワイトリストチェック
		if !validKeys[order.Key] {
			continue
		}

		var orderExpr string
		switch order.Key {
		case "priority":
			// priorityの業務順：high>medium>low（CASEで数値化）
			// ASC: 小さい順（low=1, medium=2, high=3）
			// DESC: 大きい順（high=3, medium=2, low=1）
			orderExpr = fmt.Sprintf("CASE priority WHEN 'high' THEN 3 WHEN 'medium' THEN 2 WHEN 'low' THEN 1 ELSE 0 END %s", order.Direction)
		case "dueDate":
			// dueDate null順：ASCはNULLS LAST、DESCはNULLS FIRST
			if order.Direction == domain.SortDirectionASC {
				orderExpr = "due_date ASC NULLS LAST"
			} else {
				orderExpr = "due_date DESC NULLS FIRST"
			}
		case "createdAt":
			orderExpr = fmt.Sprintf("created_at %s", order.Direction)
		case "updatedAt":
			orderExpr = fmt.Sprintf("updated_at %s", order.Direction)
		case "sortOrder":
			// sortOrderは現在テーブルにないため、スキップ（将来対応）
			continue
		default:
			continue
		}

		if orderExpr != "" {
			orderByParts = append(orderByParts, orderExpr)
		}
	}

	return orderByParts
}

