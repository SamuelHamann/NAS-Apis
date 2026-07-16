package handlers

import "net/http"

// PageData holds the fields every full-page template needs in order to
// render the shared chrome (currently just the navbar). Page-specific view
// models should embed PageData so `{{template "navbar" .}}` always has what
// it expects, e.g.:
//
//	type SomePageData struct {
//		PageData
//		Foo string
//	}
type PageData struct {
	// Title is used in the <title> tag.
	Title string
	// SignedIn reports whether a user is currently selected (see
	// session.go). When false, the navbar shows a "Sign in" button instead
	// of a name + settings gear.
	SignedIn bool
	// CurrentUser is the display name shown in the navbar. Empty when
	// SignedIn is false.
	CurrentUser string
	// ActiveNav marks which navbar link (see templates/navbar.html) should
	// be highlighted as "current page": "pantry", "recipes",
	// "ingredients" or "scan-receipt". Empty means none of them are
	// highlighted (e.g. /login, which isn't in the nav).
	ActiveNav string
}

// newPageData builds the PageData common to every page: who (if anyone) is
// signed in, resolved from the session cookie, plus which navbar link (if
// any) to highlight as the current page — see PageData.ActiveNav.
func (h *Handler) newPageData(r *http.Request, title, activeNav string) PageData {
	data := PageData{Title: title, ActiveNav: activeNav}

	id, ok := sessionUserID(r)
	if !ok {
		return data
	}

	user, err := h.store.GetUser(r.Context(), id)
	if err != nil {
		// Unknown/deleted user id (e.g. a stale cookie) — treat as signed
		// out rather than failing the whole page render.
		return data
	}

	data.SignedIn = true
	data.CurrentUser = user.Username
	return data
}
