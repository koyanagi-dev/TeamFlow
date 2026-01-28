package task

type Patch[T any] struct {
	IsSet  bool // 未指定=false
	IsNull bool // null=true
	Value  T
}

func Unset[T any]() Patch[T]      { return Patch[T]{IsSet: false} }
func Null[T any]() Patch[T]       { return Patch[T]{IsSet: true, IsNull: true} }
func Set[T any](v T) Patch[T]     { return Patch[T]{IsSet: true, Value: v} }
func (p Patch[T]) HasValue() bool { return p.IsSet && !p.IsNull }
