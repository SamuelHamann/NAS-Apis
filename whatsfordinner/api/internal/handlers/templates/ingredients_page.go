package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
)

// IngredientsPageData is the view model for templates/ingredients.html.
type IngredientsPageData struct {
	PageData

	// Items is every ingredient (optionally narrowed by the tag filter),
	// ordered alphabetically, each carrying the names of its own tags.
	Items []store.IngredientDetailRow

	// UI state -----------------------------------------------------------
	SelectedTagIDs map[int64]bool

	// Tags backs both the toolbar's filter chips and the create/edit
	// form's tag checkboxes.
	Tags []models.Tag

	// ActiveTab is "ingredients" (default) or "combined" — which panel the
	// tab switcher shows first on page load (before any client-side JS
	// takes over). See templates/scripts.html's "tabs-script".
	ActiveTab string

	// CombinedItems is every combined ingredient, alphabetical, each with
	// its component items.
	CombinedItems []store.CombinedIngredientDetailRow

	// Ingredients and Units back the combined-ingredient form's dropdowns
	// (one ingredient/unit picker per component-item row, plus the
	// combined ingredient's own unit picker).
	Ingredients []models.Ingredient
	Units       []models.Unit

	Error string
}

// IngredientsPage renders GET /ingredients: every canonical ingredient,
// alphabetical, with a live search box and a tag filter, plus inline CRUD.
func (h *Handler) IngredientsPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	data := IngredientsPageData{
		PageData:  h.newPageData(r, "Ingredients", "ingredients"),
		Error:     r.URL.Query().Get("error"),
		ActiveTab: "ingredients",
	}
	if r.URL.Query().Get("tab") == "combined" {
		data.ActiveTab = "combined"
	}

	// Build the tag filter set from ?tags=1&tags=2 (repeated param).
	filter := store.IngredientFilter{}
	data.SelectedTagIDs = map[int64]bool{}
	for _, raw := range r.URL.Query()["tags"] {
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil && id > 0 {
			filter.TagIDs = append(filter.TagIDs, id)
			data.SelectedTagIDs[id] = true
		}
	}

	// Tag/ingredient/unit options for the toolbar and both tabs' create/edit
	// forms. Same size cap as the pantry/recipes pages — a household never
	// has hundreds of tags/ingredients/units.
	const dropdownPageSize = 200
	var err error
	if data.Tags, err = h.store.ListTags(ctx, dropdownPageSize, 0); err != nil {
		h.logger.Error("list tags", "error", err)
		http.Error(w, "failed to load tags", http.StatusInternalServerError)
		return
	}
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

	if data.Items, err = h.store.ListIngredientsDetailed(ctx, filter); err != nil {
		h.logger.Error("list ingredients", "error", err)
		http.Error(w, "failed to load ingredients", http.StatusInternalServerError)
		return
	}

	if data.CombinedItems, err = h.store.ListCombinedIngredientsDetailed(ctx); err != nil {
		h.logger.Error("list combined ingredients", "error", err)
		http.Error(w, "failed to load combined ingredients", http.StatusInternalServerError)
		return
	}

	ts, ok := h.templatesCache["ingredients.html"]
	if !ok {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	if err := ts.Execute(w, data); err != nil {
		h.logger.Error("render template", "template", "ingredients.html", "error", err)
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}

// IngredientsCreate handles POST /ingredients: creates an ingredient (with
// its tags) from the "+ Add ingredient" form and redirects back to
// /ingredients preserving the current tag filter.
func (h *Handler) IngredientsCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.redirectToIngredients(w, r, "invalid form submission")
		return
	}

	in, msg := parseIngredientForm(r.Form)
	if msg != "" {
		h.redirectToIngredients(w, r, msg)
		return
	}

	if _, err := h.store.CreateIngredientWithTags(r.Context(), in); err != nil {
		h.redirectToIngredients(w, r, humanIngredientStoreError(err))
		return
	}
	http.Redirect(w, r, ingredientsRedirectURL(r, ""), http.StatusSeeOther)
}

// IngredientsUpdate handles POST /ingredients/{id}/update.
func (h *Handler) IngredientsUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := parseInt64PathHTML(r, "id")
	if !ok {
		h.redirectToIngredients(w, r, "invalid ingredient")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.redirectToIngredients(w, r, "invalid form submission")
		return
	}

	in, msg := parseIngredientForm(r.Form)
	if msg != "" {
		h.redirectToIngredients(w, r, msg)
		return
	}

	if _, err := h.store.UpdateIngredientWithTags(r.Context(), id, in); err != nil {
		h.redirectToIngredients(w, r, humanIngredientStoreError(err))
		return
	}
	http.Redirect(w, r, ingredientsRedirectURL(r, ""), http.StatusSeeOther)
}

// IngredientsDelete handles POST /ingredients/{id}/delete.
func (h *Handler) IngredientsDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseInt64PathHTML(r, "id")
	if !ok {
		h.redirectToIngredients(w, r, "invalid ingredient")
		return
	}
	if err := h.store.DeleteIngredient(r.Context(), id); err != nil {
		h.redirectToIngredients(w, r, humanIngredientStoreError(err))
		return
	}
	http.Redirect(w, r, ingredientsRedirectURL(r, ""), http.StatusSeeOther)
}

// redirectToIngredients sends the browser back to /ingredients with the
// given error message, preserving the tag filter from the current request.
func (h *Handler) redirectToIngredients(w http.ResponseWriter, r *http.Request, message string) {
	http.Redirect(w, r, ingredientsRedirectURL(r, message), http.StatusSeeOther)
}

// ingredientsRedirectURL builds "/ingredients?<preserved tags>[&error=...]".
func ingredientsRedirectURL(r *http.Request, message string) string {
	values := r.URL.Query()
	if r.Form != nil {
		if tags, ok := r.Form["view_tags"]; ok {
			values.Del("tags")
			for _, t := range tags {
				values.Add("tags", t)
			}
		}
	}
	values.Del("error")
	if message != "" {
		values.Set("error", message)
	}

	q := values.Encode()
	if q == "" {
		return "/ingredients"
	}
	return "/ingredients?" + q
}

// parseIngredientForm reads a create/update form and returns either a ready
// store.IngredientInput or a user-facing validation message (never both).
func parseIngredientForm(form map[string][]string) (store.IngredientInput, string) {
	name := strings.TrimSpace(firstNonEmpty(form["name"]))
	if name == "" {
		return store.IngredientInput{}, "name is required"
	}

	in := store.IngredientInput{Name: name}
	for _, raw := range form["tag_ids"] {
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil && id > 0 {
			in.TagIDs = append(in.TagIDs, id)
		}
	}
	return in, ""
}

// humanIngredientStoreError maps a store error to a user-facing message.
func humanIngredientStoreError(err error) string {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return "ingredient not found"
	case errors.Is(err, store.ErrConflict):
		return "an ingredient with that name already exists"
	case errors.Is(err, store.ErrReference):
		return "that ingredient is still used by a recipe, pantry, or combined ingredient"
	default:
		return "something went wrong, please try again"
	}
}
