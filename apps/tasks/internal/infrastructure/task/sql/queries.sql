-- name: FindTasksByProjectID :many
-- FindByProjectID は projectID でタスクを検索し、フィルタ・ソート・リミットを適用する。
-- フィルタ条件は動的に組み立てるため、このクエリは基本形として定義し、
-- Go側で動的にWHERE句とORDER BY句を組み立てる。
-- 注意: このクエリは実際には使用されず、sql_repository.go で動的SQLを構築する。
-- ただし、sqlc の型生成のために最低限のクエリを定義する。
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
WHERE project_id = $1
ORDER BY created_at ASC, id ASC
LIMIT $2;

