package project

import (
	"errors"
	"time"
)

// Project は TeamFlow におけるプロジェクトのドメインモデル。
type Project struct {
	ID          string
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewProject は新しいプロジェクトを生成する。
// Name が空の場合はエラーを返す。
func NewProject(id, name, description string, now time.Time) (*Project, error) {
	if name == "" {
		return nil, errors.New("project name must not be empty")
	}

	return &Project{
		ID:          id,
		Name:        name,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}
