package handlers

import "net/http"

// ListIngredients handles GET /ingredients.
func (h *Handler) ListIngredients(w http.ResponseWriter, r *http.Request) {
	limit, offset := pagination(r)
	ingredients, err := h.store.ListIngredients(r.Context(), limit, offset)
	if err != nil {
		h.respondStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ingredients)
}

// CreateIngredient handles POST /ingredients.
func (h *Handler) CreateIngredient(w http.ResponseWriter, r *http.Request) {
	var req nameRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if msg := req.validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	ingredient, err := h.store.CreateIngredient(r.Context(), req.value())
	if err != nil {
		h.respondStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ingredient)
}

// GetIngredient handles GET /ingredients/{id}.
func (h *Handler) GetIngredient(w http.ResponseWriter, r *http.Request) {
	id, ok := parseInt64Path(w, r, "id")
	if !ok {
		return
	}

	ingredient, err := h.store.GetIngredient(r.Context(), id)
	if err != nil {
		h.respondStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ingredient)
}

// UpdateIngredient handles PUT /ingredients/{id}.
func (h *Handler) UpdateIngredient(w http.ResponseWriter, r *http.Request) {
	id, ok := parseInt64Path(w, r, "id")
	if !ok {
		return
	}

	var req nameRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if msg := req.validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	ingredient, err := h.store.UpdateIngredient(r.Context(), id, req.value())
	if err != nil {
		h.respondStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ingredient)
}

// DeleteIngredient handles DELETE /ingredients/{id}.
func (h *Handler) DeleteIngredient(w http.ResponseWriter, r *http.Request) {
	id, ok := parseInt64Path(w, r, "id")
	if !ok {
		return
	}

	if err := h.store.DeleteIngredient(r.Context(), id); err != nil {
		h.respondStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
