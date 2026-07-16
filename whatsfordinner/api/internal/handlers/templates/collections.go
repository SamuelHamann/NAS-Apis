package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
)

// CollectionsCreate handles POST /collections: creates a collection owned
// by the signed-in user from the profile page's "+ Create collection" form.
func (h *Handler) CollectionsCreate(w http.ResponseWriter, r *http.Request) {
	userID, ok := sessionUserID(r)
	if !ok {
		redirectWithError(w, r, "/settings/profile", "sign in first")
		return
	}

	if err := r.ParseForm(); err != nil {
		redirectWithError(w, r, "/settings/profile", "invalid form submission")
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		redirectWithError(w, r, "/settings/profile", "collection name is required")
		return
	}

	if _, err := h.store.CreateCollection(r.Context(), userID, name); err != nil {
		redirectWithError(w, r, "/settings/profile", humanCollectionStoreError(err))
		return
	}
	http.Redirect(w, r, "/settings/profile", http.StatusSeeOther)
}

// CollectionsToggleRecipe handles POST /collections/{id}/recipes/{recipeId}/toggle:
// adds recipeId to the collection if it isn't already there, or removes it
// if it is (see templates/recipes.html / recipe_detail.html's "add to
// collection" widget — every checkbox posts to this same endpoint
// regardless of direction, since the toggle is idempotent with respect to
// whatever the current DB state is).
//
// Only the collection's owner may do this — a collection's membership is
// only ever meaningful to the user who curates it.
//
// "collection-toggle-script" (scripts.html) always calls this via fetch(),
// identified by the X-Requested-With header — that path responds with a
// small JSON body instead of redirecting, since there's no page to reload.
// A plain form submission (no JS, or some other caller) still gets the
// original redirect-and-redisplay behavior.
func (h *Handler) CollectionsToggleRecipe(w http.ResponseWriter, r *http.Request) {
	isFetch := r.Header.Get("X-Requested-With") == "fetch"
	redirectTo := safeRedirectTarget(r.FormValue("redirect_to"), "/recipes")

	fail := func(status int, message string) {
		if isFetch {
			writeError(w, status, message)
			return
		}
		redirectWithError(w, r, redirectTo, message)
	}

	userID, ok := sessionUserID(r)
	if !ok {
		fail(http.StatusUnauthorized, "sign in first")
		return
	}

	collectionID, ok := parseInt64PathHTML(r, "id")
	if !ok {
		fail(http.StatusBadRequest, "invalid collection")
		return
	}
	recipeID, ok := parseInt64PathHTML(r, "recipeId")
	if !ok {
		fail(http.StatusBadRequest, "invalid recipe")
		return
	}

	collection, err := h.store.GetCollection(r.Context(), collectionID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrNotFound) {
			status = http.StatusNotFound
		}
		fail(status, humanCollectionStoreError(err))
		return
	}
	if collection.UserID != userID {
		fail(http.StatusForbidden, "that collection isn't yours")
		return
	}

	added, err := h.store.ToggleCollectionRecipe(r.Context(), collectionID, recipeID)
	if err != nil {
		fail(http.StatusInternalServerError, humanCollectionStoreError(err))
		return
	}

	if isFetch {
		writeJSON(w, http.StatusOK, map[string]bool{"added": added})
		return
	}
	http.Redirect(w, r, redirectTo, http.StatusSeeOther)
}

// safeRedirectTarget returns raw if it looks like a same-site relative path
// (starts with a single "/", not "//" which browsers treat as
// protocol-relative to another host), otherwise fallback. Used so
// CollectionsToggleRecipe can send the browser back to whatever page it
// came from (the recipes list, preserving its filters, or a recipe's detail
// page) without trusting an arbitrary redirect target from the request.
func safeRedirectTarget(raw, fallback string) string {
	if strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//") {
		return raw
	}
	return fallback
}

// humanCollectionStoreError maps a store error from a collection operation
// to a user-facing message.
func humanCollectionStoreError(err error) string {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return "collection not found"
	case errors.Is(err, store.ErrConflict):
		return "you already have a collection with that name"
	case errors.Is(err, store.ErrReference):
		return "that recipe no longer exists"
	default:
		return "something went wrong, please try again"
	}
}
