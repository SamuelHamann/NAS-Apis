package handlers

import (
	"net/http"
	"strconv"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
)

// RecipesPageData is the view model for templates/recipes.html.
type RecipesPageData struct {
	PageData

	// Pantry context — the "missing ingredients" sort is meaningful only
	// relative to a specific pantry. When there is no selected pantry, the
	// handler forces the sort to alphabetical and the readiness column is
	// still filled from a zero-pantry query (which reports everything as
	// missing) but is not surfaced in the UI.
	SelectedPantry *models.Pantry
	Pantries       []models.Pantry

	Cards []RecipeCard

	// UI state -----------------------------------------------------------
	Sort           RecipeSort
	SelectedTagIDs map[int64]bool

	// For the toolbar's tag filter.
	Tags []models.Tag
}

// RecipesPage renders GET /recipes: every recipe, grouped and colour-coded
// by how many of its ingredients are missing from the selected pantry (or
// simply alphabetical when there is no pantry).
func (h *Handler) RecipesPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	pantries, err := h.listVisiblePantries(r)
	if err != nil {
		h.logger.Error("list pantries", "error", err)
		http.Error(w, "failed to load pantries", http.StatusInternalServerError)
		return
	}

	data := RecipesPageData{
		PageData: h.newPageData(r, "Recipes", "recipes"),
		Pantries: pantries,
		Sort:     parseRecipeSort(r.URL.Query().Get("sort")),
	}

	// Pantry picker: same rules as the pantry page — honour ?pantry_id, else
	// fall back to the first visible pantry. Nil selection is fine (no
	// pantry exists yet); we just switch to alphabetical below.
	if selected := pickSelectedPantry(pantries, r.URL.Query().Get("pantry_id")); selected != nil {
		data.SelectedPantry = selected
	}

	// Build the tag filter set from ?tags=1&tags=2 (repeated param).
	filter := store.RecipeStatusFilter{}
	data.SelectedTagIDs = map[int64]bool{}
	for _, raw := range r.URL.Query()["tags"] {
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil && id > 0 {
			filter.TagIDs = append(filter.TagIDs, id)
			data.SelectedTagIDs[id] = true
		}
	}

	// Tag options for the toolbar. Same size cap as the pantry page — a
	// household will never have hundreds of tags.
	const dropdownPageSize = 200
	if data.Tags, err = h.store.ListTags(ctx, dropdownPageSize, 0); err != nil {
		h.logger.Error("list tags", "error", err)
		http.Error(w, "failed to load tags", http.StatusInternalServerError)
		return
	}

	// Without a pantry the "missing" sort is meaningless — force
	// alphabetical so the UI matches what actually happens.
	if data.SelectedPantry == nil {
		data.Sort = RecipeSortAlphabetical
	}

	var pantryID int64
	if data.SelectedPantry != nil {
		pantryID = data.SelectedPantry.ID
	}

	rows, err := h.store.ListRecipesWithStatus(ctx, pantryID, filter)
	if err != nil {
		h.logger.Error("list recipes", "error", err)
		http.Error(w, "failed to load recipes", http.StatusInternalServerError)
		return
	}
	data.Cards = GroupRecipes(rows, data.Sort)

	ts, ok := h.templatesCache["recipes.html"]
	if !ok {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	if err := ts.Execute(w, data); err != nil {
		h.logger.Error("render template", "template", "recipes.html", "error", err)
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}
