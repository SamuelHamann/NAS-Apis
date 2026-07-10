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
		Ingredients:    OrderRecipeIngredients(rows),
		Instructions:   instructions,
		SelectedPantry: selected,
		Pantries:       pantries,
		Units:          units,

		Error:              r.URL.Query().Get("error"),
		CookMultiplier:     r.URL.Query().Get("cook_multiplier"),
		CookCreateCombined: r.URL.Query().Get("cook_combined") == "on",
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
