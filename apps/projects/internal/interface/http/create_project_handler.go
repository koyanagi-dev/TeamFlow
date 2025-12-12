package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	usecase "teamflow-projects/internal/usecase/project"
)

// ProjectHandler はプロジェクト関連の HTTP ハンドラを提供する。
type ProjectHandler struct {
	createUC *usecase.CreateProjectUsecase
	listUC   *usecase.ListProjectsUsecase
	nowFunc  func() time.Time
}

// NewProjectHandler は ProjectHandler を生成する。
func NewProjectHandler(
	createUC *usecase.CreateProjectUsecase,
	listUC *usecase.ListProjectsUsecase,
	nowFunc func() time.Time,
) http.Handler {
	return &ProjectHandler{
		createUC: createUC,
		listUC:   listUC,
		nowFunc:  nowFunc,
	}
}

type createProjectRequest struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type projectResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// ServeHTTP は /projects を処理する。
// - POST: プロジェクト作成
// - GET : プロジェクト一覧取得
func (h *ProjectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.handleCreate(w, r)
	case http.MethodGet:
		h.handleList(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *ProjectHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req createProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	in := usecase.CreateProjectInput{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		Now:         h.nowFunc(),
	}

	p, err := h.createUC.Execute(r.Context(), in)
	if err != nil {
		// バリデーションエラー or その他（簡易判定）
		if errors.Is(err, context.DeadlineExceeded) {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	resp := projectResponse{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *ProjectHandler) handleList(w http.ResponseWriter, r *http.Request) {
	if h.listUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	projects, err := h.listUC.Execute(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	responses := make([]projectResponse, 0, len(projects))
	for _, p := range projects {
		responses = append(responses, projectResponse{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(responses)
}
