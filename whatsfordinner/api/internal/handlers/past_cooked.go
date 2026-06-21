package handlers

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
)

// pastCookedRequest is the JSON body for creating or updating a past-cooked
// entry.
//
//   - recipe_id      — required on create; ignored on update (the linked
//     recipe never changes).
//   - times_cooked   — optional on create (defaults to 1); required on update.
//   - last_cooked_at — optional in both; omit to use the current time.
type pastCookedRequest struct {
	RecipeID     uuid.UUID  `json:"recipe_id"`
	TimesCooked  *int32     `json:"times_cooked"`
	LastCookedAt *time.Time `json:"last_cooked_at"`
}

func (req pastCookedRequest) validateCreate() string {
	if req.RecipeID == uuid.Nil {
		return "recipe_id is required"
	}
	if req.TimesCooked != nil && *req.TimesCooked < 1 {
		return "times_cooked must be >= 1"
	}
	return ""
}

func (req pastCookedRequest) validateUpdate() string {
	if req.TimesCooked == nil {
		return "times_cooked is required"
	}
	if *req.TimesCooked < 1 {
		return "times_cooked must be >= 1"
	}
	return ""
}

func (req pastCookedRequest) toCreateInput() store.PastCookedInput {
	in := store.PastCookedInput{
		RecipeID:     req.RecipeID,
		TimesCooked:  1,
		LastCookedAt: req.LastCookedAt,
	}
	if req.TimesCooked != nil {
		in.TimesCooked = *req.TimesCooked
	}
	return in
}

func (req pastCookedRequest) toUpdateInput() store.PastCookedInput {
	return store.PastCookedInput{
		TimesCooked:  *req.TimesCooked, // safe: validated non-nil
		LastCookedAt: req.LastCookedAt,
	}
}

// ListPastCooked handles GET /past-cooked.
func (h *Handler) ListPastCooked(w http.ResponseWriter, r *http.Request) {
	limit, offset := pagination(r)
	items, err := h.store.ListPastCooked(r.Context(), limit, offset)
	if err != nil {
		h.respondStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// CreatePastCooked handles POST /past-cooked.
func (h *Handler) CreatePastCooked(w http.ResponseWriter, r *http.Request) {
	var req pastCookedRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if msg := req.validateCreate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	item, err := h.store.CreatePastCooked(r.Context(), req.toCreateInput())
	if err != nil {
		h.respondStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

// GetPastCooked handles GET /past-cooked/{id}.
func (h *Handler) GetPastCooked(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDPath(w, r, "id")
	if !ok {
		return
	}

	item, err := h.store.GetPastCooked(r.Context(), id)
	if err != nil {
		h.respondStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// UpdatePastCooked handles PUT /past-cooked/{id}.
func (h *Handler) UpdatePastCooked(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDPath(w, r, "id")
	if !ok {
		return
	}

	var req pastCookedRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if msg := req.validateUpdate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	item, err := h.store.UpdatePastCooked(r.Context(), id, req.toUpdateInput())
	if err != nil {
		h.respondStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// DeletePastCooked handles DELETE /past-cooked/{id}.
func (h *Handler) DeletePastCooked(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDPath(w, r, "id")
	if !ok {
		return
	}

	if err := h.store.DeletePastCooked(r.Context(), id); err != nil {
		h.respondStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListMostCooked handles GET /past-cooked/most-cooked.
// Returns cooking-history entries ordered by times_cooked descending.
// Supports ?limit and ?offset for pagination.
func (h *Handler) ListMostCooked(w http.ResponseWriter, r *http.Request) {
	limit, offset := pagination(r)
	items, err := h.store.ListMostCooked(r.Context(), limit, offset)
	if err != nil {
		h.respondStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}
