package httpjson

import "encoding/json"

type Nullable[T any] struct {
    Set   bool // JSONにフィールドが存在したか（未指定=false）
    Valid bool // nullでないか（値あり=true、null=false）
    Val   T
}

func (n *Nullable[T]) UnmarshalJSON(b []byte) error {
    n.Set = true
    if string(b) == "null" {
        n.Valid = false
        var zero T
        n.Val = zero
        return nil
    }
    n.Valid = true
    return json.Unmarshal(b, &n.Val)
}
