package handlers

import (
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestHomeRendersSuccessfully(t *testing.T) {
	h := New(nil, slog.New(slog.NewTextHandler(io.Discard, nil)), parseTestTemplates(t))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.Home(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	for _, want := range []string{
		"What's for Dinner", // navbar brand
		"Sign in",           // signed-out navbar CTA (no cookie set on this request)
		"Quick recipes",
		"Settings",
		"Recipes",
		"Pantry",
		"Scan receipt",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected rendered home page to contain %q, got:\n%s", want, body)
		}
	}

	if strings.Contains(body, `class="wfd-user"`) {
		t.Errorf("expected signed-out home page not to render the navbar username span, got:\n%s", body)
	}
	if strings.Contains(body, `id="settings-menu"`) {
		t.Errorf("expected signed-out home page not to render the settings dropdown, got:\n%s", body)
	}
}

// TestHomeTemplateRendersSignedInState executes templates/home.html directly
// (bypassing the Home handler, which needs a live database to resolve the
// session cookie) to check the signed-in branch of the navbar and greeting.
func TestHomeTemplateRendersSignedInState(t *testing.T) {
	ts, ok := parseTestTemplates(t)["home.html"]
	if !ok {
		t.Fatal(`template "home.html" not found`)
	}

	data := HomeData{PageData: PageData{Title: "Home", SignedIn: true, CurrentUser: "Alice"}}

	rec := httptest.NewRecorder()
	if err := ts.Execute(rec, data); err != nil {
		t.Fatalf("execute home.html: %v", err)
	}

	body := rec.Body.String()
	for _, want := range []string{
		"Hey Alice 👋",
		`class="wfd-user"`,
		`id="settings-menu"`,
		`href="#settings-menu"`, // Settings tile jumps to the navbar menu once signed in
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected rendered home page to contain %q, got:\n%s", want, body)
		}
	}
	if strings.Contains(body, ">Sign in<") {
		t.Errorf("expected signed-in home page not to render the Sign in button, got:\n%s", body)
	}
}
