/**
 * TypeScript representations of the whatsfordinner API resources.
 *
 * These mirror the Go structs in `whatsfordinner/api/internal/models`. Fields
 * that are pointers on the Go side (nullable columns) become `T | null` here.
 * Server-managed fields (`id`, `created_at`, `updated_at`) are read-only.
 */

/** UUIDs come over the wire as strings. */
export type UUID = string;

/** ISO-8601 timestamp string. */
export type ISODateTime = string;

// ---------------------------------------------------------------------------
// Recipes
// ---------------------------------------------------------------------------

export interface Recipe {
  id: UUID;
  name: string;
  description: string | null;
  instructions: string | null;
  source_url: string | null;
  servings: number | null;
  prep_time_minutes: number | null;
  cook_time_minutes: number | null;
  created_at: ISODateTime;
  updated_at: ISODateTime;
}

export interface RecipeInput {
  name: string;
  description?: string | null;
  instructions?: string | null;
  source_url?: string | null;
  servings?: number | null;
  prep_time_minutes?: number | null;
  cook_time_minutes?: number | null;
}

export interface CookableRecipesQuery {
  tags?: number[];
  max_prep?: number;
  max_cook?: number;
  max_total?: number;
  limit?: number;
  offset?: number;
}

// ---------------------------------------------------------------------------
// Ingredients / Units / Tags / Locations (all "named entity" resources)
// ---------------------------------------------------------------------------

export interface Ingredient {
  id: number;
  name: string;
  created_at: ISODateTime;
}

export interface Unit {
  id: number;
  name: string;
  created_at: ISODateTime;
}

export interface Tag {
  id: number;
  name: string;
}

export interface FoodLocation {
  id: number;
  name: string;
  created_at: ISODateTime;
}

/** Shared request body for ingredients/units/tags/locations. */
export interface NamedInput {
  name: string;
}

// ---------------------------------------------------------------------------
// Pantry
// ---------------------------------------------------------------------------

export interface PantryIngredient {
  id: UUID;
  ingredient_id: number;
  quantity: number;
  unit_id: number;
  note: string | null;
  is_quantified: boolean;
  updated_at: ISODateTime;
  location_id: number | null;
}

export interface PantryInput {
  ingredient_id: number;
  quantity: number;
  unit_id: number;
  note?: string | null;
  is_quantified?: boolean;
  location_id?: number | null;
}

// ---------------------------------------------------------------------------
// Cooking history
// ---------------------------------------------------------------------------

export interface PastCookedRecipe {
  id: UUID;
  recipe_id: UUID;
  times_cooked: number;
  last_cooked_at: ISODateTime;
}

export interface PastCookedCreateInput {
  recipe_id: UUID;
  times_cooked?: number;
  last_cooked_at?: ISODateTime;
}

export interface PastCookedUpdateInput {
  times_cooked: number;
  last_cooked_at?: ISODateTime;
}

// ---------------------------------------------------------------------------
// Shared query parameters
// ---------------------------------------------------------------------------

export interface PaginationQuery {
  limit?: number;
  offset?: number;
}

export interface ApiErrorPayload {
  error: string;
}
