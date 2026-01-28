package main

import (
	"log"
	"net/http"
	"time"

	infra "teamflow-projects/internal/infrastructure/project"
	httphandler "teamflow-projects/internal/interface/http"
	usecase "teamflow-projects/internal/usecase/project"
)

func main() {
	// インメモリのリポジトリ
	repo := infra.NewMemoryProjectRepository()

	// ユースケース
	createUC := &usecase.CreateProjectUsecase{
		Repo: repo,
	}
	updateUC := &usecase.UpdateProjectUsecase{
		Repo: repo,
	}
	listUC := &usecase.ListProjectsUsecase{
		Repo: repo,
	}

	// HTTP ハンドラ
	projectHandler := httphandler.NewProjectHandler(createUC, listUC, time.Now)
	updateHandler := httphandler.NewUpdateProjectHandler(updateUC, time.Now)

	mux := http.NewServeMux()
	mux.Handle("/projects", projectHandler) // POST /projects, GET /projects
	mux.Handle("/projects/", updateHandler) // PUT /projects/{id}

	// ヘルスチェック
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	addr := ":8080"
	log.Printf("projects service listening on %s", addr)

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
