package main

import (
	"log"
	"net/http"
	"os"
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

	mux := http.NewServeMux()
	
	// API はすべて /api 配下
	// POST /api/tasks と GET /api/tasks?projectId=xxx (旧API)
	mux.Handle("/api/tasks", createHandler)
	mux.Handle("/api/tasks", listHandler)
	// GET /api/projects/{projectId}/tasks (新API)
	mux.Handle("/api/projects/", listHandler)
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
