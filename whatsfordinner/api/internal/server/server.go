// Package server wires together the HTTP router, middleware and handlers.
package server

import (
	"log/slog"
	"net/http"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/config"
	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/handlers"
	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
)

// Server holds the dependencies shared across HTTP handlers.
type Server struct {
	cfg    config.Config
	logger *slog.Logger
	store  *store.Store
}

// New creates a new Server with its dependencies.
func New(cfg config.Config, logger *slog.Logger, store *store.Store) *Server {
	return &Server{
		cfg:    cfg,
		logger: logger,
		store:  store,
	}
}

// Routes registers every application route and returns the root HTTP handler,
// wrapped with the common middleware stack.
//
// Routes use the Go 1.22+ pattern syntax ("METHOD /path").
//
// Layout:
//   - /health and /ready are public (no authentication).
//   - Everything else is mounted on an inner mux wrapped by requireAPIKey;
//     callers must supply a valid UUID in the X-Api-Key header.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	h := handlers.New(s.store, s.logger)

	// --- Public endpoints (no authentication) ---
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("GET /ready", h.Ready)

	// --- Protected endpoints (X-Api-Key header required) ---
	api := http.NewServeMux()

	// Recipes.
	api.HandleFunc("GET /recipes", h.ListRecipes)
	api.HandleFunc("POST /recipes", h.CreateRecipe)
	api.HandleFunc("GET /recipes/cookable", h.ListCookableRecipes)
	api.HandleFunc("GET /recipes/{id}", h.GetRecipe)
	api.HandleFunc("PUT /recipes/{id}", h.UpdateRecipe)
	api.HandleFunc("DELETE /recipes/{id}", h.DeleteRecipe)

	// Ingredients.
	api.HandleFunc("GET /ingredients", h.ListIngredients)
	api.HandleFunc("POST /ingredients", h.CreateIngredient)
	api.HandleFunc("GET /ingredients/{id}", h.GetIngredient)
	api.HandleFunc("PUT /ingredients/{id}", h.UpdateIngredient)
	api.HandleFunc("DELETE /ingredients/{id}", h.DeleteIngredient)

	// Units.
	api.HandleFunc("GET /units", h.ListUnits)
	api.HandleFunc("POST /units", h.CreateUnit)
	api.HandleFunc("GET /units/{id}", h.GetUnit)
	api.HandleFunc("PUT /units/{id}", h.UpdateUnit)
	api.HandleFunc("DELETE /units/{id}", h.DeleteUnit)

	// Tags.
	api.HandleFunc("GET /tags", h.ListTags)
	api.HandleFunc("POST /tags", h.CreateTag)
	api.HandleFunc("GET /tags/{id}", h.GetTag)
	api.HandleFunc("PUT /tags/{id}", h.UpdateTag)
	api.HandleFunc("DELETE /tags/{id}", h.DeleteTag)

	// Food locations.
	api.HandleFunc("GET /locations", h.ListFoodLocations)
	api.HandleFunc("POST /locations", h.CreateFoodLocation)
	api.HandleFunc("GET /locations/{id}", h.GetFoodLocation)
	api.HandleFunc("PUT /locations/{id}", h.UpdateFoodLocation)
	api.HandleFunc("DELETE /locations/{id}", h.DeleteFoodLocation)

	// Pantry stock.
	api.HandleFunc("GET /pantry", h.ListPantry)
	api.HandleFunc("POST /pantry", h.CreatePantryItem)
	api.HandleFunc("GET /pantry/{id}", h.GetPantryItem)
	api.HandleFunc("PUT /pantry/{id}", h.UpdatePantryItem)
	api.HandleFunc("DELETE /pantry/{id}", h.DeletePantryItem)

	// Cooking history.
	api.HandleFunc("GET /past-cooked", h.ListPastCooked)
	api.HandleFunc("POST /past-cooked", h.CreatePastCooked)
	api.HandleFunc("GET /past-cooked/most-cooked", h.ListMostCooked)
	api.HandleFunc("GET /past-cooked/{id}", h.GetPastCooked)
	api.HandleFunc("PUT /past-cooked/{id}", h.UpdatePastCooked)
	api.HandleFunc("DELETE /past-cooked/{id}", h.DeletePastCooked)

	// TODO (more complex, deferred): manage the junction tables as recipe
	// sub-resources, e.g.
	//   GET/PUT/DELETE /recipes/{id}/ingredients
	//   GET/PUT/DELETE /recipes/{id}/tags

	// Mount the protected mux behind the auth middleware. The outer mux's
	// more-specific /health and /ready patterns take precedence over "/".
	mux.Handle("/", s.requireAPIKey(api))

	return s.withMiddleware(mux)
}
