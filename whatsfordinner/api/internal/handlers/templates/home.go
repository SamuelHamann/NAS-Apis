package handlers

import "net/http"

// Home redirects "/" and "/home" to /pantry, the app's actual landing page.
// There used to be a dashboard of quick-access tiles here; the pantry page
// is what people land on every time in practice, so it's the real home page
// now and this route is just kept around so old links/bookmarks still work.
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/pantry", http.StatusSeeOther)
}
