package handlers

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
)

// recipeDetailPageFixture builds a fully populated RecipeDetailPageData with
// a mix of missing and in-stock ingredients plus multi-step instructions, so
// a single template render exercises the happy path end to end.
func recipeDetailPageFixture(t *testing.T) RecipeDetailPageData {
	t.Helper()

	now := time.Now()
	servings := int32(4)
	prep := int32(10)
	cook := int32(20)
	desc := "Quick weeknight dinner"

	rows := []store.RecipeIngredientDetailRow{
		{IngredientID: 1, IngredientName: "Basil", Quantity: floatPtr(2), UnitName: strPtr("leaves"), Missing: true},
		{IngredientID: 2, IngredientName: "Flour", Quantity: floatPtr(200), UnitName: strPtr("g"), Missing: false},
		{IngredientID: 3, IngredientName: "Tomato", Note: strPtr("ripe"), Missing: true},
	}

	return RecipeDetailPageData{
		PageData: PageData{Title: "Pasta al pomodoro", SignedIn: true, CurrentUser: "Alice", ActiveNav: "recipes"},
		Recipe: models.Recipe{
			ID: 101, Name: "Pasta al pomodoro",
			Description: &desc, Servings: &servings,
			PrepTimeMinutes: &prep, CookTimeMinutes: &cook,
			CreatedAt: now, UpdatedAt: now,
		},
		Tags:        []string{"quick", "vegetarian"},
		Ingredients: OrderRecipeIngredients(rows),
		Instructions: ParseInstructionSteps(
			"1. Boil the pasta\n2. Simmer the sauce\n3. Toss together"),
		SelectedPantry: &models.Pantry{ID: 1, Name: "Main kitchen", CreatedAt: now, UpdatedAt: now},
		Pantries: []models.Pantry{
			{ID: 1, Name: "Main kitchen"},
		},
	}
}

func TestRecipeDetailTemplateRendersRecipe(t *testing.T) {
	ts, ok := parseTestTemplates(t)["recipe_detail.html"]
	if !ok {
		t.Fatal("recipe_detail.html not found in template cache")
	}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, recipeDetailPageFixture(t)); err != nil {
		t.Fatalf("execute recipe_detail.html: %v", err)
	}
	body := buf.String()

	for _, want := range []string{
		`<title>Pasta al pomodoro`,
		"Pasta al pomodoro",
		"Quick weeknight dinner",
		"Serves <strong>4</strong>",
		"Prep <strong>10m</strong>",
		"Cook <strong>20m</strong>",
		"Main kitchen",
		`<span class="wfd-tag-chip">quick</span>`,
		`<span class="wfd-tag-chip">vegetarian</span>`,
		`href="/recipes"`, // back link
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected %q in rendered body", want)
		}
	}

	// Ingredient rows: names, quantities and the missing-highlight class.
	for _, want := range []string{
		"Basil", "Flour", "Tomato",
		"2 leaves",
		"200 g",
		"ripe",
		"wfd-ingredient-item--missing",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected %q in rendered body", want)
		}
	}

	// Missing ingredients (Basil, Tomato) must be listed before the
	// in-stock one (Flour).
	basilIdx := strings.Index(body, "Basil")
	tomatoIdx := strings.Index(body, "Tomato")
	flourIdx := strings.Index(body, "Flour")
	if basilIdx == -1 || tomatoIdx == -1 || flourIdx == -1 {
		t.Fatal("expected all three ingredients to render")
	}
	if !(basilIdx < flourIdx && tomatoIdx < flourIdx) {
		t.Errorf("expected missing ingredients (Basil, Tomato) before in-stock ones (Flour); got indices basil=%d tomato=%d flour=%d",
			basilIdx, tomatoIdx, flourIdx)
	}

	// Instructions rendered as an ordered list, one <li> per step.
	for _, want := range []string{
		"<li>Boil the pasta</li>",
		"<li>Simmer the sauce</li>",
		"<li>Toss together</li>",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected %q in rendered body", want)
		}
	}
}

func TestRecipeDetailTemplateEmptyIngredientsAndInstructions(t *testing.T) {
	ts, ok := parseTestTemplates(t)["recipe_detail.html"]
	if !ok {
		t.Fatal("recipe_detail.html not found in template cache")
	}

	data := RecipeDetailPageData{
		PageData: PageData{Title: "Mystery dish", ActiveNav: "recipes"},
		Recipe:   models.Recipe{ID: 5, Name: "Mystery dish"},
	}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, data); err != nil {
		t.Fatalf("execute recipe_detail.html: %v", err)
	}
	body := buf.String()

	if !strings.Contains(body, "No ingredients listed for this recipe.") {
		t.Error("expected the empty-ingredients message")
	}
	if !strings.Contains(body, "No instructions provided.") {
		t.Error("expected the empty-instructions message")
	}
}

func TestRecipeDetailTemplateNoMissingIngredients(t *testing.T) {
	ts, ok := parseTestTemplates(t)["recipe_detail.html"]
	if !ok {
		t.Fatal("recipe_detail.html not found in template cache")
	}

	data := recipeDetailPageFixture(t)
	rows := []store.RecipeIngredientDetailRow{
		{IngredientID: 1, IngredientName: "Flour", Missing: false},
	}
	data.Ingredients = OrderRecipeIngredients(rows)

	var buf bytes.Buffer
	if err := ts.Execute(&buf, data); err != nil {
		t.Fatalf("execute recipe_detail.html: %v", err)
	}
	body := buf.String()

	// Note: ".wfd-ingredient-item--missing" (with a leading dot) always
	// appears in the embedded <style> block regardless of data, so assert
	// on the exact class="..." attribute the template would emit instead.
	if strings.Contains(body, `class="wfd-ingredient-item wfd-ingredient-item--missing"`) {
		t.Error("expected no missing-highlight class when every ingredient is in stock")
	}
}
