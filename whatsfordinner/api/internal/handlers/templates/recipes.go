package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
)

// recipeRequest is the JSON body accepted when creating or updating a recipe.
type recipeRequest struct {
	Name            string  `json:"name"`
	Description     *string `json:"description"`
	Instructions    *string `json:"instructions"`
	SourceURL       *string `json:"source_url"`
	Servings        *int32  `json:"servings"`
	PrepTimeMinutes *int32  `json:"prep_time_minutes"`
	CookTimeMinutes *int32  `json:"cook_time_minutes"`
}

func (req recipeRequest) validate() string {
	switch {
	case strings.TrimSpace(req.Name) == "":
		return "name is required"
	case req.Servings != nil && *req.Servings <= 0:
		return "servings must be greater than 0"
	case req.PrepTimeMinutes != nil && *req.PrepTimeMinutes < 0:
		return "prep_time_minutes must be 0 or greater"
	case req.CookTimeMinutes != nil && *req.CookTimeMinutes < 0:
		return "cook_time_minutes must be 0 or greater"
	default:
		return ""
	}
}

func (req recipeRequest) toInput() store.RecipeInput {
	return store.RecipeInput{
		Name:            strings.TrimSpace(req.Name),
		Description:     req.Description,
		Instructions:    req.Instructions,
		SourceURL:       req.SourceURL,
		Servings:        req.Servings,
		PrepTimeMinutes: req.PrepTimeMinutes,
		CookTimeMinutes: req.CookTimeMinutes,
	}
}

// ListRecipes handles GET /recipes.
func (h *Handler) ListRecipes(w http.ResponseWriter, r *http.Request) {
	limit, offset := pagination(r)
	recipes, err := h.store.ListRecipes(r.Context(), limit, offset)
	if err != nil {
		h.respondStoreError(w, err)
		return
	}

	ts, ok := h.templatesCache["past_cooked.html"]
	if !ok {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}
	ts.Execute(w, recipes)

}

// CreateRecipe handles POST /recipes.
func (h *Handler) CreateRecipe(w http.ResponseWriter, r *http.Request) {
	var req recipeRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if msg := req.validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	recipe, err := h.store.CreateRecipe(r.Context(), req.toInput())
	if err != nil {
		h.respondStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, recipe)
}

// GetRecipe handles GET /recipes/{id}.
func (h *Handler) GetRecipe(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDPath(w, r, "id")
	if !ok {
		return
	}

	recipe, err := h.store.GetRecipe(r.Context(), id)
	if err != nil {
		h.respondStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, recipe)
}

// UpdateRecipe handles PUT /recipes/{id}.
func (h *Handler) UpdateRecipe(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDPath(w, r, "id")
	if !ok {
		return
	}

	var req recipeRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if msg := req.validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	recipe, err := h.store.UpdateRecipe(r.Context(), id, req.toInput())
	if err != nil {
		h.respondStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, recipe)
}

// DeleteRecipe handles DELETE /recipes/{id}.
func (h *Handler) DeleteRecipe(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDPath(w, r, "id")
	if !ok {
		return
	}

	if err := h.store.DeleteRecipe(r.Context(), id); err != nil {
		h.respondStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListCookableRecipes handles GET /recipes/cookable.
// It returns recipes whose every ingredient is present in the pantry
// (quantity > 0). All query parameters are optional:
//
//	?tags=1&tags=2  – recipe must carry ALL of the given tag IDs
//	?max_prep=30    – prep_time_minutes ≤ value
//	?max_cook=60    – cook_time_minutes ≤ value
//	?max_total=90   – prep + cook ≤ value (both must be set on the recipe)
//
// Recipes with a NULL time column are excluded when the corresponding time
// filter is set. Supports ?limit and ?offset for pagination.
func (h *Handler) ListCookableRecipes(w http.ResponseWriter, r *http.Request) {
	limit, offset := pagination(r)
	q := r.URL.Query()

	var filter store.CookableFilter

	// Parse optional tag IDs — repeated param: ?tags=1&tags=2.
	for _, raw := range q["tags"] {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			writeError(w, http.StatusBadRequest, "invalid tag id: "+raw)
			return
		}
		filter.TagIDs = append(filter.TagIDs, id)
	}

	// parseMinutes parses an optional non-negative integer query parameter
	// and writes a 400 response on invalid input.
	parseMinutes := func(key string) (*int32, bool) {
		raw := q.Get(key)
		if raw == "" {
			return nil, true
		}
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || n < 0 {
			writeError(w, http.StatusBadRequest, key+" must be a non-negative integer")
			return nil, false
		}
		v := int32(n)
		return &v, true
	}

	var ok bool
	if filter.MaxPrep, ok = parseMinutes("max_prep"); !ok {
		return
	}
	if filter.MaxCook, ok = parseMinutes("max_cook"); !ok {
		return
	}
	if filter.MaxTotal, ok = parseMinutes("max_total"); !ok {
		return
	}

	recipes, err := h.store.ListCookableRecipes(r.Context(), filter, limit, offset)
	if err != nil {
		h.respondStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, recipes)
}
