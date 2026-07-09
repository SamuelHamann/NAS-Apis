// Package handlers contains the HTTP handlers for the whatsfordinner API.
package handlers

import (
	"html/template"
	"log/slog"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
)

// Handler bundles the dependencies shared by every HTTP handler.
type Handler struct {
	store          *store.Store
	logger         *slog.Logger
	templatesCache map[string]*template.Template
}

// New creates a Handler.
func New(store *store.Store, logger *slog.Logger, cache map[string]*template.Template) *Handler {
	return &Handler{store: store, logger: logger, templatesCache: cache}
}
