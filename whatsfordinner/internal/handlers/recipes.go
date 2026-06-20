package handlers

import (
	"net/http"
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
	writeJSON(w, http.StatusOK, recipes)
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
