package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
)

// cookFormState is the parsed "Cook this recipe" dialog submission, kept
// around (rather than discarded after validation) so a redirect back to the
// recipe page — on a validation error, or a pending missing-ingredient
// confirmation — can re-open the dialog with the same choices instead of
// resetting it.
type cookFormState struct {
	PantryID       int64
	Multiplier     float64
	MultiplierRaw  string // preserved verbatim for redisplay on error
	CreateCombined bool
	Confirmed      bool
	MissingNames   []string

	// CombinedQuantity/CombinedUnitID are the combined ingredient's own
	// top-level amount, only parsed (and only meaningful) when
	// CreateCombined is true.
	CombinedQuantity    float64
	CombinedQuantityRaw string
	CombinedUnitID      *int64
}

// RecipeCook handles POST /recipes/{id}/cook: decrements the chosen pantry's
// stock for the recipe's ingredients (scaled by the submitted multiplier,
// floored at zero), records/updates past_cooked_recipes, and optionally
// folds the ingredients into a combined ingredient named after the recipe.
//
// If any ingredient is missing from the chosen pantry (zero stock) and the
// submission isn't already marked "confirmed", nothing is written yet — the
// browser is redirected back to the recipe page with the missing-ingredient
// names attached, which re-opens the dialog showing a "Cook anyway" warning
// (see templates/recipe_detail.html and RecipeDetailPage's cook_* query
// params). This mirrors every other validation flow in this app: redirect
// with enough state to redisplay the form, rather than an AJAX round trip.
func (h *Handler) RecipeCook(w http.ResponseWriter, r *http.Request) {
	id, ok := parseInt64PathHTML(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}

	recipe, err := h.store.GetRecipe(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		h.logger.Error("get recipe", "error", err)
		http.Error(w, "failed to load recipe", http.StatusInternalServerError)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.redirectToRecipeCook(w, r, id, cookFormState{}, "invalid form submission")
		return
	}

	state, msg := parseCookForm(r.Form)
	if msg != "" {
		h.redirectToRecipeCook(w, r, id, state, msg)
		return
	}

	if !state.Confirmed {
		rows, err := h.store.ListRecipeIngredients(r.Context(), id, state.PantryID)
		if err != nil {
			h.logger.Error("list recipe ingredients", "error", err)
			http.Error(w, "failed to load recipe", http.StatusInternalServerError)
			return
		}
		var missing []string
		for _, row := range rows {
			if row.Missing {
				missing = append(missing, row.IngredientName)
			}
		}
		if len(missing) > 0 {
			state.MissingNames = missing
			h.redirectToRecipeCook(w, r, id, state, "")
			return
		}
	}

	if _, err := h.store.CookRecipe(r.Context(), store.CookRecipeInput{
		RecipeID:         id,
		RecipeName:       recipe.Name,
		PantryID:         state.PantryID,
		Multiplier:       state.Multiplier,
		CreateCombined:   state.CreateCombined,
		CombinedQuantity: state.CombinedQuantity,
		CombinedUnitID:   state.CombinedUnitID,
	}); err != nil {
		h.redirectToRecipeCook(w, r, id, state, humanCookStoreError(err))
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/recipes/%d?pantry_id=%d", id, state.PantryID), http.StatusSeeOther)
}

// redirectToRecipeCook sends the browser back to /recipes/{id} carrying
// enough of state to re-open the cook dialog exactly as it was submitted
// (pantry, multiplier, checkbox, any pending missing-ingredient list) plus
// an optional error message.
func (h *Handler) redirectToRecipeCook(w http.ResponseWriter, r *http.Request, recipeID int64, state cookFormState, message string) {
	values := url.Values{}
	if state.PantryID > 0 {
		values.Set("pantry_id", strconv.FormatInt(state.PantryID, 10))
	}
	if state.MultiplierRaw != "" {
		values.Set("cook_multiplier", state.MultiplierRaw)
	}
	if state.CreateCombined {
		values.Set("cook_combined", "on")
	}
	if state.CombinedQuantityRaw != "" {
		values.Set("cook_combined_quantity", state.CombinedQuantityRaw)
	}
	if state.CombinedUnitID != nil {
		values.Set("cook_combined_unit_id", strconv.FormatInt(*state.CombinedUnitID, 10))
	}
	if len(state.MissingNames) > 0 {
		values.Set("cook_missing", strings.Join(state.MissingNames, ","))
	}
	if message != "" {
		values.Set("error", message)
	}

	location := fmt.Sprintf("/recipes/%d", recipeID)
	if q := values.Encode(); q != "" {
		location += "?" + q
	}
	http.Redirect(w, r, location, http.StatusSeeOther)
}

// parseCookForm reads the cook-dialog submission and returns either a ready
// cookFormState or a user-facing validation message (never both, though the
// partially-filled state is still returned alongside a message so its valid
// fields — e.g. a good pantry_id alongside a bad multiplier — can still be
// redisplayed).
func parseCookForm(form map[string][]string) (cookFormState, string) {
	get := func(k string) string { return strings.TrimSpace(firstNonEmpty(form[k])) }

	state := cookFormState{
		CreateCombined: get("create_combined") == "on",
		Confirmed:      get("confirmed") == "true",
	}

	pantryID, err := strconv.ParseInt(get("pantry_id"), 10, 64)
	if err != nil || pantryID <= 0 {
		return state, "please pick a pantry"
	}
	state.PantryID = pantryID

	raw := get("multiplier")
	state.MultiplierRaw = raw
	multiplier, err := strconv.ParseFloat(raw, 64)
	if err != nil || multiplier <= 0 {
		return state, "quantity to cook must be a positive number"
	}
	state.Multiplier = multiplier

	if state.CreateCombined {
		raw := get("combined_quantity")
		state.CombinedQuantityRaw = raw
		quantity, err := strconv.ParseFloat(raw, 64)
		if err != nil || quantity <= 0 {
			return state, "combined ingredient amount must be a positive number"
		}
		state.CombinedQuantity = quantity

		if raw := get("combined_unit_id"); raw != "" {
			unitID, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || unitID <= 0 {
				return state, "invalid combined ingredient unit"
			}
			state.CombinedUnitID = &unitID
		}
	}

	return state, ""
}

// humanCookStoreError maps a CookRecipe store error to a user-facing message.
func humanCookStoreError(err error) string {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return "recipe not found"
	case errors.Is(err, store.ErrReference):
		return "one of this recipe's ingredients or units no longer exists"
	case errors.Is(err, store.ErrConflict):
		return "something went wrong saving the combined ingredient"
	default:
		return "something went wrong, please try again"
	}
}
