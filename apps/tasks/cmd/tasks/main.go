package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	infra "teamflow-tasks/internal/infrastructure/task"
	httphandler "teamflow-tasks/internal/interface/http"
	usecase "teamflow-tasks/internal/usecase/task"
)

func main() {
	// インメモリのタスクリポジトリ
	repo := infra.NewMemoryTaskRepository()

	// ユースケース
	createUC := &usecase.CreateTaskUsecase{
		Repo: repo,
	}
	listUC := &usecase.ListTasksByProjectUsecase{
		Repo: repo,
	}
	updateUC := &usecase.UpdateTaskUsecase{
		Repo: repo,
	}

	// cursor secret（環境変数から取得、環境に応じて検証）
	appEnv := os.Getenv("APP_ENV")
	rawSecret := os.Getenv("CURSOR_SECRET")

	cursorSecret, err := resolveCursorSecret(appEnv, rawSecret)
	if err != nil {
		log.Fatal(err)
	}

	// HTTP ハンドラ
	createHandler := httphandler.NewCreateTaskHandler(createUC, time.Now)
	listHandler := httphandler.NewListTaskHandler(listUC, time.Now, cursorSecret)
	updateHandler := httphandler.NewUpdateTaskHandler(updateUC)

	// /api/tasks の統合ハンドラ（POST と GET の両方を処理）
	tasksHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			createHandler.ServeHTTP(w, r)
		case http.MethodGet:
			listHandler.ServeHTTP(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// /api/projects/{projectId}/tasks の統合ハンドラ（GET と POST の両方を処理）
	projectTasksHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// パスから projectId を抽出: /api/projects/{projectId}/tasks
		path := strings.TrimPrefix(r.URL.Path, "/api/projects/")
		parts := strings.Split(path, "/")

		if len(parts) < 2 || parts[1] != "tasks" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		projectID := parts[0]

		switch r.Method {
		case http.MethodGet:
			// GET /api/projects/{projectId}/tasks
			listHandler.ServeHTTP(w, r)
		case http.MethodPost:
			// POST /api/projects/{projectId}/tasks
			// パスから取得した projectId を body に追加して CreateTaskHandler に渡す
			body, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			r.Body.Close()

			// JSON を map にデコードして projectId を追加
			var reqMap map[string]interface{}
			if err := json.Unmarshal(body, &reqMap); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// projectId を追加（上書き）
			reqMap["projectId"] = projectID

			// 新しい body を作成
			newBody, err := json.Marshal(reqMap)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// リクエストボディを差し替え
			r.Body = io.NopCloser(strings.NewReader(string(newBody)))
			r.ContentLength = int64(len(newBody))

			createHandler.ServeHTTP(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	mux := http.NewServeMux()

	// API はすべて /api 配下
	// POST /api/tasks と GET /api/tasks?projectId=xxx (旧API)
	mux.Handle("/api/tasks", tasksHandler)
	// GET /api/projects/{projectId}/tasks と POST /api/projects/{projectId}/tasks (OpenAPI準拠)
	mux.Handle("/api/projects/", projectTasksHandler)
	// PATCH /api/tasks/{id}
	mux.Handle("/api/tasks/", updateHandler)

	// ヘルスチェック
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	addr := ":8081"
	log.Printf("tasks service listening on %s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
