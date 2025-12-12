package projectinfra

import (
	"context"
	"errors"

	domain "teamflow-projects/internal/domain/project"
	usecase "teamflow-projects/internal/usecase/project"
)

// MemoryProjectRepository はメモリ上にプロジェクトを保持する
// シンプルな ProjectRepository 実装。
type MemoryProjectRepository struct {
	projects map[string]*domain.Project
}

// コンパイル時にインターフェース実装を保証する。
var _ usecase.ProjectRepository = (*MemoryProjectRepository)(nil)

// ErrProjectNotFound は指定した ID のプロジェクトが存在しない場合のエラー。
var ErrProjectNotFound = errors.New("project not found")

// NewMemoryProjectRepository は空のインメモリリポジトリを生成する。
func NewMemoryProjectRepository() *MemoryProjectRepository {
	return &MemoryProjectRepository{
		projects: make(map[string]*domain.Project),
	}
}

// Save はプロジェクトをメモリ上に保存する。
func (r *MemoryProjectRepository) Save(_ context.Context, p *domain.Project) error {
	if r.projects == nil {
		r.projects = make(map[string]*domain.Project)
	}
	r.projects[p.ID] = p
	return nil
}

// FindByID は ID を指定してプロジェクトを取得する。
func (r *MemoryProjectRepository) FindByID(_ context.Context, id string) (*domain.Project, error) {
	if r.projects == nil {
		return nil, ErrProjectNotFound
	}
	p, ok := r.projects[id]
	if !ok {
		return nil, ErrProjectNotFound
	}
	return p, nil
}

// List はすべてのプロジェクトを返す。
func (r *MemoryProjectRepository) List(_ context.Context) ([]*domain.Project, error) {
	out := make([]*domain.Project, 0, len(r.projects))
	for _, p := range r.projects {
		out = append(out, p)
	}
	return out, nil
}
