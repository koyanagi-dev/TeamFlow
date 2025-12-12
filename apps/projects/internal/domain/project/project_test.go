package project

import (
	"testing"
	"time"
)

func TestNewProject_Success(t *testing.T) {
	now := time.Now()

	p, err := NewProject("proj-1", "TeamFlow 開発", "TeamFlow の開発プロジェクト", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.ID != "proj-1" {
		t.Errorf("expected ID=proj-1, got=%s", p.ID)
	}

	if p.Name != "TeamFlow 開発" {
		t.Errorf("expected Name=TeamFlow 開発, got=%s", p.Name)
	}

	if p.Description != "TeamFlow の開発プロジェクト" {
		t.Errorf("expected Description to match, got=%s", p.Description)
	}

	if !p.CreatedAt.Equal(now) {
		t.Errorf("expected CreatedAt to equal now, got=%v", p.CreatedAt)
	}

	if !p.UpdatedAt.Equal(now) {
		t.Errorf("expected UpdatedAt to equal now, got=%v", p.UpdatedAt)
	}
}

func TestNewProject_InvalidName(t *testing.T) {
	now := time.Now()

	_, err := NewProject("proj-1", "", "説明", now)
	if err == nil {
		t.Fatalf("expected error for empty name, got nil")
	}
}
