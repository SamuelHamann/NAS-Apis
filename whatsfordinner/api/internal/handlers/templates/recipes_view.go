package handlers

import (
	"sort"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
)

// RecipeSort chooses how ListRecipesWithStatus rows are grouped and
// ordered on the /recipes page.
type RecipeSort string

const (
	// RecipeSortMissing (default) groups recipes into three cards based on
	// how many of their ingredients are missing from the selected pantry:
	//     ready (0 missing) / nearly (1-4) / faraway (5+).
	// Within each card recipes are sorted by MissingCount ASC then name.
	RecipeSortMissing RecipeSort = "missing"

	// RecipeSortAlphabetical puts every recipe into a single card sorted
	// by name. Each row still carries its readiness so the color coding
	// survives the switch of sort mode.
	RecipeSortAlphabetical RecipeSort = "alphabetical"
)

// parseRecipeSort returns the sort matching s, defaulting to "missing" so
// a bad query string never breaks the page.
func parseRecipeSort(s string) RecipeSort {
	switch RecipeSort(s) {
	case RecipeSortAlphabetical:
		return RecipeSortAlphabetical
	default:
		return RecipeSortMissing
	}
}

// RecipeReadiness tags a recipe with how close to cookable it is. It drives
// both the group card (in "missing" sort) and the per-row background (in
// both sort modes), the same way ExpirationClass drives the pantry page.
type RecipeReadiness string

const (
	RecipeReady   RecipeReadiness = "ready"   // 0 missing → green
	RecipeNearly  RecipeReadiness = "nearly"  // 1..nearlyThreshold missing → orange
	RecipeFaraway RecipeReadiness = "faraway" // > nearlyThreshold → default
)

// nearlyThreshold is the "you're just a few ingredients away" cutoff, per
// the pantry page spec (spec says "4 or less").
const nearlyThreshold int64 = 4

// classifyRecipeReadiness maps a missing-ingredient count onto a readiness
// bucket. Negative counts (can't happen in practice — the SQL COUNT never
// goes below 0) are treated as "ready" so we never fail closed.
func classifyRecipeReadiness(missing int64) RecipeReadiness {
	switch {
	case missing <= 0:
		return RecipeReady
	case missing <= nearlyThreshold:
		return RecipeNearly
	default:
		return RecipeFaraway
	}
}

// RecipeItemView is one row on the recipes page — the store row plus the
// pre-computed readiness so the template can stay dumb ("apply this CSS
// class") and never has to do math.
type RecipeItemView struct {
	store.RecipeStatusRow
	Readiness RecipeReadiness
}

// RecipeCard is one visual card on the recipes page. Cards are rendered in
// the order returned; empty cards are omitted by GroupRecipes.
type RecipeCard struct {
	Title string
	// Tone is a hint for the template to colour the whole card:
	// "success" (green), "warning" (orange), or "" (default). It matches
	// PantryCard.Tone so styles.html can share the same CSS pattern.
	Tone  string
	Items []RecipeItemView
}

// GroupRecipes turns the flat store rows into the ordered cards the recipes
// page renders. The store already returns rows sorted by MissingCount ASC
// then name, so most groupers just partition; only alphabetical needs to
// re-sort.
func GroupRecipes(rows []store.RecipeStatusRow, sortMode RecipeSort) []RecipeCard {
	views := make([]RecipeItemView, len(rows))
	for i, r := range rows {
		views[i] = RecipeItemView{
			RecipeStatusRow: r,
			Readiness:       classifyRecipeReadiness(r.MissingCount),
		}
	}

	if sortMode == RecipeSortAlphabetical {
		return groupRecipesAlphabetical(views)
	}
	return groupRecipesByMissing(views)
}

func groupRecipesByMissing(items []RecipeItemView) []RecipeCard {
	var ready, nearly, faraway []RecipeItemView
	for _, it := range items {
		switch it.Readiness {
		case RecipeReady:
			ready = append(ready, it)
		case RecipeNearly:
			nearly = append(nearly, it)
		default:
			faraway = append(faraway, it)
		}
	}

	// Store already sorts by (missing asc, name asc); partitioning
	// preserves order so we don't need a secondary sort here.

	cards := make([]RecipeCard, 0, 3)
	if len(ready) > 0 {
		cards = append(cards, RecipeCard{Title: "Ready to cook", Tone: "success", Items: ready})
	}
	if len(nearly) > 0 {
		cards = append(cards, RecipeCard{Title: "Almost there (missing 4 or fewer)", Tone: "warning", Items: nearly})
	}
	if len(faraway) > 0 {
		cards = append(cards, RecipeCard{Title: "Everything else", Items: faraway})
	}
	return cards
}

func groupRecipesAlphabetical(items []RecipeItemView) []RecipeCard {
	sorted := append([]RecipeItemView(nil), items...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})
	if len(sorted) == 0 {
		return nil
	}
	return []RecipeCard{{Title: "All recipes", Items: sorted}}
}
