package handlers

import (
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

// parseTestTemplates loads the real templates/*.html files the exact same
// way internal/server.Routes does, so this test exercises the templates
// that actually ship with the app instead of a hand-crafted fixture. It
// registers the same FuncMap (dict, deref) too — templates/pantry.html
// depends on both.
func parseTestTemplates(t *testing.T) map[string]*template.Template {
	t.Helper()

	tmpl, err := template.New("").Funcs(TemplateFuncs()).ParseGlob("../../../templates/*.html")
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}

	cache := make(map[string]*template.Template)
	for _, tt := range tmpl.Templates() {
		cache[tt.Name()] = tt
	}
	return cache
}

// TestHomeRedirectsToPantry checks that "/" and "/home" (both routed to
// Home, see server.go) redirect to /pantry — the app's actual landing page
// now that the tile dashboard is gone.
func TestHomeRedirectsToPantry(t *testing.T) {
	h := New(nil, slog.New(slog.NewTextHandler(io.Discard, nil)), parseTestTemplates(t), nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.Home(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "/pantry" {
		t.Errorf("expected redirect to /pantry, got %q", got)
	}
}
