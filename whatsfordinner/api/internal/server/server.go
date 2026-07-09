// Package server wires together the HTTP router, middleware and handlers.
package server

import (
	"html/template"
	"log/slog"
	"net/http"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/config"
	apiHandlers "github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/handlers/templates"
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

	parsedTemplates, err := template.ParseGlob("templates/*.html")
	if err != nil {
		parsedTemplates, err = template.ParseGlob("./../../templates/*.html")
	}

	templates := template.Must(parsedTemplates, err)

	templateCache := make(map[string]*template.Template)
	for _, t := range templates.Templates() {
		templateCache[t.Name()] = t
	}

	mux := http.NewServeMux()
	h := apiHandlers.New(s.store, s.logger, templateCache)

	// --- Public endpoints (no authentication) ---
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("GET /ready", h.Ready)

	// // --- Protected endpoints (X-Api-Key header required) ---
	// api := http.NewServeMux()

	// // Recipes.
	// api.HandleFunc("GET /api/recipes", h.ListRecipes)
	// api.HandleFunc("POST /api/recipes", h.CreateRecipe)
	// api.HandleFunc("GET /api/recipes/cookable", h.ListCookableRecipes)
	// api.HandleFunc("GET /api/recipes/{id}", h.GetRecipe)
	// api.HandleFunc("PUT /api/recipes/{id}", h.UpdateRecipe)
	// api.HandleFunc("DELETE /api/recipes/{id}", h.DeleteRecipe)

	// // Ingredients.
	// api.HandleFunc("GET /api/ingredients", h.ListIngredients)
	// api.HandleFunc("POST /api/ingredients", h.CreateIngredient)
	// api.HandleFunc("GET /api/ingredients/{id}", h.GetIngredient)
	// api.HandleFunc("PUT /api/ingredients/{id}", h.UpdateIngredient)
	// api.HandleFunc("DELETE /api/ingredients/{id}", h.DeleteIngredient)

	// // Units.
	// api.HandleFunc("GET /api/units", h.ListUnits)
	// api.HandleFunc("POST /api/units", h.CreateUnit)
	// api.HandleFunc("GET /api/units/{id}", h.GetUnit)
	// api.HandleFunc("PUT /api/units/{id}", h.UpdateUnit)
	// api.HandleFunc("DELETE /api/units/{id}", h.DeleteUnit)

	// // Tags.
	// api.HandleFunc("GET /api/tags", h.ListTags)
	// api.HandleFunc("POST /api/tags", h.CreateTag)
	// api.HandleFunc("GET /api/tags/{id}", h.GetTag)
	// api.HandleFunc("PUT /api/tags/{id}", h.UpdateTag)
	// api.HandleFunc("DELETE /api/tags/{id}", h.DeleteTag)

	// // Food locations.
	// api.HandleFunc("GET /api/locations", h.ListFoodLocations)
	// api.HandleFunc("POST /api/locations", h.CreateFoodLocation)
	// api.HandleFunc("GET /api/locations/{id}", h.GetFoodLocation)
	// api.HandleFunc("PUT /api/locations/{id}", h.UpdateFoodLocation)
	// api.HandleFunc("DELETE /api/locations/{id}", h.DeleteFoodLocation)

	// // Pantry stock.
	// api.HandleFunc("GET /api/pantry", h.ListPantry)
	// api.HandleFunc("POST /api/pantry", h.CreatePantryItem)
	// api.HandleFunc("GET /api/pantry/{id}", h.GetPantryItem)
	// api.HandleFunc("PUT /api/pantry/{id}", h.UpdatePantryItem)
	// api.HandleFunc("DELETE /api/pantry/{id}", h.DeletePantryItem)

	// // Cooking history.
	// api.HandleFunc("GET /api/past-cooked", h.ListPastCooked)
	// api.HandleFunc("POST /api/past-cooked", h.CreatePastCooked)
	// api.HandleFunc("GET /api/past-cooked/most-cooked", h.ListMostCooked)
	// api.HandleFunc("GET /api/past-cooked/{id}", h.GetPastCooked)
	// api.HandleFunc("PUT /api/past-cooked/{id}", h.UpdatePastCooked)
	// api.HandleFunc("DELETE /api/past-cooked/{id}", h.DeletePastCooked)

	// mux.Handle("/", s.requireAPIKey(api))

	// Home page: shared navbar + dashboard of quick-access tiles. Registered
	// at both "/" and "/home" while the rest of the site is being migrated
	// off JSON responses onto server-rendered templates.
	mux.HandleFunc("GET /{$}", h.Home)
	mux.HandleFunc("GET /home", h.Home)
	mux.HandleFunc("/recipes", h.ListRecipes)

	// Sign-in / user management. There is no password: /login lists every
	// user so one can be picked (POST .../select sets a cookie), and doubles
	// as basic CRUD for the users table (create/rename/delete).
	mux.HandleFunc("GET /login", h.Login)
	mux.HandleFunc("POST /users", h.CreateUser)
	mux.HandleFunc("POST /users/{id}/update", h.UpdateUser)
	mux.HandleFunc("POST /users/{id}/delete", h.DeleteUser)
	mux.HandleFunc("POST /users/{id}/select", h.SelectUser)

	return s.withMiddleware(mux)
}
