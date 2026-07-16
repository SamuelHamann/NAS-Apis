package handlers

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
)

// recipeCookSessionPageFixture mirrors recipeDetailPageFixture's ingredients
// and instructions so both templates are exercised against the same shape of
// data.
func recipeCookSessionPageFixture(t *testing.T) RecipeCookSessionPageData {
	t.Helper()

	now := time.Now()
	unitID := int64(14)

	rows := []store.RecipeIngredientDetailRow{
		{IngredientID: 1, IngredientName: "Basil", Quantity: floatPtr(2), UnitName: strPtr("leaves"), Missing: true},
		{IngredientID: 2, IngredientName: "Flour", Quantity: floatPtr(200), UnitName: strPtr("g"), Missing: false},
	}

	return RecipeCookSessionPageData{
		PageData: PageData{Title: "Cook Pasta al pomodoro", SignedIn: true, CurrentUser: "Alice", ActiveNav: "recipes"},
		Recipe: models.Recipe{
			ID: 101, Name: "Pasta al pomodoro",
			CreatedAt: now, UpdatedAt: now,
		},
		Ingredients: OrderRecipeIngredients(rows),
		Instructions: ParseInstructionSteps(
			"1. Boil the pasta\n2. Simmer the sauce\n3. Toss together"),
		SelectedPantry:      &models.Pantry{ID: 1, Name: "Main kitchen", CreatedAt: now, UpdatedAt: now},
		PantryID:            1,
		MultiplierRaw:       "1.5",
		CreateCombined:      true,
		CombinedQuantityRaw: "1.5",
		CombinedUnitID:      &unitID,
	}
}

func TestRecipeCookSessionTemplateRendersChecklist(t *testing.T) {
	ts, ok := parseTestTemplates(t)["recipe_cook_session.html"]
	if !ok {
		t.Fatal("recipe_cook_session.html not found in template cache")
	}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, recipeCookSessionPageFixture(t)); err != nil {
		t.Fatalf("execute recipe_cook_session.html: %v", err)
	}
	body := buf.String()

	for _, want := range []string{
		"<title>Cook Pasta al pomodoro",
		"Cooking Pasta al pomodoro",
		"Main kitchen",
		"1.5",
		"Basil", "Flour",
		"2 leaves",
		"200 g",
		`action="/recipes/101/cook"`,
		`name="confirmed" value="true"`,
		`name="pantry_id" value="1"`,
		`name="multiplier" value="1.5"`,
		`name="create_combined" value="on"`,
		`name="combined_quantity" value="1.5"`,
		`name="combined_unit_id" value="14"`,
		`href="/recipes/101?pantry_id=1"`,
		"Finish cooking",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected %q in rendered body", want)
		}
	}

	// One checkbox per ingredient (2) plus one per instruction step (3).
	if got := strings.Count(body, `class="wfd-checklist__box"`); got != 5 {
		t.Errorf("expected 5 checklist checkboxes, got %d", got)
	}

	if !strings.Contains(body, `wake-lock-script`) && !strings.Contains(body, "wakeLock") {
		t.Error("expected the wake-lock script to be included")
	}
}

func TestRecipeCookSessionTemplateEmptyIngredientsAndInstructions(t *testing.T) {
	ts, ok := parseTestTemplates(t)["recipe_cook_session.html"]
	if !ok {
		t.Fatal("recipe_cook_session.html not found in template cache")
	}

	data := RecipeCookSessionPageData{
		PageData: PageData{Title: "Cook Mystery dish", ActiveNav: "recipes"},
		Recipe:   models.Recipe{ID: 5, Name: "Mystery dish"},
		PantryID: 1,
	}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, data); err != nil {
		t.Fatalf("execute recipe_cook_session.html: %v", err)
	}
	body := buf.String()

	if !strings.Contains(body, "No ingredients listed for this recipe.") {
		t.Error("expected the empty-ingredients message")
	}
	if !strings.Contains(body, "No instructions provided.") {
		t.Error("expected the empty-instructions message")
	}
	if strings.Contains(body, `name="create_combined"`) {
		t.Error("expected no combined-ingredient hidden fields when CreateCombined is false")
	}
}
