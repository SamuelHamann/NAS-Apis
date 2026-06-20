package handlers

import "net/http"

// ListTags handles GET /tags.
func (h *Handler) ListTags(w http.ResponseWriter, r *http.Request) {
	limit, offset := pagination(r)
	tags, err := h.store.ListTags(r.Context(), limit, offset)
	if err != nil {
		h.respondStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tags)
}

// CreateTag handles POST /tags.
func (h *Handler) CreateTag(w http.ResponseWriter, r *http.Request) {
	var req nameRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if msg := req.validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	tag, err := h.store.CreateTag(r.Context(), req.value())
	if err != nil {
		h.respondStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, tag)
}

// GetTag handles GET /tags/{id}.
func (h *Handler) GetTag(w http.ResponseWriter, r *http.Request) {
	id, ok := parseInt64Path(w, r, "id")
	if !ok {
		return
	}

	tag, err := h.store.GetTag(r.Context(), id)
	if err != nil {
		h.respondStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tag)
}

// UpdateTag handles PUT /tags/{id}.
func (h *Handler) UpdateTag(w http.ResponseWriter, r *http.Request) {
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

	tag, err := h.store.UpdateTag(r.Context(), id, req.value())
	if err != nil {
		h.respondStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tag)
}

// DeleteTag handles DELETE /tags/{id}.
func (h *Handler) DeleteTag(w http.ResponseWriter, r *http.Request) {
	id, ok := parseInt64Path(w, r, "id")
	if !ok {
		return
	}

	if err := h.store.DeleteTag(r.Context(), id); err != nil {
		h.respondStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
