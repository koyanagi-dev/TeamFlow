package task

import "fmt"

// ValidationError は検証エラーを表す typed error。
// HTTP 層で errors.As を使って field/code/rejectedValue を取り出せる。
type ValidationError struct {
	Field         string  // status, priority, sort, dueDateFrom, dueDateTo
	Code          string  // INVALID_ENUM, INVALID_FORMAT
	RejectedValue *string // 不正だった値（nil の場合もある）
	cause         error   // 元のエラー（Unwrap 用）
}

// Error は error インターフェースを満たす。
func (e *ValidationError) Error() string {
	if e.RejectedValue != nil {
		return fmt.Sprintf("%s: %s (rejected: %s)", e.Field, e.Code, *e.RejectedValue)
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Code)
}

// Unwrap は cause を返す（errors.Unwrap 対応）。
func (e *ValidationError) Unwrap() error {
	return e.cause
}

// --- Constructors ---

// NewInvalidEnum は INVALID_ENUM エラーを生成する。
// field: status, priority, sort など
// cause: 元のエラー（nil 可）
// rejected: 不正だった値（nil 可）
func NewInvalidEnum(field string, cause error, rejected *string) *ValidationError {
	return &ValidationError{
		Field:         field,
		Code:          "INVALID_ENUM",
		RejectedValue: rejected,
		cause:         cause,
	}
}

// NewInvalidFormat は INVALID_FORMAT エラーを生成する。
// field: dueDateFrom, dueDateTo など
// cause: 元のエラー（nil 可）
// rejected: 不正だった値（nil 可）
func NewInvalidFormat(field string, cause error, rejected *string) *ValidationError {
	return &ValidationError{
		Field:         field,
		Code:          "INVALID_FORMAT",
		RejectedValue: rejected,
		cause:         cause,
	}
}
