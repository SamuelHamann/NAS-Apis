package handlers

import (
	"bytes"
	"strings"
	"testing"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
)

func TestSafeRedirectTarget(t *testing.T) {
	tests := []struct {
		raw      string
		fallback string
		want     string
	}{
		{"/recipes?tags=1", "/recipes", "/recipes?tags=1"},
		{"", "/recipes", "/recipes"},
		{"//evil.example.com", "/recipes", "/recipes"},
		{"https://evil.example.com", "/recipes", "/recipes"},
		{"not-a-path", "/recipes", "/recipes"},
	}
	for _, tt := range tests {
		if got := safeRedirectTarget(tt.raw, tt.fallback); got != tt.want {
			t.Errorf("safeRedirectTarget(%q, %q) = %q, want %q", tt.raw, tt.fallback, got, tt.want)
		}
	}
}

func TestRecipesTemplateRendersCollectionFilterAndPicker(t *testing.T) {
	ts, ok := parseTestTemplates(t)["recipes.html"]
	if !ok {
		t.Fatal("recipes.html not found in template cache")
	}

	data := recipesPageFixture(t)
	data.UserCollections = []models.Collection{
		{ID: 1, Name: "Weeknight Dinners"},
		{ID: 2, Name: "Meal Prep"},
	}
	// Recipe 101 (from recipesPageFixture) is already in collection 1.
	data.CollectionMembership = map[int64]map[int64]bool{
		101: {1: true},
	}
	data.CurrentURL = "/recipes?sort=missing"
	data.Authors = []string{"Alice"}
	author := "Alice"
	data.Cards[0].Items[0].AuthorUsername = &author

	var buf bytes.Buffer
	if err := ts.Execute(&buf, data); err != nil {
		t.Fatalf("execute recipes.html: %v", err)
	}
	body := buf.String()

	for _, want := range []string{
		"Weeknight Dinners", "Meal Prep",
		`data-filter-group="collections"`,
		`data-filter-group="creator"`,
		`data-filter-value="1"`,
		`data-filter-value="2"`,
		`data-filter-value="Alice"`,
		`data-filter-collections="1"`,
		`data-filter-creator="Alice"`,
		`action="/collections/1/recipes/101/toggle"`,
		`action="/collections/2/recipes/101/toggle"`,
		`value="/recipes?sort=missing"`,
		"wfd-status-item--card-link",
		"wfd-status-item__stretched-link",
		`class="wfd-icon-btn wfd-icon-btn--bookmarked"`,
		"By <strong>Alice</strong>",
		"<span>|</span>",
		"syncBookmarkIcon",   // collection-toggle-script actually included
		"wfd-dropdown[open]", // dropdown-script actually included
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected %q in rendered body", want)
		}
	}

	// Collection 1's checkbox (recipe already a member) should be checked;
	// collection 2's should not.
	seg := func(marker string) string {
		i := strings.Index(body, marker)
		if i < 0 || i+300 > len(body) {
			t.Fatalf("marker %q not found (or too close to end)", marker)
		}
		return body[i : i+300]
	}
	if !strings.Contains(seg(`action="/collections/1/recipes/101/toggle"`), "checked") {
		t.Error("expected collection 1's checkbox to be pre-checked for recipe 101")
	}
	if strings.Contains(seg(`action="/collections/2/recipes/101/toggle"`), "checked") {
		t.Error("expected collection 2's checkbox to NOT be pre-checked for recipe 101")
	}
}

func TestRecipesTemplateNoBookmarkWhenNotInAnyCollection(t *testing.T) {
	ts, ok := parseTestTemplates(t)["recipes.html"]
	if !ok {
		t.Fatal("recipes.html not found in template cache")
	}

	data := recipesPageFixture(t)
	data.UserCollections = []models.Collection{{ID: 1, Name: "Weeknight Dinners"}}
	data.CollectionMembership = map[int64]map[int64]bool{} // no recipe is a member of anything
	data.CurrentURL = "/recipes"

	var buf bytes.Buffer
	if err := ts.Execute(&buf, data); err != nil {
		t.Fatalf("execute recipes.html: %v", err)
	}
	// The class name legitimately appears once, in the <style> block's own
	// rule definition — check no element actually carries it.
	if strings.Contains(buf.String(), `class="wfd-icon-btn wfd-icon-btn--bookmarked"`) {
		t.Error("expected no bookmarked icon when no recipe is in any collection")
	}
}

func TestRecipeDetailTemplateRendersAuthorAndCollectionPicker(t *testing.T) {
	ts, ok := parseTestTemplates(t)["recipe_detail.html"]
	if !ok {
		t.Fatal("recipe_detail.html not found in template cache")
	}

	author := "Sam"
	data := RecipeDetailPageData{
		PageData:        PageData{Title: "Soup", SignedIn: true, CurrentUser: "Sam"},
		Recipe:          models.Recipe{ID: 7, Name: "Soup"},
		Author:          &author,
		UserCollections: []models.Collection{{ID: 1, Name: "Weeknight Dinners"}},
		CollectionMembership: map[int64]bool{
			1: true,
		},
		CurrentURL: "/recipes/7",
	}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, data); err != nil {
		t.Fatalf("execute recipe_detail.html: %v", err)
	}
	body := buf.String()

	for _, want := range []string{
		"By <strong>Sam</strong>",
		"Weeknight Dinners",
		`action="/collections/1/recipes/7/toggle"`,
		`value="/recipes/7"`,
		`class="wfd-icon-btn wfd-icon-btn--bookmarked"`,
		"syncBookmarkIcon",
		"wfd-dropdown[open]",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected %q in rendered body", want)
		}
	}
}

func TestProfileTemplateSignedOut(t *testing.T) {
	ts, ok := parseTestTemplates(t)["profile.html"]
	if !ok {
		t.Fatal("profile.html not found in template cache")
	}

	data := ProfilePageData{PageData: PageData{Title: "Profile"}}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, data); err != nil {
		t.Fatalf("execute profile.html: %v", err)
	}
	if !strings.Contains(buf.String(), "Sign in") {
		t.Error("expected a sign-in prompt when signed out")
	}
}

func TestProfileTemplateRendersCollections(t *testing.T) {
	ts, ok := parseTestTemplates(t)["profile.html"]
	if !ok {
		t.Fatal("profile.html not found in template cache")
	}

	data := ProfilePageData{
		PageData:    PageData{Title: "Profile", SignedIn: true, CurrentUser: "Sam"},
		Collections: []models.Collection{{ID: 1, Name: "Weeknight Dinners"}},
	}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, data); err != nil {
		t.Fatalf("execute profile.html: %v", err)
	}
	body := buf.String()
	for _, want := range []string{
		"Sam",
		"Weeknight Dinners",
		`action="/collections"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected %q in rendered body", want)
		}
	}
}

func TestProfileTemplateEmptyCollections(t *testing.T) {
	ts, ok := parseTestTemplates(t)["profile.html"]
	if !ok {
		t.Fatal("profile.html not found in template cache")
	}

	data := ProfilePageData{
		PageData: PageData{Title: "Profile", SignedIn: true, CurrentUser: "Sam"},
	}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, data); err != nil {
		t.Fatalf("execute profile.html: %v", err)
	}
	if !strings.Contains(buf.String(), "No collections yet") {
		t.Error("expected the no-collections empty state to render")
	}
}

// Compile-time sanity check that store.CollectionRecipe's field names match
// what the handlers expect (RecipeID/CollectionID) — a typo here would only
// otherwise surface once real DB rows exist.
var _ = store.CollectionRecipe{CollectionID: 1, RecipeID: 2}
