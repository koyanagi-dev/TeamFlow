//go:build integration
// +build integration

package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	domain "teamflow-tasks/internal/domain/task"
	taskinfra "teamflow-tasks/internal/infrastructure/task"
	"teamflow-tasks/internal/testutil"
	usecase "teamflow-tasks/internal/usecase/task"
)

func TestMain(m *testing.M) {
	code := testutil.InitTestDB(m)
	os.Exit(code)
}

func TestTaskHandler_CursorPagination_FirstPageReturnsNextCursor(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.ResetTasksTable(t, db)

	// Setup handler with real dependencies
	repo := taskinfra.NewSQLTaskRepository(db)
	listUC := &usecase.ListTasksByProjectUsecase{Repo: repo}
	nowFunc := func() time.Time { return time.Now().UTC() }
	cursorSecret := []byte("test-secret")
	handler := NewListTaskHandler(listUC, nowFunc, cursorSecret)

	// Seed: 5件以上、limit=2で複数ページになる数
	// createdAt が同一の行を最低2件含める（tie-breaker: id）
	// テスト間でIDが重複しないように、一意のIDを使用
	testID := "first-page-next-cursor"
	base := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	testutil.InsertTasks(t, db, []testutil.SeedTask{
		{ID: testID + "-001", ProjectID: "proj-1", Title: "T1", Status: "todo", Priority: "high", CreatedAt: base.Add(1 * time.Microsecond), UpdatedAt: base.Add(1 * time.Microsecond)},
		{ID: testID + "-002", ProjectID: "proj-1", Title: "T2", Status: "todo", Priority: "medium", CreatedAt: base.Add(2 * time.Microsecond), UpdatedAt: base.Add(2 * time.Microsecond)},
		{ID: testID + "-003", ProjectID: "proj-1", Title: "T3", Status: "todo", Priority: "low", CreatedAt: base, UpdatedAt: base},  // 同じcreatedAt
		{ID: testID + "-004", ProjectID: "proj-1", Title: "T4", Status: "todo", Priority: "high", CreatedAt: base, UpdatedAt: base}, // 同じcreatedAt
		{ID: testID + "-005", ProjectID: "proj-1", Title: "T5", Status: "todo", Priority: "medium", CreatedAt: base.Add(3 * time.Microsecond), UpdatedAt: base.Add(3 * time.Microsecond)},
	})

	// 1ページ目: GET /api/projects/{projectId}/tasks?limit=2
	req1 := httptest.NewRequest(http.MethodGet, "/projects/proj-1/tasks?limit=2", nil)
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body: %s", w1.Code, w1.Body.String())
	}

	var resp1 listTasksResponse
	if err := json.NewDecoder(w1.Body).Decode(&resp1); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// 期待: page.nextCursor != null（次があるなら返る）
	if resp1.Page == nil {
		t.Fatal("expected page to be present")
	}
	if resp1.Page.NextCursor == nil {
		t.Fatal("expected nextCursor to be present on first page when there are more items")
	}

	// レスポンスのタスク数は limit 件
	if len(resp1.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(resp1.Tasks))
	}

	// 2ページ目以降: nextCursor を追って全件回収
	allTaskIDs := make(map[string]bool)
	for _, task := range resp1.Tasks {
		allTaskIDs[task.ID] = true
	}

	nextCursor := resp1.Page.NextCursor
	pageNum := 2
	for nextCursor != nil {
		req := httptest.NewRequest(http.MethodGet, "/projects/proj-1/tasks?limit=2&cursor="+*nextCursor, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("page %d: expected status 200, got %d, body: %s", pageNum, w.Code, w.Body.String())
		}

		var resp listTasksResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("page %d: failed to decode response: %v", pageNum, err)
		}

		// 重複チェック
		for _, task := range resp.Tasks {
			if allTaskIDs[task.ID] {
				t.Errorf("duplicate task ID found: %s", task.ID)
			}
			allTaskIDs[task.ID] = true
		}

		// 2ページ目で取得したデータの1件目が、1ページ目の最後のデータより「後」であることを検証
		if pageNum == 2 && len(resp.Tasks) > 0 {
			lastTaskPage1 := resp1.Tasks[len(resp1.Tasks)-1]
			firstTaskPage2 := resp.Tasks[0]
			if lastTaskPage1.CreatedAt.After(firstTaskPage2.CreatedAt) {
				t.Errorf("page 2 first task should be after page 1 last task")
			}
			if lastTaskPage1.CreatedAt.Equal(firstTaskPage2.CreatedAt) && lastTaskPage1.ID >= firstTaskPage2.ID {
				t.Errorf("page 2 first task should be after page 1 last task (tie-breaker)")
			}
		}

		nextCursor = resp.Page.NextCursor
		pageNum++

		// 無限ループ防止（最大10ページまで）
		if pageNum > 10 {
			t.Fatalf("too many pages, possible infinite loop")
		}
	}

	// 検証: 取得したタスクの ID が重複なし
	if len(allTaskIDs) != 5 {
		t.Errorf("expected 5 unique tasks, got %d", len(allTaskIDs))
	}

	// 検証: 欠落なし（seedした全件が回収できる）
	expectedIDs := []string{testID + "-001", testID + "-002", testID + "-003", testID + "-004", testID + "-005"}
	for _, expectedID := range expectedIDs {
		if !allTaskIDs[expectedID] {
			t.Errorf("expected task ID %s not found", expectedID)
		}
	}

	// 検証: 最終ページの nextCursor == null
	// (上記のループで nextCursor が nil になった時点で終了しているので、これは既に検証済み)
}

// TestTaskHandler_CursorPagination_Error_INCOMPATIBLE_WITH_CURSOR は cursor + sort の併用エラーを検証する。
func TestTaskHandler_CursorPagination_Error_INCOMPATIBLE_WITH_CURSOR(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.ResetTasksTable(t, db)

	repo := taskinfra.NewSQLTaskRepository(db)
	listUC := &usecase.ListTasksByProjectUsecase{Repo: repo}
	nowFunc := func() time.Time { return time.Now().UTC() }
	cursorSecret := []byte("test-secret")
	handler := NewListTaskHandler(listUC, nowFunc, cursorSecret)

	// 有効な cursor を生成
	base := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	payload := domain.CursorPayload{
		V:         1,
		CreatedAt: domain.FormatCursorCreatedAt(base),
		ID:        "task-001",
		ProjectID: "proj-1",
		QHash:     "test-hash",
		IssuedAt:  time.Now().Unix(),
	}
	validCursor, err := domain.EncodeCursor(payload, cursorSecret)
	if err != nil {
		t.Fatalf("failed to encode cursor: %v", err)
	}

	// cursor + sort を指定
	req := httptest.NewRequest(http.MethodGet, "/projects/proj-1/tasks?limit=2&cursor="+validCursor+"&sort=createdAt", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Details == nil || len(resp.Details.Issues) == 0 {
		t.Fatal("expected validation issues")
	}

	found := false
	for _, issue := range resp.Details.Issues {
		if issue.Code == "INCOMPATIBLE_WITH_CURSOR" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected INCOMPATIBLE_WITH_CURSOR, got issues: %+v", resp.Details.Issues)
	}
}

// TestTaskHandler_CursorPagination_Error_INVALID_FORMAT は cursor 形式不正エラーを検証する。
func TestTaskHandler_CursorPagination_Error_INVALID_FORMAT(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.ResetTasksTable(t, db)

	repo := taskinfra.NewSQLTaskRepository(db)
	listUC := &usecase.ListTasksByProjectUsecase{Repo: repo}
	nowFunc := func() time.Time { return time.Now().UTC() }
	cursorSecret := []byte("test-secret")
	handler := NewListTaskHandler(listUC, nowFunc, cursorSecret)

	// 形式不正な cursor（ドットなし）
	req1 := httptest.NewRequest(http.MethodGet, "/projects/proj-1/tasks?limit=2&cursor=not-a-valid-cursor", nil)
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)

	if w1.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body: %s", w1.Code, w1.Body.String())
	}

	var resp1 ErrorResponse
	if err := json.NewDecoder(w1.Body).Decode(&resp1); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp1.Details == nil || len(resp1.Details.Issues) == 0 {
		t.Fatal("expected validation issues")
	}

	found := false
	for _, issue := range resp1.Details.Issues {
		if issue.Code == "INVALID_FORMAT" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected INVALID_FORMAT, got issues: %+v", resp1.Details.Issues)
	}

	// base64 壊れ
	req2 := httptest.NewRequest(http.MethodGet, "/projects/proj-1/tasks?limit=2&cursor=invalid.base64!!!", nil)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body: %s", w2.Code, w2.Body.String())
	}
}

// TestTaskHandler_CursorPagination_Error_INVALID_SIGNATURE は署名改ざんエラーを検証する。
func TestTaskHandler_CursorPagination_Error_INVALID_SIGNATURE(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.ResetTasksTable(t, db)

	repo := taskinfra.NewSQLTaskRepository(db)
	listUC := &usecase.ListTasksByProjectUsecase{Repo: repo}
	nowFunc := func() time.Time { return time.Now().UTC() }
	cursorSecret := []byte("test-secret")
	handler := NewListTaskHandler(listUC, nowFunc, cursorSecret)

	// 正しい cursor を生成（qhash を計算するために query を作成）
	base := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	query, err := domain.NewTaskQuery(domain.WithLimit(2))
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}
	payload := domain.CursorPayload{
		V:         1,
		CreatedAt: domain.FormatCursorCreatedAt(base),
		ID:        "task-001",
		ProjectID: "proj-1",
		QHash:     query.ComputeQHash("proj-1"),
		IssuedAt:  time.Now().Unix(),
	}
	validCursor, err := domain.EncodeCursor(payload, cursorSecret)
	if err != nil {
		t.Fatalf("failed to encode cursor: %v", err)
	}

	// 署名を改ざん（最後の文字を変更）
	tamperedCursor := validCursor[:len(validCursor)-1] + "X"

	req := httptest.NewRequest(http.MethodGet, "/projects/proj-1/tasks?limit=2&cursor="+tamperedCursor, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Details == nil || len(resp.Details.Issues) == 0 {
		t.Fatal("expected validation issues")
	}

	found := false
	for _, issue := range resp.Details.Issues {
		if issue.Code == "INVALID_SIGNATURE" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected INVALID_SIGNATURE, got issues: %+v", resp.Details.Issues)
	}
}

// TestTaskHandler_CursorPagination_Error_EXPIRED は期限切れエラーを検証する。
func TestTaskHandler_CursorPagination_Error_EXPIRED(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.ResetTasksTable(t, db)

	repo := taskinfra.NewSQLTaskRepository(db)
	listUC := &usecase.ListTasksByProjectUsecase{Repo: repo}
	nowFunc := func() time.Time { return time.Now().UTC() }
	cursorSecret := []byte("test-secret")
	handler := NewListTaskHandler(listUC, nowFunc, cursorSecret)

	// 過去の iat で cursor を生成（24時間以上前）
	base := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	payload := domain.CursorPayload{
		V:         1,
		CreatedAt: domain.FormatCursorCreatedAt(base),
		ID:        "task-001",
		ProjectID: "proj-1",
		QHash:     "test-hash",
		IssuedAt:  time.Now().Unix() - 86401, // 24時間 + 1秒前
	}
	expiredCursor, err := domain.EncodeCursor(payload, cursorSecret)
	if err != nil {
		t.Fatalf("failed to encode cursor: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/projects/proj-1/tasks?limit=2&cursor="+expiredCursor, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Details == nil || len(resp.Details.Issues) == 0 {
		t.Fatal("expected validation issues")
	}

	found := false
	for _, issue := range resp.Details.Issues {
		if issue.Code == "EXPIRED" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected EXPIRED, got issues: %+v", resp.Details.Issues)
	}
}

// TestTaskHandler_CursorPagination_Error_QUERY_MISMATCH は qhash 不一致エラーを検証する。
func TestTaskHandler_CursorPagination_Error_QUERY_MISMATCH(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.ResetTasksTable(t, db)

	repo := taskinfra.NewSQLTaskRepository(db)
	listUC := &usecase.ListTasksByProjectUsecase{Repo: repo}
	nowFunc := func() time.Time { return time.Now().UTC() }
	cursorSecret := []byte("test-secret")
	handler := NewListTaskHandler(listUC, nowFunc, cursorSecret)

	// フィルタなしで cursor を生成
	base := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	testutil.InsertTasks(t, db, []testutil.SeedTask{
		{ID: "task-001", ProjectID: "proj-1", Title: "T1", Status: "todo", Priority: "high", CreatedAt: base, UpdatedAt: base},
	})

	query1, err := domain.NewTaskQuery(domain.WithLimit(2))
	if err != nil {
		t.Fatalf("failed to create query: %v", err)
	}

	ctx := context.Background()
	tasks1, err := repo.FindByProjectID(ctx, "proj-1", query1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tasks1) == 0 {
		t.Fatal("expected at least one task")
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
	cursor1, err := domain.EncodeCursor(payload1, cursorSecret)
	if err != nil {
		t.Fatalf("failed to encode cursor: %v", err)
	}

	// フィルタを追加して cursor を再利用（qhash 不一致）
	req := httptest.NewRequest(http.MethodGet, "/projects/proj-1/tasks?limit=2&cursor="+cursor1+"&status=done", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Details == nil || len(resp.Details.Issues) == 0 {
		t.Fatal("expected validation issues")
	}

	found := false
	for _, issue := range resp.Details.Issues {
		if issue.Code == "QUERY_MISMATCH" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected QUERY_MISMATCH, got issues: %+v", resp.Details.Issues)
	}
}

// listTasksResponse はレスポンス構造体（テスト用）
type listTasksResponse struct {
	Tasks []taskResponse `json:"tasks"`
	Page  *pageInfo      `json:"page,omitempty"`
}

type pageInfo struct {
	NextCursor *string `json:"nextCursor,omitempty"`
	Limit      int     `json:"limit,omitempty"`
}
