package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
)

// RecipeDetailPageData is the view model for templates/recipe_detail.html.
type RecipeDetailPageData struct {
	PageData

	Recipe models.Recipe
	Tags   []string
	// Author is who created this recipe, nil for seed data / recipes
	// created before this feature existed.
	Author *string

	// UserCollections is the signed-in user's own collections, backing the
	// "add to collection" widget. Empty while signed out or before they've
	// created one.
	UserCollections []models.Collection
	// CollectionMembership[collectionID] reports whether this recipe is
	// already in that collection — pre-checks the widget's boxes. Only
	// ever covers UserCollections' IDs.
	CollectionMembership map[int64]bool

	// CurrentURL is this request's full path+query, round-tripped through
	// the "add to collection" toggle form (redirect_to) so toggling a
	// checkbox lands back on this same recipe.
	CurrentURL string

	// Ingredients is missing-from-pantry rows first (alphabetical), then
	// in-stock rows (also alphabetical). See OrderRecipeIngredients.
	Ingredients []RecipeIngredientView

	// Instructions is Recipe.Instructions split into one entry per step.
	// See ParseInstructionSteps.
	Instructions []string

	// SelectedPantry drives which pantry the ingredient list is checked
	// against; nil when the household has no pantries yet (every
	// ingredient is then "missing", which is technically correct — there's
	// nothing in stock anywhere).
	SelectedPantry *models.Pantry
	Pantries       []models.Pantry

	// Units backs the combined-ingredient amount/unit fields in the cook
	// dialog, shown when "Create a combined ingredient" is checked.
	Units []models.Unit

	// TimesCooked is how many times this recipe has been cooked (0 if
	// never), shown in parentheses next to "Checked against" on the page.
	// See past_cooked_recipes / GetPastCookedByRecipeID.
	TimesCooked int32

	// Cook-dialog state, populated from query params after a redirect from
	// RecipeCook (a validation error, or a pending missing-ingredient
	// confirmation) — zero values otherwise. See recipe_cook.go.
	Error string
	// CookMultiplier is the raw string typed into the dialog (round-tripped
	// as-is so a bad value redisplays exactly as entered); defaults to "1".
	CookMultiplier     string
	CookCreateCombined bool
	// CookCombinedQuantity/CookCombinedUnitID are the combined ingredient's
	// own top-level amount fields, shown/enabled only while the checkbox
	// above is checked. CookCombinedQuantity defaults to CookMultiplier's
	// value (the two start in sync but are independently editable);
	// CookCombinedUnitID defaults to the "bunch" unit if one exists.
	CookCombinedQuantity string
	CookCombinedUnitID   int64
	// CookMissingNames is non-empty only when a cook attempt found
	// ingredients missing from the chosen pantry and is waiting for the user
	// to confirm "cook anyway" — the dialog auto-opens and shows this list.
	CookMissingNames []string
}

// RecipeDetailPage renders GET /recipes/{id}: full detail for a single
// recipe — name, description, its ingredient list (missing-from-pantry ones
// first and highlighted in red), and step-by-step instructions.
func (h *Handler) RecipeDetailPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, ok := parseInt64PathHTML(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}

	recipe, err := h.store.GetRecipe(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		h.logger.Error("get recipe", "error", err)
		http.Error(w, "failed to load recipe", http.StatusInternalServerError)
		return
	}

	pantries, err := h.listVisiblePantries(r)
	if err != nil {
		h.logger.Error("list pantries", "error", err)
		http.Error(w, "failed to load pantries", http.StatusInternalServerError)
		return
	}
	selected := pickSelectedPantry(pantries, r.URL.Query().Get("pantry_id"))

	var pantryID int64
	if selected != nil {
		pantryID = selected.ID
	}

	rows, err := h.store.ListRecipeIngredients(ctx, id, pantryID)
	if err != nil {
		h.logger.Error("list recipe ingredients", "error", err)
		http.Error(w, "failed to load recipe", http.StatusInternalServerError)
		return
	}

	tags, err := h.store.ListRecipeTagNames(ctx, id)
	if err != nil {
		h.logger.Error("list recipe tags", "error", err)
		http.Error(w, "failed to load recipe", http.StatusInternalServerError)
		return
	}

	author, err := h.store.GetRecipeAuthorUsername(ctx, id)
	if err != nil {
		h.logger.Error("get recipe author", "error", err)
		http.Error(w, "failed to load recipe", http.StatusInternalServerError)
		return
	}

	// TimesCooked stays 0 (its zero value) when the recipe has never been
	// cooked — ErrNotFound just means no past_cooked_recipes row exists yet,
	// not a real error.
	var timesCooked int32
	pastCooked, err := h.store.GetPastCookedByRecipeID(ctx, id)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		h.logger.Error("get past cooked", "error", err)
		http.Error(w, "failed to load recipe", http.StatusInternalServerError)
		return
	} else if err == nil {
		timesCooked = pastCooked.TimesCooked
	}

	var instructions []string
	if recipe.Instructions != nil {
		instructions = ParseInstructionSteps(*recipe.Instructions)
	}

	// Dropdown data for the cook dialog's combined-ingredient amount/unit
	// fields. Same size cap as the ingredients/pantry pages — a household
	// never has hundreds of units.
	const dropdownPageSize = 200
	units, err := h.store.ListUnits(ctx, dropdownPageSize, 0)
	if err != nil {
		h.logger.Error("list units", "error", err)
		http.Error(w, "failed to load form data", http.StatusInternalServerError)
		return
	}

	data := RecipeDetailPageData{
		PageData:       h.newPageData(r, recipe.Name, "recipes"),
		Recipe:         recipe,
		Tags:           tags,
		Author:         author,
		Ingredients:    OrderRecipeIngredients(rows),
		Instructions:   instructions,
		SelectedPantry: selected,
		Pantries:       pantries,
		Units:          units,
		TimesCooked:    timesCooked,
		CurrentURL:     r.URL.RequestURI(),

		Error:              r.URL.Query().Get("error"),
		CookMultiplier:     r.URL.Query().Get("cook_multiplier"),
		CookCreateCombined: r.URL.Query().Get("cook_combined") == "on",
	}

	// Collections + membership for this one recipe: only for the signed-in
	// user, and only their own collections.
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
		data.CollectionMembership = map[int64]bool{}
		for _, p := range pairs {
			if p.RecipeID == id {
				data.CollectionMembership[p.CollectionID] = true
			}
		}
	}

	if data.CookMultiplier == "" {
		data.CookMultiplier = "1"
	}
	if raw := r.URL.Query().Get("cook_missing"); raw != "" {
		data.CookMissingNames = strings.Split(raw, ",")
	}

	// The combined ingredient's own amount defaults to the same value as
	// "quantity to cook" (see CookCombinedQuantity's doc comment above).
	data.CookCombinedQuantity = r.URL.Query().Get("cook_combined_quantity")
	if data.CookCombinedQuantity == "" {
		data.CookCombinedQuantity = data.CookMultiplier
	}
	if raw := r.URL.Query().Get("cook_combined_unit_id"); raw != "" {
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
			data.CookCombinedUnitID = id
		}
	} else {
		for _, u := range units {
			if strings.EqualFold(u.Name, "bunch") {
				data.CookCombinedUnitID = u.ID
				break
			}
		}
	}

	ts, ok := h.templatesCache["recipe_detail.html"]
	if !ok {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	if err := ts.Execute(w, data); err != nil {
		h.logger.Error("render template", "template", "recipe_detail.html", "error", err)
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}
