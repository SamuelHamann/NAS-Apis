package handlers

import (
	"bytes"
	"strings"
	"testing"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
)

func TestAdminTemplateRendersAccessMatrix(t *testing.T) {
	ts, ok := parseTestTemplates(t)["admin.html"]
	if !ok {
		t.Fatal("admin.html not found in template cache")
	}

	data := AdminPageData{
		PageData: PageData{Title: "Admin"},
		Pantries: []models.Pantry{{ID: 1, Name: "Main kitchen"}, {ID: 2, Name: "Garage freezer"}},
		Users:    []models.User{{ID: 10, Username: "Sam"}, {ID: 20, Username: "Alex"}},
		Access: map[int64]map[int64]bool{
			10: {1: true},
		},
	}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, data); err != nil {
		t.Fatalf("execute admin.html: %v", err)
	}
	body := buf.String()

	for _, want := range []string{
		"Main kitchen", "Garage freezer", "Sam", "Alex",
		`value="10:1"`, `value="10:2"`, `value="20:1"`, `value="20:2"`,
		`action="/settings/admin/pantries"`,
		`action="/settings/admin/pantries/1/update"`,
		`action="/settings/admin/pantries/2/update"`,
		`action="/settings/admin/pantry-access"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected %q in rendered body", want)
		}
	}

	seg := func(marker string) string {
		i := strings.Index(body, marker)
		if i < 0 || i+200 > len(body) {
			t.Fatalf("marker %q not found (or too close to end)", marker)
		}
		return body[i : i+200]
	}
	if !strings.Contains(seg(`value="10:1"`), "checked") {
		t.Error("expected Sam/Main kitchen (10:1) to be pre-checked")
	}
	if strings.Contains(seg(`value="10:2"`), "checked") {
		t.Error("expected Sam/Garage freezer (10:2) to NOT be pre-checked")
	}
	if strings.Contains(seg(`value="20:1"`), "checked") {
		t.Error("expected Alex/Main kitchen (20:1) to NOT be pre-checked")
	}
}

func TestAdminTemplateRendersEmptyStates(t *testing.T) {
	ts, ok := parseTestTemplates(t)["admin.html"]
	if !ok {
		t.Fatal("admin.html not found in template cache")
	}

	data := AdminPageData{PageData: PageData{Title: "Admin"}}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, data); err != nil {
		t.Fatalf("execute admin.html: %v", err)
	}
	body := buf.String()

	if !strings.Contains(body, "No pantries yet") {
		t.Error("expected the no-pantries empty state to render")
	}
	if !strings.Contains(body, "No users yet") {
		t.Error("expected the no-users empty state to render")
	}
}

func TestAdminTemplateRendersError(t *testing.T) {
	ts, ok := parseTestTemplates(t)["admin.html"]
	if !ok {
		t.Fatal("admin.html not found in template cache")
	}

	data := AdminPageData{
		PageData: PageData{Title: "Admin"},
		Error:    "pantry name is required",
	}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, data); err != nil {
		t.Fatalf("execute admin.html: %v", err)
	}
	if !strings.Contains(buf.String(), "pantry name is required") {
		t.Error("expected the error banner to render")
	}
}

func TestParseAccessCell(t *testing.T) {
	tests := []struct {
		raw        string
		wantUserID int64
		wantPantry int64
		wantOK     bool
	}{
		{"10:1", 10, 1, true},
		{"0:1", 0, 0, false},
		{"10:0", 0, 0, false},
		{"10", 0, 0, false},
		{"10:1:extra", 0, 0, false},
		{"abc:1", 0, 0, false},
		{"", 0, 0, false},
	}

	for _, tt := range tests {
		got, ok := parseAccessCell(tt.raw)
		if ok != tt.wantOK {
			t.Errorf("parseAccessCell(%q) ok = %v, want %v", tt.raw, ok, tt.wantOK)
			continue
		}
		if ok && (got.UserID != tt.wantUserID || got.PantryID != tt.wantPantry) {
			t.Errorf("parseAccessCell(%q) = %+v, want {UserID:%d PantryID:%d}", tt.raw, got, tt.wantUserID, tt.wantPantry)
		}
	}
}
