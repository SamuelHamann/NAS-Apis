package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
)

// RecipeFormPageData is the view model for templates/recipe_form.html,
// shared by both GET /recipes/new (Recipe nil) and GET /recipes/{id}/edit
// (Recipe set) — the template picks the form's action URL/submit label off
// whether Recipe is nil.
type RecipeFormPageData struct {
	PageData

	// Recipe is nil when creating a new recipe.
	Recipe *models.Recipe

	// Items is the recipe's current ingredient rows — empty for a new
	// recipe, or one with none yet. Prefills the repeatable ingredient-row
	// list (see recipe_ingredient_row in templates/recipe_form.html).
	Items []store.RecipeIngredientEditRow

	// Ingredients/Units/Tags back the ingredient picker's dropdown options,
	// each row's unit picker, and the recipe's own tag checkboxes.
	Ingredients []models.Ingredient
	Units       []models.Unit
	Tags        []models.Tag
	// SelectedTagIDs pre-checks the recipe's own tag chips when editing.
	SelectedTagIDs map[int64]bool

	Error string
}

// RecipeNewPage renders GET /recipes/new: a blank recipe form.
func (h *Handler) RecipeNewPage(w http.ResponseWriter, r *http.Request) {
	data := RecipeFormPageData{
		PageData: h.newPageData(r, "New recipe", "recipes"),
		Error:    r.URL.Query().Get("error"),
	}
	h.renderRecipeForm(w, r, data)
}

// RecipeEditPage renders GET /recipes/{id}/edit: the same form, prefilled
// with the recipe's current fields, ingredient list and tags.
func (h *Handler) RecipeEditPage(w http.ResponseWriter, r *http.Request) {
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

	items, err := h.store.ListRecipeIngredientsForEdit(ctx, id)
	if err != nil {
		h.logger.Error("list recipe ingredients", "error", err)
		http.Error(w, "failed to load recipe", http.StatusInternalServerError)
		return
	}

	tagIDs, err := h.store.ListRecipeTagIDs(ctx, id)
	if err != nil {
		h.logger.Error("list recipe tags", "error", err)
		http.Error(w, "failed to load recipe", http.StatusInternalServerError)
		return
	}

	data := RecipeFormPageData{
		PageData:       h.newPageData(r, "Edit "+recipe.Name, "recipes"),
		Recipe:         &recipe,
		Items:          items,
		SelectedTagIDs: make(map[int64]bool, len(tagIDs)),
		Error:          r.URL.Query().Get("error"),
	}
	for _, tagID := range tagIDs {
		data.SelectedTagIDs[tagID] = true
	}

	h.renderRecipeForm(w, r, data)
}

// renderRecipeForm fills in the dropdown data shared by both the new and
// edit pages and renders recipe_form.html.
func (h *Handler) renderRecipeForm(w http.ResponseWriter, r *http.Request, data RecipeFormPageData) {
	ctx := r.Context()

	// Same size cap as the other pages' dropdowns — a household never has
	// hundreds of ingredients/units/tags.
	const dropdownPageSize = 200

	var err error
	if data.Ingredients, err = h.store.ListIngredients(ctx, dropdownPageSize, 0); err != nil {
		h.logger.Error("list ingredients", "error", err)
		http.Error(w, "failed to load form data", http.StatusInternalServerError)
		return
	}
	if data.Units, err = h.store.ListUnits(ctx, dropdownPageSize, 0); err != nil {
		h.logger.Error("list units", "error", err)
		http.Error(w, "failed to load form data", http.StatusInternalServerError)
		return
	}
	if data.Tags, err = h.store.ListTags(ctx, dropdownPageSize, 0); err != nil {
		h.logger.Error("list tags", "error", err)
		http.Error(w, "failed to load form data", http.StatusInternalServerError)
		return
	}
	if data.SelectedTagIDs == nil {
		data.SelectedTagIDs = map[int64]bool{}
	}

	ts, ok := h.templatesCache["recipe_form.html"]
	if !ok {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	if err := ts.Execute(w, data); err != nil {
		h.logger.Error("render template", "template", "recipe_form.html", "error", err)
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}

// RecipeCreate handles POST /recipes: creates a recipe (with its ingredient
// list and tags) and redirects to its detail page.
func (h *Handler) RecipeCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		redirectWithError(w, r, "/recipes/new", "invalid form submission")
		return
	}

	in, msg := parseRecipeForm(r.Form)
	if msg != "" {
		redirectWithError(w, r, "/recipes/new", msg)
		return
	}

	recipe, err := h.store.CreateRecipeWithRelations(r.Context(), in)
	if err != nil {
		redirectWithError(w, r, "/recipes/new", humanRecipeStoreError(err))
		return
	}
	http.Redirect(w, r, "/recipes/"+strconv.FormatInt(recipe.ID, 10), http.StatusSeeOther)
}

// RecipeUpdate handles POST /recipes/{id}/update: overwrites a recipe's
// fields, ingredient list and tags, and redirects to its detail page.
func (h *Handler) RecipeUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := parseInt64PathHTML(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	editURL := "/recipes/" + strconv.FormatInt(id, 10) + "/edit"

	if err := r.ParseForm(); err != nil {
		redirectWithError(w, r, editURL, "invalid form submission")
		return
	}

	in, msg := parseRecipeForm(r.Form)
	if msg != "" {
		redirectWithError(w, r, editURL, msg)
		return
	}

	if _, err := h.store.UpdateRecipeWithRelations(r.Context(), id, in); err != nil {
		redirectWithError(w, r, editURL, humanRecipeStoreError(err))
		return
	}
	http.Redirect(w, r, "/recipes/"+strconv.FormatInt(id, 10), http.StatusSeeOther)
}

// parseRecipeForm reads a create/update form and returns either a ready
// store.RecipeWithRelationsInput or a user-facing validation message (never
// both). Ingredient rows are submitted as repeated same-name fields
// (item_ingredient_id/item_quantity/item_unit_id/item_note), one set per row
// in DOM order — same convention as parseCombinedIngredientForm. Unlike a
// combined ingredient, a recipe is allowed to have zero ingredients (e.g. a
// recipe entered before its ingredient list is filled in).
func parseRecipeForm(form map[string][]string) (store.RecipeWithRelationsInput, string) {
	name := strings.TrimSpace(firstNonEmpty(form["name"]))
	if name == "" {
		return store.RecipeWithRelationsInput{}, "name is required"
	}

	in := store.RecipeWithRelationsInput{RecipeInput: store.RecipeInput{Name: name}}

	if description := strings.TrimSpace(firstNonEmpty(form["description"])); description != "" {
		in.Description = &description
	}
	if instructions := strings.TrimSpace(firstNonEmpty(form["instructions"])); instructions != "" {
		in.Instructions = &instructions
	}
	if sourceURL := strings.TrimSpace(firstNonEmpty(form["source_url"])); sourceURL != "" {
		in.SourceURL = &sourceURL
	}

	servings, msg := parseOptionalNonNegativeInt32(form["servings"], "servings")
	if msg != "" {
		return store.RecipeWithRelationsInput{}, msg
	}
	in.Servings = servings

	prep, msg := parseOptionalNonNegativeInt32(form["prep_time_minutes"], "prep time")
	if msg != "" {
		return store.RecipeWithRelationsInput{}, msg
	}
	in.PrepTimeMinutes = prep

	cook, msg := parseOptionalNonNegativeInt32(form["cook_time_minutes"], "cook time")
	if msg != "" {
		return store.RecipeWithRelationsInput{}, msg
	}
	in.CookTimeMinutes = cook

	for _, raw := range form["tag_ids"] {
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil && id > 0 {
			in.TagIDs = append(in.TagIDs, id)
		}
	}

	ingredientIDs := form["item_ingredient_id"]
	quantities := form["item_quantity"]
	unitIDs := form["item_unit_id"]
	notes := form["item_note"]

	seen := map[int64]bool{}
	for i := range ingredientIDs {
		rawIngredientID := strings.TrimSpace(ingredientIDs[i])
		rawQuantity := strings.TrimSpace(get(quantities, i))
		rawUnitID := strings.TrimSpace(get(unitIDs, i))
		rawNote := strings.TrimSpace(get(notes, i))

		if rawIngredientID == "" && rawQuantity == "" && rawUnitID == "" && rawNote == "" {
			continue // blank row (e.g. "+ Add ingredient" clicked but never filled in)
		}

		ingredientID, err := strconv.ParseInt(rawIngredientID, 10, 64)
		if err != nil || ingredientID <= 0 {
			return store.RecipeWithRelationsInput{}, "each ingredient row needs an ingredient selected"
		}
		if seen[ingredientID] {
			return store.RecipeWithRelationsInput{}, "each ingredient can only appear once in a recipe"
		}
		seen[ingredientID] = true

		item := store.RecipeIngredientInput{IngredientID: ingredientID}
		if rawQuantity != "" {
			q, err := strconv.ParseFloat(rawQuantity, 64)
			if err != nil || q < 0 {
				return store.RecipeWithRelationsInput{}, "ingredient quantity must be zero or a positive number"
			}
			item.Quantity = &q
		}
		if rawUnitID != "" {
			u, err := strconv.ParseInt(rawUnitID, 10, 64)
			if err != nil || u <= 0 {
				return store.RecipeWithRelationsInput{}, "invalid ingredient unit"
			}
			item.UnitID = &u
		}
		if rawNote != "" {
			item.Note = &rawNote
		}
		in.Ingredients = append(in.Ingredients, item)
	}

	return in, ""
}

// parseOptionalNonNegativeInt32 parses form's first non-empty value (if any)
// as a non-negative int32, returning (nil, "") when the field was left
// blank entirely.
func parseOptionalNonNegativeInt32(form []string, label string) (*int32, string) {
	raw := strings.TrimSpace(firstNonEmpty(form))
	if raw == "" {
		return nil, ""
	}
	n, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || n < 0 {
		return nil, label + " must be zero or a positive whole number"
	}
	v := int32(n)
	return &v, ""
}

// humanRecipeStoreError maps a store error from a recipe operation to a
// user-facing message.
func humanRecipeStoreError(err error) string {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return "recipe not found"
	case errors.Is(err, store.ErrReference):
		return "one of the selected ingredients, units or tags no longer exists"
	default:
		return "something went wrong, please try again"
	}
}
