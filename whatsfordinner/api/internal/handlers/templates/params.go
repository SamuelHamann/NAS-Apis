package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// nameRequest is the JSON body shared by the simple name-only resources
// (ingredients, units, tags).
type nameRequest struct {
	Name string `json:"name"`
}

func (n nameRequest) validate() string {
	if strings.TrimSpace(n.Name) == "" {
		return "name is required"
	}
	return ""
}

func (n nameRequest) value() string {
	return strings.TrimSpace(n.Name)
}

// parseUUIDPath parses a UUID path parameter, writing a 400 response on failure.
func parseUUIDPath(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue(name))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid "+name)
		return uuid.Nil, false
	}
	return id, true
}

// parseInt64Path parses a positive integer path parameter, writing a 400
// response on failure.
func parseInt64Path(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid "+name)
		return 0, false
	}
	return id, true
}

// parseInt64PathHTML is the HTML-flow variant of parseInt64Path: it returns
// (id, false) instead of writing an error response, so the caller can
// choose how to respond (e.g. render a 404 page instead of a JSON error).
func parseInt64PathHTML(r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// pagination reads optional ?limit and ?offset query parameters.
func pagination(r *http.Request) (limit, offset int) {
	return atoiDefault(r.URL.Query().Get("limit"), 0),
		atoiDefault(r.URL.Query().Get("offset"), 0)
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return def
}
