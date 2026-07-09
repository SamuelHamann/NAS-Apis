package handlers

import (
	"errors"
	"fmt"
	"html/template"
	"reflect"
)

// TemplateFuncs returns the FuncMap shared by every html/template in the
// app. It's defined in this package (not internal/server) so both the
// production template loader and the test template loader can call it
// without dragging in the server package (which would be a cycle).
//
// Currently registered:
//   - dict "k" v "k2" v2 ...   build ad-hoc maps to pass multiple values
//     to sub-templates. templates/pantry.html uses this to send both the
//     current item and the whole page data to the pantry_item / pantry_form
//     partials.
//   - deref *T                 dereference a pointer field (*string,
//     *int64, ...) so templates can compare/print it without needing extra
//     Go-side helpers on every model.
func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"dict":  templateDict,
		"deref": templateDeref,
	}
}

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
// itself nil (either a typed nil pointer or an untyped nil interface).
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
