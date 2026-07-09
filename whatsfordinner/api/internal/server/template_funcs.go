package server

import (
	"errors"
	"fmt"
	"reflect"
)

// templateDict lets templates build ad-hoc maps to pass multiple values to a
// sub-template, e.g. `{{template "row" (dict "Item" . "Page" $)}}`. Keys must
// be strings; there must be an even number of arguments.
func templateDict(pairs ...any) (map[string]any, error) {
	if len(pairs)%2 != 0 {
		return nil, errors.New("dict: odd number of arguments")
	}
	out := make(map[string]any, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		key, ok := pairs[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict: key at index %d is not a string (got %T)", i, pairs[i])
		}
		out[key] = pairs[i+1]
	}
	return out, nil
}

// templateDeref returns the element the pointer v points to, or nil if v is
// itself nil (either a typed nil pointer or an untyped nil interface). This
// lets templates work with the nullable pointer fields on our model structs
// (`*string`, `*int64`, ...) without having to pre-flatten them in Go.
func templateDeref(v any) any {
	if v == nil {
		return nil
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		return v
	}
	if rv.IsNil() {
		return nil
	}
	return rv.Elem().Interface()
}
