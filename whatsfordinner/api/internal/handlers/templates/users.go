package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
)

// LoginData is the view model for templates/login.html.
type LoginData struct {
	PageData
	Users []models.User
	Error string
}

// Login renders the "who's cooking?" page: every user, each editable/
// deletable in place, plus a create-user form. It is also where the
// navbar's "Sign in" button and "Change user" menu entry both point.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	users, err := h.store.ListUsers(r.Context())
	if err != nil {
		h.logger.Error("list users", "error", err)
		http.Error(w, "failed to load users", http.StatusInternalServerError)
		return
	}

	ts, ok := h.templatesCache["login.html"]
	if !ok {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}

	data := LoginData{
		PageData: h.newPageData(r, "Sign in"),
		Users:    users,
		Error:    r.URL.Query().Get("error"),
	}

	if err := ts.Execute(w, data); err != nil {
		h.logger.Error("render template", "template", "login.html", "error", err)
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}

// CreateUser handles POST /users: creates a new user from the "+ Create new
// user" form and returns to the list.
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		redirectWithError(w, r, "/login", "invalid form submission")
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	if username == "" {
		redirectWithError(w, r, "/login", "username is required")
		return
	}

	if _, err := h.store.CreateUser(r.Context(), username); err != nil {
		h.respondUserStoreError(w, r, err)
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// UpdateUser handles POST /users/{id}/update: renames a user in place from
// the pencil-icon edit form.
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUserIDPath(r)
	if !ok {
		redirectWithError(w, r, "/login", "invalid user")
		return
	}

	if err := r.ParseForm(); err != nil {
		redirectWithError(w, r, "/login", "invalid form submission")
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	if username == "" {
		redirectWithError(w, r, "/login", "username is required")
		return
	}

	if _, err := h.store.UpdateUser(r.Context(), id, username); err != nil {
		h.respondUserStoreError(w, r, err)
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// DeleteUser handles POST /users/{id}/delete from the trash-icon button.
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUserIDPath(r)
	if !ok {
		redirectWithError(w, r, "/login", "invalid user")
		return
	}

	if err := h.store.DeleteUser(r.Context(), id); err != nil {
		h.respondUserStoreError(w, r, err)
		return
	}

	// If the signed-in user just deleted themselves, sign this browser out
	// too so it doesn't keep pointing at a user id that no longer exists.
	if currentID, ok := sessionUserID(r); ok && currentID == id {
		clearSessionUser(w)
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// SelectUser handles POST /users/{id}/select: "signs in" as this user by
// storing their id in a cookie. See session.go for why there is no password.
func (h *Handler) SelectUser(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUserIDPath(r)
	if !ok {
		redirectWithError(w, r, "/login", "invalid user")
		return
	}

	if _, err := h.store.GetUser(r.Context(), id); err != nil {
		h.respondUserStoreError(w, r, err)
		return
	}

	setSessionUser(w, id)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// parseUserIDPath parses the "id" path value for the /users/{id}/... routes.
func parseUserIDPath(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// respondUserStoreError maps a store error to a redirect back to /login with
// a human-readable message. Unlike the JSON handlers, these are HTML forms,
// so we never want to write a raw {"error": "..."} body to the browser.
func (h *Handler) respondUserStoreError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		redirectWithError(w, r, "/login", "user not found")
	case errors.Is(err, store.ErrConflict):
		redirectWithError(w, r, "/login", "that username is already taken")
	default:
		h.logger.Error("store error", "error", err)
		redirectWithError(w, r, "/login", "something went wrong, please try again")
	}
}
