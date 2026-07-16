package handlers

import (
	"net/http"
	"sort"
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

	// For the toolbar's tag filter (server-side, full reload — see
	// RecipeStatusFilter.TagIDs). Collection/creator filtering, below, is
	// client-side only (see "checkbox-filter-script" in scripts.html) so
	// there's no equivalent Selected*/query-param plumbing for them: every
	// recipe matching the pantry+tag filters is always rendered, and
	// checking a collection/creator box just hides/shows rows already on
	// the page.
	Tags []models.Tag

	// Authors is every distinct AuthorUsername among the currently rendered
	// recipes (Cards), alphabetical — backs the "filter by creator" dropdown.
	Authors []string

	// UserCollections is the signed-in user's own collections — backs both
	// the toolbar's "filter by collection" dropdown and each row's "add to
	// collection" widget. Empty (not just while signed out) when the user
	// hasn't created one yet.
	UserCollections []models.Collection
	// CollectionMembership[recipeID][collectionID] reports whether that
	// recipe is already in that collection — pre-checks the "add to
	// collection" widget's boxes, and (as a comma-separated string, see the
	// mapKeysCSV template func) backs each row's client-side collection
	// filter. Only ever covers UserCollections' IDs.
	CollectionMembership map[int64]map[int64]bool

	// CurrentURL is this request's full path+query, round-tripped through
	// the "add to collection" toggle forms (redirect_to) so a non-JS
	// fallback submission lands back on the same page instead of a bare
	// /recipes.
	CurrentURL string
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
		PageData:   h.newPageData(r, "Recipes", "recipes"),
		Pantries:   pantries,
		Sort:       parseRecipeSort(r.URL.Query().Get("sort")),
		CurrentURL: r.URL.RequestURI(),
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

	// Collections + membership: only for the signed-in user, and only their
	// own collections — "filter by collection" and "add to collection" are
	// both meaningless without one.
	if data.SignedIn {
		userID, _ := sessionUserID(r)
		if data.UserCollections, err = h.store.ListCollectionsForUser(ctx, userID); err != nil {
			h.logger.Error("list collections", "error", err)
			http.Error(w, "failed to load collections", http.StatusInternalServerError)
			return
		}
		pairs, err := h.store.ListCollectionRecipesForUser(ctx, userID)
		if err != nil {
			h.logger.Error("list collection recipes", "error", err)
			http.Error(w, "failed to load collections", http.StatusInternalServerError)
			return
		}
		data.CollectionMembership = make(map[int64]map[int64]bool, len(pairs))
		for _, p := range pairs {
			if data.CollectionMembership[p.RecipeID] == nil {
				data.CollectionMembership[p.RecipeID] = make(map[int64]bool)
			}
			data.CollectionMembership[p.RecipeID][p.CollectionID] = true
		}
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

	// Distinct authors among these recipes, for the "filter by creator"
	// dropdown — cheap to derive from rows already in hand rather than a
	// separate query.
	seenAuthors := map[string]bool{}
	for _, row := range rows {
		if row.AuthorUsername == nil || seenAuthors[*row.AuthorUsername] {
			continue
		}
		seenAuthors[*row.AuthorUsername] = true
		data.Authors = append(data.Authors, *row.AuthorUsername)
	}
	sort.Strings(data.Authors)

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
