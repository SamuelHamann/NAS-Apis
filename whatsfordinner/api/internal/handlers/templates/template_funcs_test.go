package handlers

import "testing"

func TestTemplateMapKeysCSV(t *testing.T) {
	tests := []struct {
		name string
		m    map[int64]bool
		want string
	}{
		{"nil map", nil, ""},
		{"empty map", map[int64]bool{}, ""},
		{"single key", map[int64]bool{7: true}, "7"},
		{"sorted output", map[int64]bool{3: true, 1: true, 2: true}, "1,2,3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := templateMapKeysCSV(tt.m); got != tt.want {
				t.Errorf("templateMapKeysCSV(%v) = %q, want %q", tt.m, got, tt.want)
			}
		})
	}
}
