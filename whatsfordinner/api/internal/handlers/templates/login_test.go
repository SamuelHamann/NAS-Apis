package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
)

// These tests execute templates/login.html directly against hand-built view
// models rather than going through the Login handler, since the handler
// needs a live database (h.store.ListUsers) that isn't available in this
// package's unit tests. They still catch template wiring mistakes (typos in
// {{template}} names, broken ranges, etc.).

func TestLoginTemplateRendersUsersList(t *testing.T) {
	ts, ok := parseTestTemplates(t)["login.html"]
	if !ok {
		t.Fatal(`template "login.html" not found`)
	}

	data := LoginData{
		PageData: PageData{Title: "Sign in"},
		Users: []models.User{
			{ID: 1, Username: "Alice"},
			{ID: 2, Username: "Bob"},
		},
		Error: "that username is already taken",
	}

	rec := httptest.NewRecorder()
	if err := ts.Execute(rec, data); err != nil {
		t.Fatalf("execute login.html: %v", err)
	}

	body := rec.Body.String()
	for _, want := range []string{
		"Who's cooking?",
		"Alice",
		"Bob",
		`action="/users/1/select"`,
		`action="/users/1/update"`,
		`action="/users/1/delete"`,
		`action="/users"`,
		"Create new user",
		"that username is already taken",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected rendered login page to contain %q, got:\n%s", want, body)
		}
	}
}

func TestLoginTemplateRendersEmptyState(t *testing.T) {
	ts, ok := parseTestTemplates(t)["login.html"]
	if !ok {
		t.Fatal(`template "login.html" not found`)
	}

	data := LoginData{PageData: PageData{Title: "Sign in"}}

	rec := httptest.NewRecorder()
	if err := ts.Execute(rec, data); err != nil {
		t.Fatalf("execute login.html: %v", err)
	}

	if !strings.Contains(rec.Body.String(), "No users yet") {
		t.Errorf("expected empty state message, got:\n%s", rec.Body.String())
	}
}
