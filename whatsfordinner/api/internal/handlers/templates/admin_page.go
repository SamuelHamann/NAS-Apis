package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
)

// AdminPageData is the view model for templates/admin.html.
type AdminPageData struct {
	PageData

	Pantries []models.Pantry
	Users    []models.User

	// Access[userID][pantryID] reports whether that user currently has
	// access to that pantry — used to pre-check the access matrix's
	// checkboxes. A nil inner map (user has no rows yet) is safe to index:
	// Go returns the zero value (false) for a missing key.
	Access map[int64]map[int64]bool

	Error string
}

// AdminPage renders GET /settings/admin: create-a-pantry form, plus a
// user x pantry checkbox matrix (which users can see which pantries, via
// the user_pantry join table).
func (h *Handler) AdminPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	data := AdminPageData{
		PageData: h.newPageData(r, "Admin", ""),
		Error:    r.URL.Query().Get("error"),
	}

	var err error
	if data.Pantries, err = h.store.ListPantries(ctx); err != nil {
		h.logger.Error("list pantries", "error", err)
		http.Error(w, "failed to load pantries", http.StatusInternalServerError)
		return
	}
	if data.Users, err = h.store.ListUsers(ctx); err != nil {
		h.logger.Error("list users", "error", err)
		http.Error(w, "failed to load users", http.StatusInternalServerError)
		return
	}

	access, err := h.store.ListUserPantryAccess(ctx)
	if err != nil {
		h.logger.Error("list user pantry access", "error", err)
		http.Error(w, "failed to load pantry access", http.StatusInternalServerError)
		return
	}
	data.Access = make(map[int64]map[int64]bool, len(data.Users))
	for _, a := range access {
		if data.Access[a.UserID] == nil {
			data.Access[a.UserID] = make(map[int64]bool)
		}
		data.Access[a.UserID][a.PantryID] = true
	}

	ts, ok := h.templatesCache["admin.html"]
	if !ok {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	if err := ts.Execute(w, data); err != nil {
		h.logger.Error("render template", "template", "admin.html", "error", err)
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}

// AdminCreatePantry handles POST /settings/admin/pantries: creates a new
// pantry from the "+ Create pantry" form.
func (h *Handler) AdminCreatePantry(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		redirectWithError(w, r, "/settings/admin", "invalid form submission")
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		redirectWithError(w, r, "/settings/admin", "pantry name is required")
		return
	}

	if _, err := h.store.CreatePantry(r.Context(), name); err != nil {
		h.logger.Error("create pantry", "error", err)
		redirectWithError(w, r, "/settings/admin", "something went wrong, please try again")
		return
	}
	http.Redirect(w, r, "/settings/admin", http.StatusSeeOther)
}

// AdminUpdatePantry handles POST /settings/admin/pantries/{id}/update:
// renames a pantry in place from the pencil-icon edit form.
func (h *Handler) AdminUpdatePantry(w http.ResponseWriter, r *http.Request) {
	id, ok := parseInt64PathHTML(r, "id")
	if !ok {
		redirectWithError(w, r, "/settings/admin", "invalid pantry")
		return
	}

	if err := r.ParseForm(); err != nil {
		redirectWithError(w, r, "/settings/admin", "invalid form submission")
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		redirectWithError(w, r, "/settings/admin", "pantry name is required")
		return
	}

	if _, err := h.store.UpdatePantry(r.Context(), id, name); err != nil {
		redirectWithError(w, r, "/settings/admin", humanPantryContainerStoreError(err))
		return
	}
	http.Redirect(w, r, "/settings/admin", http.StatusSeeOther)
}

// humanPantryContainerStoreError maps a store error from a pantry
// (container, not pantry_ingredients — see humanPantryStoreError in
// pantry_page.go for that) operation to a user-facing message.
func humanPantryContainerStoreError(err error) string {
	if errors.Is(err, store.ErrNotFound) {
		return "pantry not found"
	}
	return "something went wrong, please try again"
}

// AdminSetPantryAccess handles POST /settings/admin/pantry-access: replaces
// every user_pantry row with the checked cells of the access matrix.
// Each checked cell is submitted as one "access" value of the form
// "<user_id>:<pantry_id>" (see templates/admin.html).
func (h *Handler) AdminSetPantryAccess(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		redirectWithError(w, r, "/settings/admin", "invalid form submission")
		return
	}

	// De-duplicate: nothing stops a malformed submission from repeating the
	// same cell, and SetUserPantryAccess has no unique constraint to lean on.
	seen := make(map[store.UserPantryAccess]bool)
	var access []store.UserPantryAccess
	for _, raw := range r.Form["access"] {
		pair, ok := parseAccessCell(raw)
		if !ok || seen[pair] {
			continue
		}
		seen[pair] = true
		access = append(access, pair)
	}

	if err := h.store.SetUserPantryAccess(r.Context(), access); err != nil {
		h.logger.Error("set user pantry access", "error", err)
		redirectWithError(w, r, "/settings/admin", "something went wrong, please try again")
		return
	}
	http.Redirect(w, r, "/settings/admin", http.StatusSeeOther)
}

// parseAccessCell parses one "<user_id>:<pantry_id>" checkbox value from the
// admin page's access matrix.
func parseAccessCell(raw string) (store.UserPantryAccess, bool) {
	userRaw, pantryRaw, found := strings.Cut(raw, ":")
	if !found {
		return store.UserPantryAccess{}, false
	}
	userID, err := strconv.ParseInt(userRaw, 10, 64)
	if err != nil || userID <= 0 {
		return store.UserPantryAccess{}, false
	}
	pantryID, err := strconv.ParseInt(pantryRaw, 10, 64)
	if err != nil || pantryID <= 0 {
		return store.UserPantryAccess{}, false
	}
	return store.UserPantryAccess{UserID: userID, PantryID: pantryID}, true
}
