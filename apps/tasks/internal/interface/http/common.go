package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// OptionalString は JSON で null と未指定を区別するための型。
// - 未指定: nil
// - null: &OptionalString{Value: nil, IsSet: true}
// - 値あり: &OptionalString{Value: &str, IsSet: true}
type OptionalString struct {
	Value *string
	IsSet bool
}

// UnmarshalJSON は JSON を Unmarshal し、null と未指定を区別する。
func (o *OptionalString) UnmarshalJSON(data []byte) error {
	o.IsSet = true
	if string(data) == "null" {
		o.Value = nil
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	o.Value = &s
	return nil
}

// nullableString は JSON で null を受け取ることができる文字列型。
// UnmarshalJSON で null と未指定を区別するため、null の場合は存在フラグを立てる。
type nullableString struct {
	value   *string
	isNull  bool
	present bool
}

func (ns *nullableString) UnmarshalJSON(data []byte) error {
	ns.present = true
	var s *string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s == nil {
		ns.isNull = true
		ns.value = nil
	} else {
		ns.isNull = false
		ns.value = s
	}
	return nil
}

// toPtr は将来の拡張用に残しているが、現在は未使用
// nolint:unused
func (ns *nullableString) toPtr() *string {
	if !ns.present {
		return nil // 未指定
	}
	if ns.isNull {
		empty := ""
		return &empty // null の場合は空文字列を返す
	}
	return ns.value // 文字列が指定された場合
}

// taskResponse はタスクのレスポンス用構造体。
type taskResponse struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"projectId"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority"`
	AssigneeID  *string    `json:"assigneeId"`
	DueDate     *time.Time `json:"dueDate"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type errorResponse struct {
	Error  string `json:"error"`
	Detail string `json:"detail"`
}

// writeErrorResponse はエラーレスポンスを書き込む。
func writeErrorResponse(w http.ResponseWriter, statusCode int, errorMsg, detail string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	resp := errorResponse{
		Error:  errorMsg,
		Detail: detail,
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// isValidUUID は文字列が有効な UUID 形式かどうかをチェックする。
func isValidUUID(s string) bool {
	// UUID 形式: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx (36文字)
	if len(s) != 36 {
		return false
	}
	parts := strings.Split(s, "-")
	if len(parts) != 5 {
		return false
	}
	expectedLengths := []int{8, 4, 4, 4, 12}
	for i, part := range parts {
		if len(part) != expectedLengths[i] {
			return false
		}
		for _, r := range part {
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
				return false
			}
		}
	}
	return true
}
