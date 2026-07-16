package handlers

import (
	"errors"
	"fmt"
	"html/template"
	"reflect"
	"sort"
	"strconv"
	"strings"
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
//   - hasString []string s      report whether s appears in the slice.
//     templates/ingredients.html uses this to pre-check a tag's checkbox
//     when editing an ingredient (comparing against its []string TagNames).
//   - mapKeysCSV map[int64]bool  render a map's keys as a sorted,
//     comma-separated string (e.g. "1,3,7"). templates/recipes.html uses
//     this to emit each recipe row's data-filter-collections attribute from
//     its per-recipe membership map, for "checkbox-filter-script" to read.
func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"dict":       templateDict,
		"deref":      templateDeref,
		"hasString":  templateHasString,
		"mapKeysCSV": templateMapKeysCSV,
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

// templateHasString reports whether s appears in xs.
func templateHasString(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

// templateMapKeysCSV renders m's keys (a recipe's collection-membership set,
// typically) as a sorted, comma-separated string, e.g. "1,3,7", or "" for an
// empty/nil map. Sorted so the output — and the DOM attribute using it — is
// deterministic across renders.
func templateMapKeysCSV(m map[int64]bool) string {
	keys := make([]int64, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = strconv.FormatInt(k, 10)
	}
	return strings.Join(parts, ",")
}
