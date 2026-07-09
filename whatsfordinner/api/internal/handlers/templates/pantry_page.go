package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
)

// PantryPageData is the view model for templates/pantry.html.
type PantryPageData struct {
	PageData

	// SelectedPantry is the pantry currently being displayed. Nil while
	// the DB has no pantries at all (the page then renders a "create your
	// first pantry" empty state).
	SelectedPantry *models.Pantry
	// Pantries is every pantry that exists — populates the pantry-picker
	// dropdown at the top of the page.
	Pantries []models.Pantry

	// Cards is the grouped, colour-coded list to render. Always contains at
	// least one card unless the pantry is empty (in which case Cards is nil
	// and the template shows an empty state).
	Cards []PantryCard

	// UI state -----------------------------------------------------------
	Sort           PantrySort
	SelectedTagIDs map[int64]bool // fast lookup for pre-checking filter checkboxes

	// Form data ---------------------------------------------------------
	Ingredients []models.Ingredient
	Units       []models.Unit
	Locations   []models.FoodLocation
	Tags        []models.Tag

	Error string
}

// PantryPage renders GET /pantry: the pantry contents for the selected
// pantry, grouped/coloured according to the ?sort= query param, filtered by
// any ?tags= query params.
//
// The pantry-picker dropdown is scoped to the signed-in user via the
// user_pantry join table. When the user has no memberships (fresh install
// bootstrap) or no one is signed in we fall back to every pantry so the UI
// isn't a dead end.
func (h *Handler) PantryPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	pantries, err := h.listVisiblePantries(r)
	if err != nil {
		h.logger.Error("list pantries", "error", err)
		http.Error(w, "failed to load pantries", http.StatusInternalServerError)
		return
	}

	data := PantryPageData{
		PageData: h.newPageData(r, "Pantry"),
		Pantries: pantries,
		Sort:     parsePantrySort(r.URL.Query().Get("sort")),
		Error:    r.URL.Query().Get("error"),
	}

	// Pantry picker: honour ?pantry_id=..., otherwise fall back to the
	// first pantry (placeholder until a session-backed picker exists on the
	// home page).
	selected := pickSelectedPantry(pantries, r.URL.Query().Get("pantry_id"))
	if selected != nil {
		data.SelectedPantry = selected
	}

	// Build the tag filter set from ?tags=1&tags=2 (repeated param).
	filter := store.PantryFilter{}
	data.SelectedTagIDs = map[int64]bool{}
	for _, raw := range r.URL.Query()["tags"] {
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil && id > 0 {
			filter.TagIDs = append(filter.TagIDs, id)
			data.SelectedTagIDs[id] = true
		}
	}

	// Load form dropdowns and the tag filter list. These tables are small
	// (household-scale — a few dozen entries each), so we pull the full
	// list once per page render and rely on the store's maxPageSize (200)
	// as a safety cap. Ask for the cap explicitly rather than the default
	// page size (50) so all options show up in the dropdowns.
	const dropdownPageSize = 200
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
	if data.Locations, err = h.store.ListFoodLocations(ctx, dropdownPageSize, 0); err != nil {
		h.logger.Error("list locations", "error", err)
		http.Error(w, "failed to load form data", http.StatusInternalServerError)
		return
	}
	if data.Tags, err = h.store.ListTags(ctx, dropdownPageSize, 0); err != nil {
		h.logger.Error("list tags", "error", err)
		http.Error(w, "failed to load form data", http.StatusInternalServerError)
		return
	}

	if selected != nil {
		rows, err := h.store.ListPantryDetailed(ctx, selected.ID, filter)
		if err != nil {
			h.logger.Error("list pantry", "error", err)
			http.Error(w, "failed to load pantry", http.StatusInternalServerError)
			return
		}
		data.Cards = GroupPantry(rows, data.Sort, time.Now())
	}

	ts, ok := h.templatesCache["pantry.html"]
	if !ok {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	if err := ts.Execute(w, data); err != nil {
		h.logger.Error("render template", "template", "pantry.html", "error", err)
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}

// pickSelectedPantry returns the pantry matching the ?pantry_id=... query
// param, or the first pantry as a fallback, or nil when there are none.
func pickSelectedPantry(pantries []models.Pantry, raw string) *models.Pantry {
	if len(pantries) == 0 {
		return nil
	}
	if raw != "" {
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
			for i := range pantries {
				if pantries[i].ID == id {
					return &pantries[i]
				}
			}
		}
	}
	return &pantries[0]
}

// listVisiblePantries returns the pantries the current request is allowed
// to see: the signed-in user's memberships when they have any, otherwise
// every pantry as a fallback. That fallback matters in two cases:
//
//  1. No one is signed in (the whole "sign-in" flow is optional on this
//     shared household device, so browsing /pantry without a cookie should
//     still work).
//  2. The user is signed in but has zero user_pantry rows yet — a fresh
//     install where the join table hasn't been seeded. Refusing to show
//     anything here would strand the user with no way to bootstrap.
func (h *Handler) listVisiblePantries(r *http.Request) ([]models.Pantry, error) {
	ctx := r.Context()

	userID, ok := sessionUserID(r)
	if !ok {
		return h.store.ListPantries(ctx)
	}

	pantries, err := h.store.ListPantriesForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(pantries) > 0 {
		return pantries, nil
	}
	// Bootstrap fallback: the user has no memberships yet, so show every
	// pantry rather than an empty picker. Once a "manage pantry members"
	// UI exists this fallback can go away.
	return h.store.ListPantries(ctx)
}

// PantryCreate handles POST /pantry: creates a pantry entry from the "+ Add
// item" form and redirects back to /pantry preserving the current view.
func (h *Handler) PantryCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.redirectToPantry(w, r, "invalid form submission")
		return
	}

	in, msg := parsePantryForm(r.Form)
	if msg != "" {
		h.redirectToPantry(w, r, msg)
		return
	}

	if _, err := h.store.CreatePantryItem(r.Context(), in); err != nil {
		h.redirectToPantry(w, r, humanPantryStoreError(err))
		return
	}

	// Preserve the current sort/filter/pantry when redirecting back.
	http.Redirect(w, r, pantryRedirectURL(r, ""), http.StatusSeeOther)
}

// PantryUpdate handles POST /pantry/{id}/update.
func (h *Handler) PantryUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDPathHTML(r, "id")
	if !ok {
		h.redirectToPantry(w, r, "invalid pantry entry")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.redirectToPantry(w, r, "invalid form submission")
		return
	}

	in, msg := parsePantryForm(r.Form)
	if msg != "" {
		h.redirectToPantry(w, r, msg)
		return
	}

	if _, err := h.store.UpdatePantryItem(r.Context(), id, in); err != nil {
		h.redirectToPantry(w, r, humanPantryStoreError(err))
		return
	}
	http.Redirect(w, r, pantryRedirectURL(r, ""), http.StatusSeeOther)
}

// PantryDelete handles POST /pantry/{id}/delete.
func (h *Handler) PantryDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDPathHTML(r, "id")
	if !ok {
		h.redirectToPantry(w, r, "invalid pantry entry")
		return
	}
	if err := h.store.DeletePantryItem(r.Context(), id); err != nil {
		h.redirectToPantry(w, r, humanPantryStoreError(err))
		return
	}
	http.Redirect(w, r, pantryRedirectURL(r, ""), http.StatusSeeOther)
}

// redirectToPantry sends the browser back to /pantry with the given error
// message and preserves the sort/filter/pantry query params from the
// current request so the user doesn't lose their view state after a
// validation error.
func (h *Handler) redirectToPantry(w http.ResponseWriter, r *http.Request, message string) {
	http.Redirect(w, r, pantryRedirectURL(r, message), http.StatusSeeOther)
}

// pantryRedirectURL builds "/pantry?<preserved params>[&error=...]".
func pantryRedirectURL(r *http.Request, message string) string {
	// The form encodes the currently active view as hidden fields
	// (pantry_id, sort, tags), so pull them from r.Form first. Fall back to
	// the URL query for GET requests where no form was submitted.
	values := r.URL.Query()
	if r.Form != nil {
		for _, k := range []string{"pantry_id", "sort"} {
			if v := r.Form.Get(k); v != "" {
				values.Set(k, v)
			}
		}
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
		return "/pantry"
	}
	return "/pantry?" + q
}

// parsePantryForm reads a create/update form and returns either a ready
// PantryInput or a user-facing validation message (never both).
func parsePantryForm(form map[string][]string) (store.PantryInput, string) {
	get := func(k string) string { return strings.TrimSpace(firstNonEmpty(form[k])) }

	pantryID, err := strconv.ParseInt(get("pantry_id"), 10, 64)
	if err != nil || pantryID <= 0 {
		return store.PantryInput{}, "please pick a pantry"
	}

	ingredientID, err := strconv.ParseInt(get("ingredient_id"), 10, 64)
	if err != nil || ingredientID <= 0 {
		return store.PantryInput{}, "please pick an ingredient"
	}

	unitID, err := strconv.ParseInt(get("unit_id"), 10, 64)
	if err != nil || unitID <= 0 {
		return store.PantryInput{}, "please pick a unit"
	}

	quantity, err := strconv.ParseFloat(get("quantity"), 64)
	if err != nil || quantity < 0 {
		return store.PantryInput{}, "quantity must be zero or a positive number"
	}

	in := store.PantryInput{
		PantryID:     pantryID,
		IngredientID: ingredientID,
		UnitID:       unitID,
		Quantity:     quantity,
		IsQuantified: true, // sensible default; HTML form doesn't expose the toggle yet
	}

	if raw := get("location_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			return store.PantryInput{}, "invalid location"
		}
		in.LocationID = &id
	}

	if raw := get("note"); raw != "" {
		in.Note = &raw
	}

	if raw := get("expiration_date"); raw != "" {
		// HTML <input type="date"> gives ISO YYYY-MM-DD in the browser TZ;
		// pgx's DATE binding is TZ-agnostic so we can parse in UTC.
		t, err := time.Parse("2006-01-02", raw)
		if err != nil {
			return store.PantryInput{}, "invalid expiration date (expected YYYY-MM-DD)"
		}
		in.ExpirationDate = &t
	}

	return in, ""
}

// firstNonEmpty returns the first non-empty value in xs, or "" if there
// isn't one. Handy for pulling a single value out of the map[string][]string
// returned by parsed forms.
func firstNonEmpty(xs []string) string {
	for _, x := range xs {
		if x != "" {
			return x
		}
	}
	return ""
}

// parseUUIDPathHTML is the HTML-flow variant of parseUUIDPath: it returns
// (id, false) instead of writing an error response, so the caller can
// redirect with a flash message.
func parseUUIDPathHTML(r *http.Request, name string) (uuid.UUID, bool) {
	parsed, err := uuid.Parse(r.PathValue(name))
	if err != nil {
		return uuid.Nil, false
	}
	return parsed, true
}

// humanPantryStoreError maps a store error to a user-facing message.
func humanPantryStoreError(err error) string {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return "pantry entry not found"
	case errors.Is(err, store.ErrConflict):
		return "that ingredient is already in a pantry"
	case errors.Is(err, store.ErrReference):
		return "one of the picked ingredient/unit/location no longer exists"
	default:
		return "something went wrong, please try again"
	}
}
