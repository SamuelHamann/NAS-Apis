package handlers

import (
	"errors"
	"net/http"

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

	data := RecipeDetailPageData{
		PageData:       h.newPageData(r, recipe.Name, "recipes"),
		Recipe:         recipe,
		Tags:           tags,
		Ingredients:    OrderRecipeIngredients(rows),
		Instructions:   instructions,
		SelectedPantry: selected,
		Pantries:       pantries,
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
