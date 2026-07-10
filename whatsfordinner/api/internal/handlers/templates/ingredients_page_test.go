package handlers

import (
	"bytes"
	"strings"
	"testing"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
)

// ingredientsPageFixture builds a fully populated IngredientsPageData: one
// ingredient with tags, one without, a preselected tag filter and an error
// message, so the test can assert the happy path and the edge cases of
// ingredients.html in one shot.
func ingredientsPageFixture() IngredientsPageData {
	return IngredientsPageData{
		PageData: PageData{Title: "Ingredients", SignedIn: true, CurrentUser: "Alice", ActiveNav: "ingredients"},
		Items: []store.IngredientDetailRow{
			{ID: 10, Name: "Milk", TagNames: []string{"dairy", "vegetarian"}},
			{ID: 11, Name: "Bread", TagNames: nil},
			{ID: 12, Name: "Rice"},
		},
		SelectedTagIDs: map[int64]bool{5: true},
		Tags: []models.Tag{
			{ID: 5, Name: "vegan"},
			{ID: 6, Name: "dairy"},
		},
	}
}

func TestIngredientsTemplateRendersListAndForms(t *testing.T) {
	ts, ok := parseTestTemplates(t)["ingredients.html"]
	if !ok {
		t.Fatal("ingredients.html not found in template cache")
	}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, ingredientsPageFixture()); err != nil {
		t.Fatalf("execute ingredients.html: %v", err)
	}
	body := buf.String()

	for _, want := range []string{
		`<title>Ingredients`,
		"Milk", "Bread", "Rice", // ingredient rows
		"dairy", "vegetarian", // tag chips on the Milk row
		"vegan",          // tag filter chip
		"Add ingredient", // create toggle
		`name="tag_ids"`, // tag checkboxes in the form
		`data-search-input="#ingredients-list"`,
		`data-search-name="Milk"`,
		`action="/ingredients/10/update"`,
		`action="/ingredients/10/delete"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected %q in rendered body", want)
		}
	}
}

func TestIngredientsTemplateRendersErrorAndEmptyState(t *testing.T) {
	ts, ok := parseTestTemplates(t)["ingredients.html"]
	if !ok {
		t.Fatal("ingredients.html not found in template cache")
	}

	data := ingredientsPageFixture()
	data.Items = nil
	data.Error = "an ingredient with that name already exists"

	var buf bytes.Buffer
	if err := ts.Execute(&buf, data); err != nil {
		t.Fatalf("execute ingredients.html: %v", err)
	}
	body := buf.String()

	if !strings.Contains(body, "an ingredient with that name already exists") {
		t.Error("expected the error banner to render")
	}
	if !strings.Contains(body, "No ingredients match this tag filter.") {
		t.Error("expected the tag-filtered empty state to render")
	}
}

func TestParseIngredientForm(t *testing.T) {
	form := map[string][]string{
		"name":    {"  Olive oil  "},
		"tag_ids": {"5", "6", "not-a-number", "0"},
	}
	in, msg := parseIngredientForm(form)
	if msg != "" {
		t.Fatalf("expected no validation message, got %q", msg)
	}
	if in.Name != "Olive oil" {
		t.Errorf("expected trimmed name %q, got %q", "Olive oil", in.Name)
	}
	if len(in.TagIDs) != 2 || in.TagIDs[0] != 5 || in.TagIDs[1] != 6 {
		t.Errorf("expected TagIDs [5 6], got %v", in.TagIDs)
	}
}

func TestParseIngredientFormRequiresName(t *testing.T) {
	_, msg := parseIngredientForm(map[string][]string{"name": {"   "}})
	if msg == "" {
		t.Error("expected a validation message for a blank name")
	}
}
