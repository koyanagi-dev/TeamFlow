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
//
// 使用例:
//   if errors.Is(err, ErrDueDateFromAfterTo) {
//       // field=dueDateFrom, code=CONSTRAINT_VIOLATION として処理
//   }

// Query validation errors
var (
	// ErrDueDateFromAfterTo は dueDateFrom > dueDateTo の場合のエラー。
	// HTTP 層: field=dueDateFrom, code=CONSTRAINT_VIOLATION
	ErrDueDateFromAfterTo = errors.New("dueDateFrom must not be after dueDateTo")

	// ErrLimitOutOfRange は limit が 1-200 の範囲外の場合のエラー。
	// HTTP 層: field=limit, code=INVALID_RANGE
	ErrLimitOutOfRange = errors.New("limit must be between 1 and 200")

	// ErrSortIncompatibleWithCursor は cursor と sort の併用時のエラー。
	// HTTP 層: field=sort, code=INCOMPATIBLE_WITH_CURSOR
	ErrSortIncompatibleWithCursor = errors.New("sort is incompatible with cursor")
)

// Cursor validation errors
var (
	// ErrCursorInvalidFormat は cursor の形式が不正な場合のエラー。
	// HTTP 層: field=cursor, code=INVALID_FORMAT
	ErrCursorInvalidFormat = errors.New("invalid cursor format")

	// ErrCursorInvalidSignature は cursor の署名が不正な場合のエラー。
	// HTTP 層: field=cursor, code=INVALID_SIGNATURE
	ErrCursorInvalidSignature = errors.New("invalid cursor signature")

	// ErrCursorExpired は cursor の有効期限が切れている場合のエラー。
	// HTTP 層: field=cursor, code=EXPIRED
	ErrCursorExpired = errors.New("cursor expired")

	// ErrCursorQueryMismatch は cursor のクエリ条件が一致しない場合のエラー。
	// HTTP 層: field=cursor, code=QUERY_MISMATCH
	ErrCursorQueryMismatch = errors.New("cursor query mismatch")
)
