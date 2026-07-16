package handlers

import (
	"net/http"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
)

// ProfilePageData is the view model for templates/profile.html.
type ProfilePageData struct {
	PageData

	// Collections is nil (not just empty) while signed out — the template
	// tells the two states apart via .SignedIn instead.
	Collections []models.Collection

	Error string
}

// ProfilePage renders GET /settings/profile: the signed-in user's
// collections, plus a "+ Create collection" form. Collections are where a
// user curates named lists of recipes (see collections.go and the "add to
// collection" widget on the recipes list/detail pages).
//
// There's no password-protected auth in this app (see session.go) — being
// signed out just means the page shows a "sign in first" prompt instead of
// erroring.
func (h *Handler) ProfilePage(w http.ResponseWriter, r *http.Request) {
	data := ProfilePageData{
		PageData: h.newPageData(r, "Profile", ""),
		Error:    r.URL.Query().Get("error"),
	}

	if data.SignedIn {
		userID, _ := sessionUserID(r)
		collections, err := h.store.ListCollectionsForUser(r.Context(), userID)
		if err != nil {
			h.logger.Error("list collections", "error", err)
			http.Error(w, "failed to load collections", http.StatusInternalServerError)
			return
		}
		data.Collections = collections
	}

	ts, ok := h.templatesCache["profile.html"]
	if !ok {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	if err := ts.Execute(w, data); err != nil {
		h.logger.Error("render template", "template", "profile.html", "error", err)
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}
