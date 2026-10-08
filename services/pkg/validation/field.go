package validation

import (
	"bytes"
	"encoding/json"
	"reflect"
)

// Field is a JSON property that tells apart "absent", "null" and a value —
// zod's .nullable().optional(). Partial updates need all three: absent
// keeps the stored value, null clears it.
type Field[T any] struct {
	Set   bool // present in the body (null or a value)
	Valid bool // a non-null value
	Value T
}

func (f *Field[T]) UnmarshalJSON(b []byte) error {
	f.Set = true
	if bytes.Equal(bytes.TrimSpace(b), []byte("null")) {
		f.Valid = false
		return nil
	}
	if err := json.Unmarshal(b, &f.Value); err != nil {
		return err
	}
	f.Valid = true
	return nil
}

// Ptr returns the value, or nil when null or absent.
func (f Field[T]) Ptr() *T {
	if !f.Valid {
		return nil
	}
	v := f.Value
	return &v
}

// validationValue is what `binding` tags see: a nil pointer (so omitempty
// skips it) unless a value was given — then a pointer to it, so that zero
// values like 0 or "" are still validated rather than treated as empty.
func (f Field[T]) validationValue() any {
	if !f.Valid {
		return (*T)(nil)
	}
	v := f.Value
	return &v
}

type fieldValuer interface{ validationValue() any }

func init() {
	validate.RegisterCustomTypeFunc(func(v reflect.Value) any {
		if f, ok := v.Interface().(fieldValuer); ok {
			return f.validationValue()
		}
		return nil
	}, Field[string]{}, Field[Trimmed]{}, Field[int]{}, Field[int64]{}, Field[float64]{}, Field[bool]{}, Field[[]string]{})
}
