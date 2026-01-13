package http

import (
	"errors"
	"strconv"
	"strings"
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

// toValidationIssue: domain.NewTaskQuery 由来のエラーを ValidationIssue に変換する
// - まずは err.Error() の prefix/contains で判定（要求仕様通り）
// - rejectedValue は取り出せるものだけ詰める
func toValidationIssue(err error) ValidationIssue {
	// 念のため nil ガード（呼び出し側が err != nil を保証する想定）
	if err == nil {
		return ValidationIssue{
			Location: "query",
			Field:    "unknown",
			Code:     "UNKNOWN",
			Message:  "Unknown validation error",
		}
	}

	msg := err.Error()

	// Handler 側パースエラー（例: limit の Atoi 失敗）をここで扱いたい場合は、
	// 呼び出し側で sentinel error をラップして渡すのがおすすめ。
	// 例: err = fmt.Errorf("%w: %s", errInvalidLimitFormat, rawLimit)
	if errors.Is(err, errInvalidLimitFormat) {
		rejected := extractRejectedValue(msg) // fallback: エラーメッセージ末尾など
		return ValidationIssue{
			Location:      "query",
			Field:         "limit",
			Code:          "INVALID_FORMAT",
			Message:       "limit は整数で指定してください（例: limit=50）。",
			RejectedValue: rejected,
		}
	}

	switch {
	case strings.HasPrefix(msg, "invalid status in filter:"):
		rejected := extractAfterColon(msg)
		return ValidationIssue{
			Location:      "query",
			Field:         "status",
			Code:          "INVALID_ENUM",
			Message:       "status は 'todo','doing','in_progress','done' のいずれかをカンマ区切りで指定してください（例: status=todo,in_progress）。",
			RejectedValue: rejected,
		}

	case strings.HasPrefix(msg, "invalid priority in filter:"):
		rejected := extractAfterColon(msg)
		return ValidationIssue{
			Location:      "query",
			Field:         "priority",
			Code:          "INVALID_ENUM",
			Message:       "priority は 'high','medium','low' のいずれかをカンマ区切りで指定してください（例: priority=high,medium）。",
			RejectedValue: rejected,
		}

	case strings.HasPrefix(msg, "invalid dueDateFrom format"):
		rejected := extractAfterColon(msg)
		return ValidationIssue{
			Location:      "query",
			Field:         "dueDateFrom",
			Code:          "INVALID_FORMAT",
			Message:       "dueDateFrom は YYYY-MM-DD 形式で指定してください（例: dueDateFrom=2026-01-10）。",
			RejectedValue: rejected,
		}

	case strings.HasPrefix(msg, "invalid dueDateTo format"):
		rejected := extractAfterColon(msg)
		return ValidationIssue{
			Location:      "query",
			Field:         "dueDateTo",
			Code:          "INVALID_FORMAT",
			Message:       "dueDateTo は YYYY-MM-DD 形式で指定してください（例: dueDateTo=2026-01-10）。",
			RejectedValue: rejected,
		}

	case strings.Contains(msg, "dueDateFrom must not be after dueDateTo"):
		return ValidationIssue{
			Location: "query",
			Field:    "dueDateFrom",
			Code:     "CONSTRAINT_VIOLATION",
			Message:  "dueDateFrom は dueDateTo 以下の日付にしてください（例: dueDateFrom=2026-01-01&dueDateTo=2026-01-10）。",
		}

	case strings.HasPrefix(msg, "invalid sort key:"):
		rejected := extractSortKey(msg)
		return ValidationIssue{
			Location:      "query",
			Field:         "sort",
			Code:          "INVALID_ENUM",
			Message:       "sort は 'sortOrder','createdAt','updatedAt','dueDate','priority' のみ指定できます（例: sort=-priority,createdAt）。",
			RejectedValue: rejected,
		}

	case strings.Contains(msg, "limit must be between 1 and 200"):
		rejected := extractAfterColon(msg)
		return ValidationIssue{
			Location:      "query",
			Field:         "limit",
			Code:          "INVALID_RANGE",
			Message:       "limit は 1〜200 の整数で指定してください（未指定または 1 未満は 200 に正規化されます）。",
			RejectedValue: rejected,
		}

	case strings.Contains(msg, "sort is incompatible with cursor"):
		rejected := extractRejectedValue(msg)
		return ValidationIssue{
			Location:      "query",
			Field:         "sort",
			Code:          "INCOMPATIBLE_WITH_CURSOR",
			Message:       "cursor を使用する場合、sort は指定できません。",
			RejectedValue: rejected,
		}

	case strings.Contains(msg, "invalid cursor format"):
		rejected := extractRejectedValue(msg)
		return ValidationIssue{
			Location:      "query",
			Field:         "cursor",
			Code:          "INVALID_FORMAT",
			Message:       "cursor の形式が不正です。",
			RejectedValue: rejected,
		}

	case strings.Contains(msg, "invalid cursor signature"):
		rejected := extractRejectedValue(msg)
		return ValidationIssue{
			Location:      "query",
			Field:         "cursor",
			Code:          "INVALID_SIGNATURE",
			Message:       "cursor の署名が不正です。",
			RejectedValue: rejected,
		}

	case strings.Contains(msg, "cursor expired"):
		rejected := extractRejectedValue(msg)
		return ValidationIssue{
			Location:      "query",
			Field:         "cursor",
			Code:          "EXPIRED",
			Message:       "cursor の有効期限が切れています。",
			RejectedValue: rejected,
		}

	case strings.Contains(msg, "cursor query mismatch"):
		rejected := extractRejectedValue(msg)
		return ValidationIssue{
			Location:      "query",
			Field:         "cursor",
			Code:          "QUERY_MISMATCH",
			Message:       "cursor のクエリ条件が一致しません。フィルタ等が変更された可能性があります。",
			RejectedValue: rejected,
		}

	default:
		// 想定外でも 400 の形式は崩さない（ログには msg を残すのが推奨）
		return ValidationIssue{
			Location: "query",
			Field:    "unknown",
			Code:     "UNKNOWN",
			Message:  "クエリパラメータが不正です。入力内容を確認してください。",
		}
	}
}

// --- Sentinel errors（handler側のパースエラーなどを識別したい時用） ---

var errInvalidLimitFormat = errors.New("invalid limit format")

// ParseLimit: handler側で limit の parse をするならこういう小関数にまとめると便利。
// - 失敗したら sentinel error でラップして toValidationIssue が判定できるようにする。
func ParseLimit(raw string) (int, error) {
	if raw == "" {
		// 未指定は上位で default を入れる運用にする（例: 200）
		return 0, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		// rejectedValue を拾いたいなら、エラーメッセージに raw を含める
		return 0, wrapInvalidLimitFormat(raw)
	}
	return v, nil
}

func wrapInvalidLimitFormat(raw string) error {
	// err.Error() に raw を含める（extractRejectedValue で拾える）
	return errors.Join(errInvalidLimitFormat, errors.New("rejected="+raw))
}

// --- rejectedValue 抽出ユーティリティ ---

func extractAfterColon(s string) *string {
	// "xxx: yyy" の yyy を取る
	i := strings.Index(s, ":")
	if i < 0 || i+1 >= len(s) {
		return nil
	}
	v := strings.TrimSpace(s[i+1:])
	if v == "" {
		return nil
	}
	return &v
}

func extractSortKey(s string) *string {
	// "invalid sort key: X (valid keys: ...)" の X を取る
	// まず ":" 以降を取り、"(" より前を切る
	after := extractAfterColon(s)
	if after == nil {
		return nil
	}
	v := *after
	if j := strings.Index(v, "("); j >= 0 {
		v = strings.TrimSpace(v[:j])
	}
	if v == "" {
		return nil
	}
	return &v
}

func extractRejectedValue(s string) *string {
	// Joinで "rejected=xxx" を混ぜたときに拾う簡易版
	// 例: "... rejected=abc"
	idx := strings.LastIndex(s, "rejected=")
	if idx < 0 {
		return nil
	}
	v := strings.TrimSpace(s[idx+len("rejected="):])
	if v == "" {
		return nil
	}
	return &v
}
