package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
)

// CombinedIngredientsCreate handles POST /combined-ingredients: creates a
// combined ingredient (with its component items) from the "+ Add combined
// ingredient" form and redirects back to /ingredients?tab=combined.
func (h *Handler) CombinedIngredientsCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.redirectToCombinedIngredients(w, r, "invalid form submission")
		return
	}

	in, msg := parseCombinedIngredientForm(r.Form)
	if msg != "" {
		h.redirectToCombinedIngredients(w, r, msg)
		return
	}

	if _, err := h.store.CreateCombinedIngredientWithItems(r.Context(), in); err != nil {
		h.redirectToCombinedIngredients(w, r, humanCombinedIngredientStoreError(err))
		return
	}
	http.Redirect(w, r, combinedIngredientsRedirectURL(""), http.StatusSeeOther)
}

// CombinedIngredientsUpdate handles POST /combined-ingredients/{id}/update.
func (h *Handler) CombinedIngredientsUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := parseInt64PathHTML(r, "id")
	if !ok {
		h.redirectToCombinedIngredients(w, r, "invalid combined ingredient")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.redirectToCombinedIngredients(w, r, "invalid form submission")
		return
	}

	in, msg := parseCombinedIngredientForm(r.Form)
	if msg != "" {
		h.redirectToCombinedIngredients(w, r, msg)
		return
	}

	if _, err := h.store.UpdateCombinedIngredientWithItems(r.Context(), id, in); err != nil {
		h.redirectToCombinedIngredients(w, r, humanCombinedIngredientStoreError(err))
		return
	}
	http.Redirect(w, r, combinedIngredientsRedirectURL(""), http.StatusSeeOther)
}

// CombinedIngredientsDelete handles POST /combined-ingredients/{id}/delete.
func (h *Handler) CombinedIngredientsDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseInt64PathHTML(r, "id")
	if !ok {
		h.redirectToCombinedIngredients(w, r, "invalid combined ingredient")
		return
	}
	if err := h.store.DeleteCombinedIngredient(r.Context(), id); err != nil {
		h.redirectToCombinedIngredients(w, r, humanCombinedIngredientStoreError(err))
		return
	}
	http.Redirect(w, r, combinedIngredientsRedirectURL(""), http.StatusSeeOther)
}

// redirectToCombinedIngredients sends the browser back to
// /ingredients?tab=combined with the given error message.
func (h *Handler) redirectToCombinedIngredients(w http.ResponseWriter, r *http.Request, message string) {
	http.Redirect(w, r, combinedIngredientsRedirectURL(message), http.StatusSeeOther)
}

// combinedIngredientsRedirectURL builds "/ingredients?tab=combined[&error=...]".
// Unlike ingredientsRedirectURL, there's no filter/sort state to preserve for
// this resource.
func combinedIngredientsRedirectURL(message string) string {
	if message == "" {
		return "/ingredients?tab=combined"
	}
	values := make(url.Values)
	values.Set("tab", "combined")
	values.Set("error", message)
	return "/ingredients?" + values.Encode()
}

// parseCombinedIngredientForm reads a create/update form and returns either
// a ready store.CombinedIngredientInput or a user-facing validation message
// (never both).
//
// Component item rows are submitted as repeated same-name fields
// (item_ingredient_id, item_quantity, item_unit_id, item_note), one set per
// row in DOM order — mirroring the existing tag_ids repeated-checkbox
// pattern in ingredient_form. This relies on the item-rows JS removing a
// row's DOM node entirely (not just hiding it) on remove-click, so the four
// slices always stay the same length and index-aligned.
func parseCombinedIngredientForm(form map[string][]string) (store.CombinedIngredientInput, string) {
	name := strings.TrimSpace(firstNonEmpty(form["name"]))
	if name == "" {
		return store.CombinedIngredientInput{}, "name is required"
	}

	quantity, err := strconv.ParseFloat(strings.TrimSpace(firstNonEmpty(form["quantity"])), 64)
	if err != nil || quantity < 0 {
		return store.CombinedIngredientInput{}, "quantity must be zero or a positive number"
	}

	unitID, err := strconv.ParseInt(strings.TrimSpace(firstNonEmpty(form["unit_id"])), 10, 64)
	if err != nil || unitID <= 0 {
		return store.CombinedIngredientInput{}, "please pick a unit"
	}

	in := store.CombinedIngredientInput{
		Name:     name,
		Quantity: quantity,
		UnitID:   &unitID,
	}
	if note := strings.TrimSpace(firstNonEmpty(form["note"])); note != "" {
		in.Note = &note
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
			// Blank row (e.g. "+ Add ingredient" clicked but never filled in).
			continue
		}

		ingredientID, err := strconv.ParseInt(rawIngredientID, 10, 64)
		if err != nil || ingredientID <= 0 {
			return store.CombinedIngredientInput{}, "each ingredient row needs an ingredient selected"
		}
		if seen[ingredientID] {
			return store.CombinedIngredientInput{}, "each component ingredient can only appear once"
		}
		seen[ingredientID] = true

		item := store.CombinedIngredientItemInput{IngredientID: ingredientID}
		if rawQuantity != "" {
			q, err := strconv.ParseFloat(rawQuantity, 64)
			if err != nil || q < 0 {
				return store.CombinedIngredientInput{}, "item quantity must be zero or a positive number"
			}
			item.Quantity = &q
		}
		if rawUnitID != "" {
			u, err := strconv.ParseInt(rawUnitID, 10, 64)
			if err != nil || u <= 0 {
				return store.CombinedIngredientInput{}, "invalid item unit"
			}
			item.UnitID = &u
		}
		if rawNote != "" {
			item.Note = &rawNote
		}
		in.Items = append(in.Items, item)
	}

	if len(in.Items) == 0 {
		return store.CombinedIngredientInput{}, "add at least one ingredient to the bundle"
	}

	return in, ""
}

// get returns xs[i], or "" if i is out of range. Component item rows are
// submitted as parallel same-length slices, but a defensive bounds check
// keeps a malformed submission (e.g. missing one field on one row) from
// panicking instead of failing validation.
func get(xs []string, i int) string {
	if i < 0 || i >= len(xs) {
		return ""
	}
	return xs[i]
}

// humanCombinedIngredientStoreError maps a store error to a user-facing
// message.
func humanCombinedIngredientStoreError(err error) string {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return "combined ingredient not found"
	case errors.Is(err, store.ErrConflict):
		return "a combined ingredient with that name already exists"
	case errors.Is(err, store.ErrReference):
		return "one of the selected ingredients or units no longer exists"
	default:
		return "something went wrong, please try again"
	}
}
