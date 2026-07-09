package handlers

import (
	"testing"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
)

// recipeRow is a compact constructor for a store.RecipeStatusRow. Only the
// fields the grouping/classification code cares about are populated — the
// rest zero-value cleanly. ID is stable-hashed off the name so failed test
// output stays readable.
func recipeRow(name string, missing int64, tags ...string) store.RecipeStatusRow {
	return store.RecipeStatusRow{
		Recipe: models.Recipe{
			ID:   int64(len(name)), // arbitrary, deterministic, non-zero
			Name: name,
		},
		MissingCount: missing,
		TagNames:     tags,
	}
}

func TestClassifyRecipeReadiness(t *testing.T) {
	cases := []struct {
		missing int64
		want    RecipeReadiness
	}{
		{-3, RecipeReady},   // defensive: never happens in practice
		{0, RecipeReady},    // fully stocked
		{1, RecipeNearly},   // one missing = "almost there"
		{4, RecipeNearly},   // inclusive upper bound
		{5, RecipeFaraway},  // just over the line
		{42, RecipeFaraway}, // far away
	}
	for _, tc := range cases {
		if got := classifyRecipeReadiness(tc.missing); got != tc.want {
			t.Errorf("classifyRecipeReadiness(%d) = %q, want %q", tc.missing, got, tc.want)
		}
	}
}

func TestParseRecipeSortFallback(t *testing.T) {
	for _, s := range []string{"", "wat", "cook-time", "MISSING"} {
		if got := parseRecipeSort(s); got != RecipeSortMissing {
			t.Errorf("parseRecipeSort(%q) = %q, want %q", s, got, RecipeSortMissing)
		}
	}
	for _, s := range []string{"missing", "alphabetical"} {
		if got := parseRecipeSort(s); string(got) != s {
			t.Errorf("parseRecipeSort(%q) = %q, want %q", s, got, s)
		}
	}
}

func TestGroupRecipesByMissing(t *testing.T) {
	rows := []store.RecipeStatusRow{
		// Store returns rows already sorted by missing_count ASC then name;
		// mirror that here so the test exercises the same input shape.
		recipeRow("Bread", 0),
		recipeRow("Salad", 0),
		recipeRow("Pasta", 2),
		recipeRow("Stew", 4),
		recipeRow("Curry", 7),
		recipeRow("Sushi", 12),
	}

	cards := GroupRecipes(rows, RecipeSortMissing)

	if len(cards) != 3 {
		t.Fatalf("expected 3 cards, got %d", len(cards))
	}
	if cards[0].Title != "Ready to cook" || cards[0].Tone != "success" {
		t.Errorf("card 0 = %+v, want Ready to cook / success", cards[0])
	}
	if cards[1].Title != "Almost there (missing 4 or fewer)" || cards[1].Tone != "warning" {
		t.Errorf("card 1 = %+v, want Almost there / warning", cards[1])
	}
	if cards[2].Title != "Everything else" || cards[2].Tone != "" {
		t.Errorf("card 2 = %+v, want Everything else / no tone", cards[2])
	}

	// Ready card: Bread, Salad (input order preserved by partition).
	if names := recipeNames(cards[0].Items); names[0] != "Bread" || names[1] != "Salad" {
		t.Errorf("ready card items = %v, want [Bread, Salad]", names)
	}
	// Nearly card: Pasta (2), Stew (4).
	if names := recipeNames(cards[1].Items); names[0] != "Pasta" || names[1] != "Stew" {
		t.Errorf("nearly card items = %v, want [Pasta, Stew]", names)
	}
	// Faraway card: Curry (7), Sushi (12).
	if names := recipeNames(cards[2].Items); names[0] != "Curry" || names[1] != "Sushi" {
		t.Errorf("faraway card items = %v, want [Curry, Sushi]", names)
	}

	// Every row must carry its own Readiness so the template can tint
	// individual rows even in an alphabetical layout.
	for _, it := range cards[0].Items {
		if it.Readiness != RecipeReady {
			t.Errorf("ready card item %q got readiness %q", it.Name, it.Readiness)
		}
	}
	for _, it := range cards[1].Items {
		if it.Readiness != RecipeNearly {
			t.Errorf("nearly card item %q got readiness %q", it.Name, it.Readiness)
		}
	}
}

func TestGroupRecipesByMissingOmitsEmptyCards(t *testing.T) {
	rows := []store.RecipeStatusRow{
		recipeRow("Curry", 8),
		recipeRow("Sushi", 15),
	}
	cards := GroupRecipes(rows, RecipeSortMissing)
	if len(cards) != 1 || cards[0].Title != "Everything else" {
		t.Fatalf("expected single 'Everything else' card, got %+v", cards)
	}
}

func TestGroupRecipesAlphabetical(t *testing.T) {
	rows := []store.RecipeStatusRow{
		recipeRow("Curry", 7),
		recipeRow("Bread", 0),
		recipeRow("Salad", 3),
	}

	cards := GroupRecipes(rows, RecipeSortAlphabetical)
	if len(cards) != 1 || cards[0].Title != "All recipes" {
		t.Fatalf("alphabetical sort should yield one 'All recipes' card, got %+v", cards)
	}
	if names := recipeNames(cards[0].Items); names[0] != "Bread" || names[1] != "Curry" || names[2] != "Salad" {
		t.Errorf("alphabetical items = %v, want [Bread, Curry, Salad]", names)
	}
	// Colour class survives the sort mode change.
	if cards[0].Items[0].Readiness != RecipeReady {
		t.Errorf("Bread should still be flagged ready in alphabetical mode, got %q", cards[0].Items[0].Readiness)
	}
	if cards[0].Items[2].Readiness != RecipeNearly {
		t.Errorf("Salad (3 missing) should be flagged nearly in alphabetical mode, got %q", cards[0].Items[2].Readiness)
	}
}

func TestGroupRecipesAlphabeticalEmpty(t *testing.T) {
	if cards := GroupRecipes(nil, RecipeSortAlphabetical); cards != nil {
		t.Errorf("expected nil cards for empty input, got %+v", cards)
	}
}

func recipeNames(items []RecipeItemView) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Name
	}
	return out
}
