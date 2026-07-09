package handlers

import "net/http"

// HomeData is the view model rendered by templates/home.html.
type HomeData struct {
	PageData
}

// Home renders the application's home page: the shared navbar plus a
// dashboard of quick-access tiles (recipes, pantry, ...).
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	ts, ok := h.templatesCache["home.html"]
	if !ok {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}

	data := HomeData{PageData: h.newPageData(r, "Home")}

	if err := ts.Execute(w, data); err != nil {
		h.logger.Error("render template", "template", "home.html", "error", err)
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}
