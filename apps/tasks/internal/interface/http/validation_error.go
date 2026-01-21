package http

import (
	"errors"
	"log"
	"strconv"

	domain "teamflow-tasks/internal/domain/task"
)

// ValidationIssue: OpenAPIの schema（ValidationIssue）と対応する構造体
type ValidationIssue struct {
	Location      string  `json:"location"`                // "query" | "path" | "body"
	Field         string  `json:"field"`                   // 例: status, priority, sort, dueDateFrom
	Code          string  `json:"code"`                    // 例: INVALID_ENUM
	Message       string  `json:"message"`                 // フロントが直すべき内容がわかる文言
	RejectedValue *string `json:"rejectedValue,omitempty"` // 出せる場合のみ
}

type ErrorResponse struct {
	Error   string        `json:"error"`
	Message string        `json:"message"`
	Details *ErrorDetails `json:"details,omitempty"`
}

type ErrorDetails struct {
	Issues []ValidationIssue `json:"issues,omitempty"`
}

// NewValidationErrorResponse: 400用の統一レスポンス生成
func NewValidationErrorResponse(issues ...ValidationIssue) ErrorResponse {
	resp := ErrorResponse{
		Error:   "VALIDATION_ERROR",
		Message: "Invalid query parameters",
	}
	if len(issues) > 0 {
		resp.Details = &ErrorDetails{Issues: issues}
	}
	return resp
}

// toValidationIssue: domain のエラーを ValidationIssue に変換する。
// errors.Is / errors.As を使用し、文字列判定は行わない。
func toValidationIssue(err error) ValidationIssue {
	// nil ガード
	if err == nil {
		return ValidationIssue{
			Location: "query",
			Field:    "unknown",
			Code:     "UNKNOWN",
			Message:  "Unknown validation error",
		}
	}

	// 1. Handler 側 typed error: InvalidLimitError
	var ile *InvalidLimitError
	if errors.As(err, &ile) {
		rejected := ile.RejectedValue
		return ValidationIssue{
			Location:      "query",
			Field:         "limit",
			Code:          "INVALID_FORMAT",
			Message:       "limit は整数で指定してください（例: limit=50）。",
			RejectedValue: &rejected,
		}
	}

	// 2. Domain typed error: ValidationError (INVALID_ENUM / INVALID_FORMAT)
	var ve *domain.ValidationError
	if errors.As(err, &ve) {
		return ValidationIssue{
			Location:      "query",
			Field:         ve.Field,
			Code:          ve.Code,
			Message:       getMessageForFieldAndCode(ve.Field, ve.Code),
			RejectedValue: ve.RejectedValue,
		}
	}

	// 3. Domain sentinel errors
	switch {
	case errors.Is(err, domain.ErrDueDateFromAfterTo):
		return ValidationIssue{
			Location: "query",
			Field:    "dueDateFrom",
			Code:     "CONSTRAINT_VIOLATION",
			Message:  "dueDateFrom は dueDateTo 以下の日付にしてください（例: dueDateFrom=2026-01-01&dueDateTo=2026-01-10）。",
		}

	case errors.Is(err, domain.ErrLimitOutOfRange):
		return ValidationIssue{
			Location: "query",
			Field:    "limit",
			Code:     "INVALID_RANGE",
			Message:  "limit は 1〜200 の整数で指定してください（未指定または 1 未満は 200 に正規化されます）。",
		}

	case errors.Is(err, domain.ErrSortIncompatibleWithCursor):
		return ValidationIssue{
			Location: "query",
			Field:    "sort",
			Code:     "INCOMPATIBLE_WITH_CURSOR",
			Message:  "cursor を使用する場合、sort は指定できません。",
		}

	case errors.Is(err, domain.ErrCursorInvalidFormat):
		return ValidationIssue{
			Location: "query",
			Field:    "cursor",
			Code:     "INVALID_FORMAT",
			Message:  "cursor の形式が不正です。",
		}

	case errors.Is(err, domain.ErrCursorInvalidSignature):
		return ValidationIssue{
			Location: "query",
			Field:    "cursor",
			Code:     "INVALID_SIGNATURE",
			Message:  "cursor の署名が不正です。",
		}

	case errors.Is(err, domain.ErrCursorExpired):
		return ValidationIssue{
			Location: "query",
			Field:    "cursor",
			Code:     "EXPIRED",
			Message:  "cursor の有効期限が切れています。",
		}

	case errors.Is(err, domain.ErrCursorQueryMismatch):
		return ValidationIssue{
			Location: "query",
			Field:    "cursor",
			Code:     "QUERY_MISMATCH",
			Message:  "cursor のクエリ条件が一致しません。フィルタ等が変更された可能性があります。",
		}
	}

	// fallback: 想定外でも 400 の形式は崩さない（ログ出力してデバッグ可能に）
	log.Printf("WARNING: unmapped validation error: %T %v", err, err)
	return ValidationIssue{
		Location: "query",
		Field:    "unknown",
		Code:     "UNKNOWN",
		Message:  "クエリパラメータが不正です。入力内容を確認してください。",
	}
}

// getMessageForFieldAndCode は field と code の組み合わせから固定メッセージを返す。
// 現行の message と完全一致を保証する。
func getMessageForFieldAndCode(field, code string) string {
	// field + code による固定 mapping（互換維持）
	switch field {
	case "status":
		if code == "INVALID_ENUM" {
			return "status は 'todo','doing','in_progress','done' のいずれかをカンマ区切りで指定してください（例: status=todo,in_progress）。"
		}
	case "priority":
		if code == "INVALID_ENUM" {
			return "priority は 'high','medium','low' のいずれかをカンマ区切りで指定してください（例: priority=high,medium）。"
		}
	case "dueDateFrom":
		if code == "INVALID_FORMAT" {
			return "dueDateFrom は YYYY-MM-DD 形式で指定してください（例: dueDateFrom=2026-01-10）。"
		}
	case "dueDateTo":
		if code == "INVALID_FORMAT" {
			return "dueDateTo は YYYY-MM-DD 形式で指定してください（例: dueDateTo=2026-01-10）。"
		}
	case "sort":
		if code == "INVALID_ENUM" {
			return "sort は 'sortOrder','createdAt','updatedAt','dueDate','priority' のみ指定できます（例: sort=-priority,createdAt）。"
		}
	}

	// fallback
	return "クエリパラメータが不正です。入力内容を確認してください。"
}

// --- InvalidLimitError: handler側の limit パースエラー用 typed error ---

// InvalidLimitError は limit パース失敗時のエラー。
// 文字列パースに頼らず、構造化された rejectedValue を持つ。
type InvalidLimitError struct {
	RejectedValue string // パースに失敗した元の値
	cause         error  // 元のエラー（strconv.Atoi の戻り値など）
}

// Error は error インターフェースを満たす。
func (e *InvalidLimitError) Error() string {
	return "invalid limit format: " + e.RejectedValue
}

// Unwrap は cause を返す（errors.Unwrap 対応）。
func (e *InvalidLimitError) Unwrap() error {
	return e.cause
}

// ParseLimit: handler側で limit の parse をする。
// 失敗したら InvalidLimitError を返し、toValidationIssue で errors.As で判定できる。
func ParseLimit(raw string) (int, error) {
	if raw == "" {
		// 未指定は上位で default を入れる運用にする（例: 200）
		return 0, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, &InvalidLimitError{RejectedValue: raw, cause: err}
	}
	return v, nil
}
