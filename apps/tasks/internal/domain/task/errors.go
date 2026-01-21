package task

import (
	"errors"
	"fmt"
)

// ErrInvalidPatch は patch 適用時のエラーを生成する。
func ErrInvalidPatch(reason string) error {
	return fmt.Errorf("invalid patch: %s", reason)
}

// --- Sentinel Errors ---
// これらは errors.Is で判定可能。HTTP 層で ValidationIssue に変換される。

// Query validation errors
var (
	// ErrDueDateFromAfterTo は dueDateFrom > dueDateTo の場合のエラー。
	ErrDueDateFromAfterTo = errors.New("dueDateFrom must not be after dueDateTo")

	// ErrLimitOutOfRange は limit が 1-200 の範囲外の場合のエラー。
	ErrLimitOutOfRange = errors.New("limit must be between 1 and 200")

	// ErrSortIncompatibleWithCursor は cursor と sort の併用時のエラー。
	ErrSortIncompatibleWithCursor = errors.New("sort is incompatible with cursor")
)

// Cursor validation errors
var (
	// ErrCursorInvalidFormat は cursor の形式が不正な場合のエラー。
	ErrCursorInvalidFormat = errors.New("invalid cursor format")

	// ErrCursorInvalidSignature は cursor の署名が不正な場合のエラー。
	ErrCursorInvalidSignature = errors.New("invalid cursor signature")

	// ErrCursorExpired は cursor の有効期限が切れている場合のエラー。
	ErrCursorExpired = errors.New("cursor expired")

	// ErrCursorQueryMismatch は cursor のクエリ条件が一致しない場合のエラー。
	ErrCursorQueryMismatch = errors.New("cursor query mismatch")
)
