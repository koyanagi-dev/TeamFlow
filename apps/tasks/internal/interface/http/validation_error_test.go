package http

import "testing"

func TestGetMessageForFieldAndCode(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		code     string
		expected string
	}{
		{
			name:     "status INVALID_ENUM",
			field:    "status",
			code:     "INVALID_ENUM",
			expected: "status は 'todo','doing','in_progress','done' のいずれかをカンマ区切りで指定してください（例: status=todo,in_progress）。",
		},
		{
			name:     "priority INVALID_ENUM",
			field:    "priority",
			code:     "INVALID_ENUM",
			expected: "priority は 'high','medium','low' のいずれかをカンマ区切りで指定してください（例: priority=high,medium）。",
		},
		{
			name:     "dueDateFrom INVALID_FORMAT",
			field:    "dueDateFrom",
			code:     "INVALID_FORMAT",
			expected: "dueDateFrom は YYYY-MM-DD 形式で指定してください（例: dueDateFrom=2026-01-10）。",
		},
		{
			name:     "dueDateTo INVALID_FORMAT",
			field:    "dueDateTo",
			code:     "INVALID_FORMAT",
			expected: "dueDateTo は YYYY-MM-DD 形式で指定してください（例: dueDateTo=2026-01-10）。",
		},
		{
			name:     "sort INVALID_ENUM",
			field:    "sort",
			code:     "INVALID_ENUM",
			expected: "sort は 'sortOrder','createdAt','updatedAt','dueDate','priority' のみ指定できます（例: sort=-priority,createdAt）。",
		},
		{
			name:     "unknown field fallback",
			field:    "unknown",
			code:     "UNKNOWN",
			expected: "クエリパラメータが不正です。入力内容を確認してください。",
		},
		{
			name:     "status with wrong code fallback",
			field:    "status",
			code:     "INVALID_FORMAT",
			expected: "クエリパラメータが不正です。入力内容を確認してください。",
		},
		{
			name:     "arbitrary field and code fallback",
			field:    "xxx",
			code:     "yyy",
			expected: "クエリパラメータが不正です。入力内容を確認してください。",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getMessageForFieldAndCode(tt.field, tt.code)
			if got != tt.expected {
				t.Errorf("getMessageForFieldAndCode(%q, %q) = %q, want %q", tt.field, tt.code, got, tt.expected)
			}
		})
	}
}

func TestParseLimit(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    int
		expectError bool
	}{
		{
			name:        "empty string returns 0",
			input:       "",
			expected:    0,
			expectError: false,
		},
		{
			name:        "valid integer",
			input:       "50",
			expected:    50,
			expectError: false,
		},
		{
			name:        "invalid string returns error",
			input:       "abc",
			expected:    0,
			expectError: true,
		},
		{
			name:        "float string returns error",
			input:       "1.5",
			expected:    0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLimit(tt.input)
			if tt.expectError {
				if err == nil {
					t.Errorf("ParseLimit(%q) expected error, got nil", tt.input)
				}
				// InvalidLimitError であることを確認
				var ile *InvalidLimitError
				if err != nil {
					if _, ok := err.(*InvalidLimitError); !ok {
						t.Errorf("ParseLimit(%q) expected *InvalidLimitError, got %T", tt.input, err)
					} else {
						ile = err.(*InvalidLimitError)
						if ile.RejectedValue != tt.input {
							t.Errorf("InvalidLimitError.RejectedValue = %q, want %q", ile.RejectedValue, tt.input)
						}
					}
				}
			} else {
				if err != nil {
					t.Errorf("ParseLimit(%q) unexpected error: %v", tt.input, err)
				}
				if got != tt.expected {
					t.Errorf("ParseLimit(%q) = %d, want %d", tt.input, got, tt.expected)
				}
			}
		})
	}
}
