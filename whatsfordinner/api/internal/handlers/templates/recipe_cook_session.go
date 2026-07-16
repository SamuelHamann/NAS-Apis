package handlers

import (
	"errors"
	"net/http"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
)

// RecipeCookSessionPageData is the view model for
// templates/recipe_cook_session.html.
type RecipeCookSessionPageData struct {
	PageData

	Recipe models.Recipe

	// Ingredients/Instructions mirror RecipeDetailPageData's fields of the
	// same name — see OrderRecipeIngredients/ParseInstructionSteps in
	// recipe_detail_view.go — but here every row/step also gets a checkbox
	// (see templates/recipe_cook_session.html), a pure client-side aid with
	// no bearing on the actual cook.
	Ingredients  []RecipeIngredientView
	Instructions []string

	SelectedPantry *models.Pantry

	// The cook dialog's choices (see recipe_detail.html), carried forward as
	// hidden fields on the "Finish cooking" form so POSTing it performs the
	// exact cook that was configured, without asking again.
	PantryID            int64
	MultiplierRaw       string
	CreateCombined      bool
	CombinedQuantityRaw string
	CombinedUnitID      *int64
}

// RecipeCookSessionPage renders GET /recipes/{id}/cook: a simplified,
// checklist-friendly view of the recipe meant to stay open while actually
// cooking it — every ingredient and instruction step gets its own checkbox
// to tick off. Nothing is written to the pantry here; that only happens once
// "Finish cooking" is pressed, which POSTs the same recipe id to RecipeCook
// (recipe_cook.go).
//
// Reached by submitting the recipe detail page's "Cook this recipe" dialog,
// which now GETs here (see recipe_detail.html) instead of posting straight
// to RecipeCook. This handler reuses parseCookForm and the missing
// -ingredient confirmation flow from recipe_cook.go so a bad submission, or
// ingredients missing from the chosen pantry, redirects back to the recipe
// detail page to reopen that dialog exactly as before — this page never has
// to duplicate that validation.
func (h *Handler) RecipeCookSessionPage(w http.ResponseWriter, r *http.Request) {
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

	state, msg := parseCookForm(r.URL.Query())
	if msg != "" {
		h.redirectToRecipeCook(w, r, id, state, msg)
		return
	}

	rows, err := h.store.ListRecipeIngredients(ctx, id, state.PantryID)
	if err != nil {
		h.logger.Error("list recipe ingredients", "error", err)
		http.Error(w, "failed to load recipe", http.StatusInternalServerError)
		return
	}

	if !state.Confirmed {
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

	pantries, err := h.listVisiblePantries(r)
	if err != nil {
		h.logger.Error("list pantries", "error", err)
		http.Error(w, "failed to load pantries", http.StatusInternalServerError)
		return
	}
	selected := pickSelectedPantry(pantries, r.URL.Query().Get("pantry_id"))

	var instructions []string
	if recipe.Instructions != nil {
		instructions = ParseInstructionSteps(*recipe.Instructions)
	}

	data := RecipeCookSessionPageData{
		PageData:            h.newPageData(r, "Cook "+recipe.Name, "recipes"),
		Recipe:              recipe,
		Ingredients:         OrderRecipeIngredients(rows),
		Instructions:        instructions,
		SelectedPantry:      selected,
		PantryID:            state.PantryID,
		MultiplierRaw:       state.MultiplierRaw,
		CreateCombined:      state.CreateCombined,
		CombinedQuantityRaw: state.CombinedQuantityRaw,
		CombinedUnitID:      state.CombinedUnitID,
	}

	ts, ok := h.templatesCache["recipe_cook_session.html"]
	if !ok {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	if err := ts.Execute(w, data); err != nil {
		h.logger.Error("render template", "template", "recipe_cook_session.html", "error", err)
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}
