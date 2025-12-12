package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	infra "teamflow-projects/internal/infrastructure/project"
	usecase "teamflow-projects/internal/usecase/project"
)

type updateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// UpdateProjectHandler は PUT /projects/{id} を処理する HTTP ハンドラ。
type UpdateProjectHandler struct {
	updateUC *usecase.UpdateProjectUsecase
	nowFunc  func() time.Time
}

// NewUpdateProjectHandler は UpdateProjectHandler を生成する。
func NewUpdateProjectHandler(updateUC *usecase.UpdateProjectUsecase, nowFunc func() time.Time) http.Handler {
	return &UpdateProjectHandler{
		updateUC: updateUC,
		nowFunc:  nowFunc,
	}
}

func (h *UpdateProjectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// パスから /projects/{id} の {id} 部分を取り出す
	path := strings.TrimPrefix(r.URL.Path, "/projects/")
	if path == "" || strings.Contains(path, "/") {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	id := path

	var req updateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	in := usecase.UpdateProjectInput{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Now:         h.nowFunc(),
	}

	p, err := h.updateUC.Execute(r.Context(), in)
	if err != nil {
		// name 空などのバリデーションエラー
		if errors.Is(err, infra.ErrProjectNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// UpdateProjectUsecase 側では name 空の場合は errors.New("project name must not be empty")
		// としているので、それっぽい文言なら 400 にする。
		if strings.Contains(err.Error(), "must not be empty") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// その他は内部エラー
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// ここがポイント：createProjectResponse ではなく projectResponse を使う
	resp := projectResponse{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
