package handlers

import (
	"net/http"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
)

// pantryRequest is the JSON body accepted when creating or updating a pantry
// stock entry.
type pantryRequest struct {
	IngredientID int64    `json:"ingredient_id"`
	Quantity     *float64 `json:"quantity"`
	UnitID       int64    `json:"unit_id"`
	Note         *string  `json:"note"`
	IsQuantified *bool    `json:"is_quantified"`
	LocationID   *int64   `json:"location_id"` // optional — food_locations FK
}

func (req pantryRequest) validate() string {
	switch {
	case req.IngredientID <= 0:
		return "ingredient_id is required"
	case req.UnitID <= 0:
		return "unit_id is required"
	case req.Quantity == nil:
		return "quantity is required"
	case *req.Quantity < 0:
		return "quantity must be 0 or greater"
	default:
		return ""
	}
}

func (req pantryRequest) toInput() store.PantryInput {
	isQuantified := true
	if req.IsQuantified != nil {
		isQuantified = *req.IsQuantified
	}
	return store.PantryInput{
		IngredientID: req.IngredientID,
		Quantity:     *req.Quantity,
		UnitID:       req.UnitID,
		Note:         req.Note,
		IsQuantified: isQuantified,
		LocationID:   req.LocationID,
	}
}

// ListPantry handles GET /pantry.
func (h *Handler) ListPantry(w http.ResponseWriter, r *http.Request) {
	limit, offset := pagination(r)
	items, err := h.store.ListPantry(r.Context(), limit, offset)
	if err != nil {
		h.respondStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// CreatePantryItem handles POST /pantry.
func (h *Handler) CreatePantryItem(w http.ResponseWriter, r *http.Request) {
	var req pantryRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if msg := req.validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	item, err := h.store.CreatePantryItem(r.Context(), req.toInput())
	if err != nil {
		h.respondStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

// GetPantryItem handles GET /pantry/{id}.
func (h *Handler) GetPantryItem(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDPath(w, r, "id")
	if !ok {
		return
	}

	item, err := h.store.GetPantryItem(r.Context(), id)
	if err != nil {
		h.respondStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// UpdatePantryItem handles PUT /pantry/{id}.
func (h *Handler) UpdatePantryItem(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDPath(w, r, "id")
	if !ok {
		return
	}

	var req pantryRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if msg := req.validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	item, err := h.store.UpdatePantryItem(r.Context(), id, req.toInput())
	if err != nil {
		h.respondStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// DeletePantryItem handles DELETE /pantry/{id}.
func (h *Handler) DeletePantryItem(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDPath(w, r, "id")
	if !ok {
		return
	}

	if err := h.store.DeletePantryItem(r.Context(), id); err != nil {
		h.respondStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
