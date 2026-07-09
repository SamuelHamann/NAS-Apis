package handlers

import "net/http"

// ListFoodLocations handles GET /locations.
func (h *Handler) ListFoodLocations(w http.ResponseWriter, r *http.Request) {
	limit, offset := pagination(r)
	locations, err := h.store.ListFoodLocations(r.Context(), limit, offset)
	if err != nil {
		h.respondStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, locations)
}

// CreateFoodLocation handles POST /locations.
func (h *Handler) CreateFoodLocation(w http.ResponseWriter, r *http.Request) {
	var req nameRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if msg := req.validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	location, err := h.store.CreateFoodLocation(r.Context(), req.value())
	if err != nil {
		h.respondStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, location)
}

// GetFoodLocation handles GET /locations/{id}.
func (h *Handler) GetFoodLocation(w http.ResponseWriter, r *http.Request) {
	id, ok := parseInt64Path(w, r, "id")
	if !ok {
		return
	}

	location, err := h.store.GetFoodLocation(r.Context(), id)
	if err != nil {
		h.respondStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, location)
}

// UpdateFoodLocation handles PUT /locations/{id}.
func (h *Handler) UpdateFoodLocation(w http.ResponseWriter, r *http.Request) {
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

	location, err := h.store.UpdateFoodLocation(r.Context(), id, req.value())
	if err != nil {
		h.respondStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, location)
}

// DeleteFoodLocation handles DELETE /locations/{id}.
func (h *Handler) DeleteFoodLocation(w http.ResponseWriter, r *http.Request) {
	id, ok := parseInt64Path(w, r, "id")
	if !ok {
		return
	}

	if err := h.store.DeleteFoodLocation(r.Context(), id); err != nil {
		h.respondStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
