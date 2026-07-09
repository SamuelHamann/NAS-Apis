package handlers

import "net/http"

// ListUnits handles GET /units.
func (h *Handler) ListUnits(w http.ResponseWriter, r *http.Request) {
	limit, offset := pagination(r)
	units, err := h.store.ListUnits(r.Context(), limit, offset)
	if err != nil {
		h.respondStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, units)
}

// CreateUnit handles POST /units.
func (h *Handler) CreateUnit(w http.ResponseWriter, r *http.Request) {
	var req nameRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if msg := req.validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	unit, err := h.store.CreateUnit(r.Context(), req.value())
	if err != nil {
		h.respondStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, unit)
}

// GetUnit handles GET /units/{id}.
func (h *Handler) GetUnit(w http.ResponseWriter, r *http.Request) {
	id, ok := parseInt64Path(w, r, "id")
	if !ok {
		return
	}

	unit, err := h.store.GetUnit(r.Context(), id)
	if err != nil {
		h.respondStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, unit)
}

// UpdateUnit handles PUT /units/{id}.
func (h *Handler) UpdateUnit(w http.ResponseWriter, r *http.Request) {
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

	unit, err := h.store.UpdateUnit(r.Context(), id, req.value())
	if err != nil {
		h.respondStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, unit)
}

// DeleteUnit handles DELETE /units/{id}.
func (h *Handler) DeleteUnit(w http.ResponseWriter, r *http.Request) {
	id, ok := parseInt64Path(w, r, "id")
	if !ok {
		return
	}

	if err := h.store.DeleteUnit(r.Context(), id); err != nil {
		h.respondStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
