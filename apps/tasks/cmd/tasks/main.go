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

	// HTTP ハンドラ（POST /tasks, GET /tasks?projectId=...）
	taskHandler := httphandler.NewTaskHandler(createUC, listUC, updateUC, time.Now, cursorSecret)

	mux := http.NewServeMux()
	
	// API はすべて /api 配下
	mux.Handle("/api/", http.StripPrefix("/api", taskHandler))

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
