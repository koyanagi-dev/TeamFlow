package task

import (
	"testing"
	"time"
)

func TestNewTaskQuery_Default(t *testing.T) {
	q, err := NewTaskQuery()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if q.Limit != 200 {
		t.Errorf("expected default limit=200, got=%d", q.Limit)
	}

	if len(q.Statuses) != 0 {
		t.Errorf("expected empty statuses, got=%v", q.Statuses)
	}
}

func TestNewTaskQuery_StatusFilter(t *testing.T) {
	tests := []struct {
		name      string
		statusStr string
		want      []TaskStatus
		wantErr   bool
	}{
		{
			name:      "single status",
			statusStr: "todo",
			want:      []TaskStatus{StatusTodo},
			wantErr:   false,
		},
		{
			name:      "multiple statuses",
			statusStr: "todo,in_progress,done",
			want:      []TaskStatus{StatusTodo, StatusInProgress, StatusDone},
			wantErr:   false,
		},
		{
			name:      "doing normalized to in_progress",
			statusStr: "doing",
			want:      []TaskStatus{StatusInProgress},
			wantErr:   false,
		},
		{
			name:      "mixed with doing",
			statusStr: "todo,doing,done",
			want:      []TaskStatus{StatusTodo, StatusInProgress, StatusDone},
			wantErr:   false,
		},
		{
			name:      "invalid status",
			statusStr: "invalid",
			want:      nil,
			wantErr:   true,
		},
		{
			name:      "empty string",
			statusStr: "",
			want:      nil,
			wantErr:   false,
		},
		{
			name:      "duplicate statuses",
			statusStr: "todo,todo,in_progress",
			want:      []TaskStatus{StatusTodo, StatusInProgress},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := NewTaskQuery(WithStatusFilter(tt.statusStr))
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTaskQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if len(q.Statuses) != len(tt.want) {
				t.Errorf("Statuses length = %d, want %d", len(q.Statuses), len(tt.want))
				return
			}

			for i, want := range tt.want {
				if q.Statuses[i] != want {
					t.Errorf("Statuses[%d] = %v, want %v", i, q.Statuses[i], want)
				}
			}
		})
	}
}

func TestNewTaskQuery_PriorityFilter(t *testing.T) {
	tests := []struct {
		name        string
		priorityStr string
		want        []TaskPriority
		wantErr     bool
	}{
		{
			name:        "single priority",
			priorityStr: "high",
			want:        []TaskPriority{PriorityHigh},
			wantErr:     false,
		},
		{
			name:        "multiple priorities",
			priorityStr: "high,medium,low",
			want:        []TaskPriority{PriorityHigh, PriorityMedium, PriorityLow},
			wantErr:     false,
		},
		{
			name:        "invalid priority",
			priorityStr: "invalid",
			want:        nil,
			wantErr:     true,
		},
		{
			name:        "empty string",
			priorityStr: "",
			want:        nil,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := NewTaskQuery(WithPriorityFilter(tt.priorityStr))
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTaskQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if len(q.Priorities) != len(tt.want) {
				t.Errorf("Priorities length = %d, want %d", len(q.Priorities), len(tt.want))
				return
			}

			for i, want := range tt.want {
				if q.Priorities[i] != want {
					t.Errorf("Priorities[%d] = %v, want %v", i, q.Priorities[i], want)
				}
			}
		})
	}
}

func TestNewTaskQuery_Limit(t *testing.T) {
	tests := []struct {
		name    string
		input   int
		want    int
		wantErr bool
	}{
		{
			name:    "valid limit",
			input:   100,
			want:    100,
			wantErr: false,
		},
		{
			name:    "max limit",
			input:   200,
			want:    200,
			wantErr: false,
		},
		{
			name:    "over max limit",
			input:   300,
			want:    200,
			wantErr: false,
		},
		{
			name:    "under min limit",
			input:   0,
			want:    200,
			wantErr: false,
		},
		{
			name:    "negative limit",
			input:   -10,
			want:    200,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := NewTaskQuery(WithLimit(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTaskQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if q.Limit != tt.want {
				t.Errorf("Limit = %d, want %d", q.Limit, tt.want)
			}
		})
	}
}

func TestNewTaskQuery_Sort(t *testing.T) {
	tests := []struct {
		name    string
		sortStr string
		want    []SortOrder
		wantErr bool
	}{
		{
			name:    "single ASC",
			sortStr: "createdAt",
			want: []SortOrder{
				{Key: "createdAt", Direction: SortDirectionASC},
			},
			wantErr: false,
		},
		{
			name:    "single DESC",
			sortStr: "-priority",
			want: []SortOrder{
				{Key: "priority", Direction: SortDirectionDESC},
			},
			wantErr: false,
		},
		{
			name:    "multiple sort",
			sortStr: "-priority,createdAt",
			want: []SortOrder{
				{Key: "priority", Direction: SortDirectionDESC},
				{Key: "createdAt", Direction: SortDirectionASC},
			},
			wantErr: false,
		},
		{
			name:    "invalid sort key",
			sortStr: "invalidKey",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "empty string",
			sortStr: "",
			want:    nil,
			wantErr: false,
		},
		{
			name:    "all valid keys",
			sortStr: "sortOrder,createdAt,updatedAt,dueDate,priority",
			want: []SortOrder{
				{Key: "sortOrder", Direction: SortDirectionASC},
				{Key: "createdAt", Direction: SortDirectionASC},
				{Key: "updatedAt", Direction: SortDirectionASC},
				{Key: "dueDate", Direction: SortDirectionASC},
				{Key: "priority", Direction: SortDirectionASC},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := NewTaskQuery(WithSort(tt.sortStr))
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTaskQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if len(q.SortOrders) != len(tt.want) {
				t.Errorf("SortOrders length = %d, want %d", len(q.SortOrders), len(tt.want))
				return
			}

			for i, want := range tt.want {
				if q.SortOrders[i].Key != want.Key {
					t.Errorf("SortOrders[%d].Key = %v, want %v", i, q.SortOrders[i].Key, want.Key)
				}
				if q.SortOrders[i].Direction != want.Direction {
					t.Errorf("SortOrders[%d].Direction = %v, want %v", i, q.SortOrders[i].Direction, want.Direction)
				}
			}
		})
	}
}

func TestNewTaskQuery_DueDateRange(t *testing.T) {
	tests := []struct {
		name          string
		dueDateFrom   string
		dueDateTo     string
		wantErr       bool
		validateFunc  func(*testing.T, *TaskQuery)
	}{
		{
			name:        "valid range",
			dueDateFrom: "2024-01-01",
			dueDateTo:   "2024-12-31",
			wantErr:     false,
			validateFunc: func(t *testing.T, q *TaskQuery) {
				if q.DueDateFrom == nil {
					t.Fatal("DueDateFrom should not be nil")
				}
				if q.DueDateTo == nil {
					t.Fatal("DueDateTo should not be nil")
				}
				expectedFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
				expectedTo := time.Date(2024, 12, 31, 23, 59, 59, 999999999, time.UTC)
				if !q.DueDateFrom.Equal(expectedFrom) {
					t.Errorf("DueDateFrom = %v, want %v", q.DueDateFrom, expectedFrom)
				}
				if !q.DueDateTo.Equal(expectedTo) {
					t.Errorf("DueDateTo = %v, want %v", q.DueDateTo, expectedTo)
				}
			},
		},
		{
			name:        "invalid format",
			dueDateFrom: "2024/01/01",
			dueDateTo:   "",
			wantErr:     true,
		},
		{
			name:        "empty strings",
			dueDateFrom: "",
			dueDateTo:   "",
			wantErr:     false,
			validateFunc: func(t *testing.T, q *TaskQuery) {
				if q.DueDateFrom != nil {
					t.Error("DueDateFrom should be nil")
				}
				if q.DueDateTo != nil {
					t.Error("DueDateTo should be nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := NewTaskQuery(WithDueDateRangeFilter(tt.dueDateFrom, tt.dueDateTo))
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTaskQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if tt.validateFunc != nil {
				tt.validateFunc(t, q)
			}
		})
	}
}

