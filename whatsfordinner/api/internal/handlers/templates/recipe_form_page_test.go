package handlers

import (
	"bytes"
	"strings"
	"testing"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
)

func int32Ptr(i int32) *int32 { return &i }
func int64Ptr(i int64) *int64 { return &i }

func recipeFormFixtureData() (models.Recipe, []models.Ingredient, []models.Unit, []models.Tag) {
	recipe := models.Recipe{
		ID:          42,
		Name:        "Pancakes",
		Description: strPtr("A tasty dish"),
		Servings:    int32Ptr(4),
	}
	ingredients := []models.Ingredient{{ID: 1, Name: "Flour"}, {ID: 2, Name: "Sugar"}}
	units := []models.Unit{{ID: 10, Name: "cup"}}
	tags := []models.Tag{{ID: 100, Name: "quick"}}
	return recipe, ingredients, units, tags
}

func TestRecipeFormTemplateRendersNew(t *testing.T) {
	ts, ok := parseTestTemplates(t)["recipe_form.html"]
	if !ok {
		t.Fatal("recipe_form.html not found in template cache")
	}

	_, ingredients, units, tags := recipeFormFixtureData()
	data := RecipeFormPageData{
		PageData:    PageData{Title: "New recipe"},
		Ingredients: ingredients,
		Units:       units,
		Tags:        tags,
	}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, data); err != nil {
		t.Fatalf("execute recipe_form.html: %v", err)
	}
	body := buf.String()

	for _, want := range []string{
		"New recipe",
		`action="/recipes"`,
		"Create recipe",
		"Flour", "Sugar", "cup", "quick",
		"data-ingredient-picker",
		"data-item-rows",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected %q in rendered body", want)
		}
	}
}

func TestRecipeFormTemplateRendersEdit(t *testing.T) {
	ts, ok := parseTestTemplates(t)["recipe_form.html"]
	if !ok {
		t.Fatal("recipe_form.html not found in template cache")
	}

	recipe, ingredients, units, tags := recipeFormFixtureData()
	data := RecipeFormPageData{
		PageData: PageData{Title: "Edit " + recipe.Name},
		Recipe:   &recipe,
		Items: []store.RecipeIngredientEditRow{
			{IngredientID: 1, IngredientName: "Flour", Quantity: floatPtr(2), UnitID: int64Ptr(10), UnitName: ptrString("cup")},
		},
		Ingredients:    ingredients,
		Units:          units,
		Tags:           tags,
		SelectedTagIDs: map[int64]bool{100: true},
	}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, data); err != nil {
		t.Fatalf("execute recipe_form.html: %v", err)
	}
	body := buf.String()

	for _, want := range []string{
		"Edit Pancakes",
		`action="/recipes/42/update"`,
		"Save changes",
		"A tasty dish",
		`value="4"`,
		`selected>Flour`, // the recipe's own ingredient row prefilled
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected %q in rendered body", want)
		}
	}
}

func TestRecipeDetailTemplateRendersEditLink(t *testing.T) {
	ts, ok := parseTestTemplates(t)["recipe_detail.html"]
	if !ok {
		t.Fatal("recipe_detail.html not found in template cache")
	}

	data := RecipeDetailPageData{
		PageData: PageData{Title: "Soup"},
		Recipe:   models.Recipe{ID: 7, Name: "Soup"},
	}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, data); err != nil {
		t.Fatalf("execute recipe_detail.html: %v", err)
	}
	if !strings.Contains(buf.String(), `href="/recipes/7/edit"`) {
		t.Error("expected an edit link pointing at /recipes/7/edit")
	}
}

func TestParseRecipeFormRequiresName(t *testing.T) {
	_, msg := parseRecipeForm(map[string][]string{})
	if msg == "" {
		t.Fatal("expected a validation message when name is missing")
	}
}

func TestParseRecipeFormAllowsZeroIngredients(t *testing.T) {
	in, msg := parseRecipeForm(map[string][]string{"name": {"Soup"}})
	if msg != "" {
		t.Fatalf("expected no validation error, got %q", msg)
	}
	if len(in.Ingredients) != 0 {
		t.Errorf("expected zero ingredients, got %d", len(in.Ingredients))
	}
}

func TestParseRecipeFormRejectsDuplicateIngredients(t *testing.T) {
	form := map[string][]string{
		"name":               {"Soup"},
		"item_ingredient_id": {"1", "1"},
	}
	_, msg := parseRecipeForm(form)
	if msg != "each ingredient can only appear once in a recipe" {
		t.Errorf("expected duplicate-ingredient message, got %q", msg)
	}
}

func TestParseRecipeFormSkipsBlankIngredientRows(t *testing.T) {
	form := map[string][]string{
		"name":               {"Soup"},
		"item_ingredient_id": {"1", ""},
		"item_quantity":      {"2", ""},
		"item_unit_id":       {"10", ""},
		"item_note":          {"", ""},
	}
	in, msg := parseRecipeForm(form)
	if msg != "" {
		t.Fatalf("expected no validation error, got %q", msg)
	}
	if len(in.Ingredients) != 1 {
		t.Fatalf("expected exactly 1 ingredient (blank row skipped), got %d", len(in.Ingredients))
	}
	if in.Ingredients[0].IngredientID != 1 {
		t.Errorf("expected ingredient id 1, got %d", in.Ingredients[0].IngredientID)
	}
}

func TestParseRecipeFormValidatesOptionalMinutes(t *testing.T) {
	form := map[string][]string{
		"name":              {"Soup"},
		"prep_time_minutes": {"-5"},
	}
	_, msg := parseRecipeForm(form)
	if msg == "" {
		t.Fatal("expected a validation message for a negative prep time")
	}
}
