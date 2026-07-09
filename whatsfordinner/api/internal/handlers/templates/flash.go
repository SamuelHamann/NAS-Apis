package handlers

import (
	"net/http"
	"net/url"
)

// redirectWithError redirects to location with ?error=<message> attached so
// the destination page can render a small inline error banner. This keeps
// form-handling mutation handlers simple: validate, redirect.
func redirectWithError(w http.ResponseWriter, r *http.Request, location, message string) {
	http.Redirect(w, r, location+"?error="+url.QueryEscape(message), http.StatusSeeOther)
}
