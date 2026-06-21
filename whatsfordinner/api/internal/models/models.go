// Package models contains the Go representations of the whatsfordinner database
// tables. Struct tags drive both JSON (de)serialization (`json`) and row
// scanning via pgx (`db`).
package models

import (
	"time"

	"github.com/google/uuid"
)

// Recipe maps to the recipes table.
type Recipe struct {
	ID              uuid.UUID `json:"id" db:"id"`
	Name            string    `json:"name" db:"name"`
	Description     *string   `json:"description" db:"description"`
	Instructions    *string   `json:"instructions" db:"instructions"`
	SourceURL       *string   `json:"source_url" db:"source_url"`
	Servings        *int32    `json:"servings" db:"servings"`
	PrepTimeMinutes *int32    `json:"prep_time_minutes" db:"prep_time_minutes"`
	CookTimeMinutes *int32    `json:"cook_time_minutes" db:"cook_time_minutes"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// Ingredient maps to the ingredients table (canonical, reusable ingredients).
type Ingredient struct {
	ID        int64     `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Unit maps to the units table (canonical measurement units).
type Unit struct {
	ID        int64     `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Tag maps to the tags table (vegan, quick, gluten-free…).
type Tag struct {
	ID   int64  `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

// PantryIngredient maps to the pantry_ingredients table (current home stock).
type PantryIngredient struct {
	ID           uuid.UUID `json:"id" db:"id"`
	IngredientID int64     `json:"ingredient_id" db:"ingredient_id"`
	Quantity     float64   `json:"quantity" db:"quantity"`
	UnitID       int64     `json:"unit_id" db:"unit_id"`
	Note         *string   `json:"note" db:"note"`
	IsQuantified bool      `json:"is_quantified" db:"is_quantified"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
	LocationID   *int64    `json:"location_id" db:"location_id"`
}

// FoodLocation maps to the food_locations table (fridge, freezer, pantry, …).
type FoodLocation struct {
	ID        int64     `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// RecipeIngredient maps to the recipe_ingredients junction table.
//
// Full CRUD for this relationship is intentionally deferred — it is managed as
// a sub-resource of a recipe (see the "more complex endpoints" TODO).
type RecipeIngredient struct {
	RecipeID     uuid.UUID `json:"recipe_id" db:"recipe_id"`
	IngredientID int64     `json:"ingredient_id" db:"ingredient_id"`
	Quantity     *float64  `json:"quantity" db:"quantity"`
	UnitID       *int64    `json:"unit_id" db:"unit_id"`
	Note         *string   `json:"note" db:"note"`
}

// RecipeTag maps to the recipe_tags junction table.
//
// Like RecipeIngredient, this relationship is managed as a sub-resource of a
// recipe and its endpoints are deferred.
type RecipeTag struct {
	RecipeID uuid.UUID `json:"recipe_id" db:"recipe_id"`
	TagID    int64     `json:"tag_id" db:"tag_id"`
}

// PastCookedRecipe maps to the past_cooked_recipes table.
// There is at most one row per recipe (recipe_id is UNIQUE).
type PastCookedRecipe struct {
	ID           uuid.UUID `json:"id" db:"id"`
	RecipeID     uuid.UUID `json:"recipe_id" db:"recipe_id"`
	TimesCooked  int32     `json:"times_cooked" db:"times_cooked"`
	LastCookedAt time.Time `json:"last_cooked_at" db:"last_cooked_at"`
}

// APIKey maps to the api_keys table. The UUID id IS the key — clients pass it
// verbatim in the X-Api-Key request header.
type APIKey struct {
	ID        uuid.UUID `json:"id" db:"id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
