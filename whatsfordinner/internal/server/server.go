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
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	h := handlers.New(s.store, s.logger)

	// Operational endpoints.
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("GET /ready", h.Ready)

	// Recipes.
	mux.HandleFunc("GET /recipes", h.ListRecipes)
	mux.HandleFunc("POST /recipes", h.CreateRecipe)
	mux.HandleFunc("GET /recipes/cookable", h.ListCookableRecipes)
	mux.HandleFunc("GET /recipes/{id}", h.GetRecipe)
	mux.HandleFunc("PUT /recipes/{id}", h.UpdateRecipe)
	mux.HandleFunc("DELETE /recipes/{id}", h.DeleteRecipe)

	// Ingredients.
	mux.HandleFunc("GET /ingredients", h.ListIngredients)
	mux.HandleFunc("POST /ingredients", h.CreateIngredient)
	mux.HandleFunc("GET /ingredients/{id}", h.GetIngredient)
	mux.HandleFunc("PUT /ingredients/{id}", h.UpdateIngredient)
	mux.HandleFunc("DELETE /ingredients/{id}", h.DeleteIngredient)

	// Units.
	mux.HandleFunc("GET /units", h.ListUnits)
	mux.HandleFunc("POST /units", h.CreateUnit)
	mux.HandleFunc("GET /units/{id}", h.GetUnit)
	mux.HandleFunc("PUT /units/{id}", h.UpdateUnit)
	mux.HandleFunc("DELETE /units/{id}", h.DeleteUnit)

	// Tags.
	mux.HandleFunc("GET /tags", h.ListTags)
	mux.HandleFunc("POST /tags", h.CreateTag)
	mux.HandleFunc("GET /tags/{id}", h.GetTag)
	mux.HandleFunc("PUT /tags/{id}", h.UpdateTag)
	mux.HandleFunc("DELETE /tags/{id}", h.DeleteTag)

	// Food locations.
	mux.HandleFunc("GET /locations", h.ListFoodLocations)
	mux.HandleFunc("POST /locations", h.CreateFoodLocation)
	mux.HandleFunc("GET /locations/{id}", h.GetFoodLocation)
	mux.HandleFunc("PUT /locations/{id}", h.UpdateFoodLocation)
	mux.HandleFunc("DELETE /locations/{id}", h.DeleteFoodLocation)

	// Pantry stock.
	mux.HandleFunc("GET /pantry", h.ListPantry)
	mux.HandleFunc("POST /pantry", h.CreatePantryItem)
	mux.HandleFunc("GET /pantry/{id}", h.GetPantryItem)
	mux.HandleFunc("PUT /pantry/{id}", h.UpdatePantryItem)
	mux.HandleFunc("DELETE /pantry/{id}", h.DeletePantryItem)

	// Cooking history.
	mux.HandleFunc("GET /past-cooked", h.ListPastCooked)
	mux.HandleFunc("POST /past-cooked", h.CreatePastCooked)
	mux.HandleFunc("GET /past-cooked/most-cooked", h.ListMostCooked)
	mux.HandleFunc("GET /past-cooked/{id}", h.GetPastCooked)
	mux.HandleFunc("PUT /past-cooked/{id}", h.UpdatePastCooked)
	mux.HandleFunc("DELETE /past-cooked/{id}", h.DeletePastCooked)

	// TODO (more complex, deferred): manage the junction tables as recipe
	// sub-resources, e.g.
	//   GET/PUT/DELETE /recipes/{id}/ingredients
	//   GET/PUT/DELETE /recipes/{id}/tags

	return s.withMiddleware(mux)
}
