package handlers

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
)

// recipesPageFixture builds a fully populated RecipesPageData with recipes
// in every readiness bucket, so a single template render exercises the
// happy path plus the toolbar, empty checks, tag chips and colour classes.
func recipesPageFixture(t *testing.T) RecipesPageData {
	t.Helper()

	now := time.Now()
	servings := int32(4)
	prep := int32(10)
	cook := int32(20)
	desc := "Quick weeknight dinner"

	rows := []store.RecipeStatusRow{
		{
			Recipe: models.Recipe{
				ID: 101, Name: "Pasta al pomodoro",
				Description: &desc, Servings: &servings,
				PrepTimeMinutes: &prep, CookTimeMinutes: &cook,
				CreatedAt: now, UpdatedAt: now,
			},
			MissingCount: 0, // ready
			TagNames:     []string{"quick", "vegetarian"},
		},
		{
			Recipe:       models.Recipe{ID: 102, Name: "Chicken curry", CreatedAt: now, UpdatedAt: now},
			MissingCount: 3, // nearly
			TagNames:     []string{"spicy"},
		},
		{
			Recipe:       models.Recipe{ID: 103, Name: "Sushi platter", CreatedAt: now, UpdatedAt: now},
			MissingCount: 12, // faraway
			TagNames:     nil,
		},
	}

	return RecipesPageData{
		PageData:       PageData{Title: "Recipes", SignedIn: true, CurrentUser: "Alice", ActiveNav: "recipes"},
		SelectedPantry: &models.Pantry{ID: 1, Name: "Main kitchen", CreatedAt: now, UpdatedAt: now},
		Pantries: []models.Pantry{
			{ID: 1, Name: "Main kitchen"},
			{ID: 2, Name: "Cabin"},
		},
		Cards:          GroupRecipes(rows, RecipeSortMissing),
		Sort:           RecipeSortMissing,
		SelectedTagIDs: map[int64]bool{5: true},
		Tags: []models.Tag{
			{ID: 5, Name: "quick"},
			{ID: 6, Name: "vegetarian"},
		},
	}
}

func TestRecipesTemplateRendersCards(t *testing.T) {
	ts, ok := parseTestTemplates(t)["recipes.html"]
	if !ok {
		t.Fatal("recipes.html not found in template cache")
	}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, recipesPageFixture(t)); err != nil {
		t.Fatalf("execute recipes.html: %v", err)
	}
	body := buf.String()

	// Chrome + toolbar.
	for _, want := range []string{
		`<title>Recipes`,
		`Main kitchen`, // subtitle + hidden field + option
		`Cabin`,        // second pantry in the picker
		`name="sort"`,
		`Ingredient availability`,
		`Alphabetical`,
		`quick`, `vegetarian`, // tag chips in the toolbar filter
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected %q in rendered body", want)
		}
	}

	// Card titles + tones.
	for _, want := range []string{
		`Ready to cook`,
		`wfd-status-card--success`,
		`Almost there (missing 4 or fewer)`,
		`wfd-status-card--warning`,
		`Everything else`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected %q in rendered body", want)
		}
	}

	// Per-row tint classes (must survive even in the alphabetical mode —
	// tested below).
	if !strings.Contains(body, "wfd-status-item--success") {
		t.Error("expected wfd-status-item--success class on the ready row")
	}
	if !strings.Contains(body, "wfd-status-item--warning") {
		t.Error("expected wfd-status-item--warning class on the nearly row")
	}

	// Row content.
	for _, want := range []string{
		"Pasta al pomodoro",
		"Chicken curry",
		"Sushi platter",
		"Ready to cook",              // qty label for the 0-missing row
		"Missing <strong>3</strong>", // qty label for the nearly row
		"Serves <strong>4</strong>",
		"Prep <strong>10m</strong>",
		"Cook <strong>20m</strong>",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected %q in rendered body", want)
		}
	}

	// Tag chip strip on the recipe row (distinct from toolbar chips —
	// non-interactive).
	if !strings.Contains(body, `<span class="wfd-tag-chip">quick</span>`) {
		t.Error("expected a quick tag chip on the pasta recipe row")
	}
}

// TestRecipesTemplateDoesNotNestForms is the recipe-side counterpart of
// TestPantryTemplateDoesNotNestForms. Same rationale: an accidental nested
// <form> reintroduces "this.form is null" and merges Add-item fields into
// the toolbar URL. The recipes page has no Add-item form today, but the
// toolbar structure is identical and could regress if someone copy-pastes
// the old pantry layout back in.
func TestRecipesTemplateDoesNotNestForms(t *testing.T) {
	ts, ok := parseTestTemplates(t)["recipes.html"]
	if !ok {
		t.Fatal("recipes.html not found in template cache")
	}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, recipesPageFixture(t)); err != nil {
		t.Fatalf("execute recipes.html: %v", err)
	}
	body := buf.String()

	const filterFormOpen = `<form id="recipes-view" method="get" action="/recipes" hidden>`
	if !strings.Contains(body, filterFormOpen+`</form>`) {
		t.Errorf("filter form must be immediately closed (empty). Body excerpt:\n%s",
			excerpt(body, "recipes-view", 200))
	}

	for _, want := range []string{
		`name="sort" form="recipes-view"`,
		`name="tags" value="5" form="recipes-view"`,
		`name="pantry_id" form="recipes-view"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected %q in rendered body", want)
		}
	}
}

func TestRecipesTemplateForcesAlphabeticalWithoutPantry(t *testing.T) {
	// Even if the caller somehow passed Sort=missing, the handler forces
	// alphabetical when there is no pantry; here we just verify the template
	// renders correctly in that state (missing option disabled, no pantry
	// picker, no crash on nil SelectedPantry).
	ts, ok := parseTestTemplates(t)["recipes.html"]
	if !ok {
		t.Fatal("recipes.html not found in template cache")
	}

	data := RecipesPageData{
		PageData: PageData{Title: "Recipes", ActiveNav: "recipes"},
		Sort:     RecipeSortAlphabetical,
		Cards: GroupRecipes([]store.RecipeStatusRow{
			{Recipe: models.Recipe{ID: 1, Name: "Bread"}, MissingCount: 0},
		}, RecipeSortAlphabetical),
	}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, data); err != nil {
		t.Fatalf("execute recipes.html: %v", err)
	}
	body := buf.String()

	if !strings.Contains(body, "Add a pantry") {
		t.Error("expected 'Add a pantry' subtitle when no pantry is selected")
	}
	if !strings.Contains(body, `value="missing"      disabled`) {
		t.Error("expected the 'missing' sort option to be disabled without a pantry")
	}
	if !strings.Contains(body, "All recipes") {
		t.Error("expected the alphabetical card title")
	}
}

func TestRecipesTemplateEmptyState(t *testing.T) {
	ts, ok := parseTestTemplates(t)["recipes.html"]
	if !ok {
		t.Fatal("recipes.html not found in template cache")
	}

	data := recipesPageFixture(t)
	data.Cards = nil
	data.SelectedTagIDs = map[int64]bool{5: true}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, data); err != nil {
		t.Fatalf("execute recipes.html: %v", err)
	}
	body := buf.String()

	if !strings.Contains(body, "No recipes match this filter") {
		t.Error("expected filter empty-state message when Cards is nil and tags are selected")
	}
}
