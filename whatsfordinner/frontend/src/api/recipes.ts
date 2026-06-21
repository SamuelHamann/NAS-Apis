/** Recipes — CRUD plus the `/recipes/cookable` query endpoint. */

import { api } from "./client";
import type {
  CookableRecipesQuery,
  PaginationQuery,
  Recipe,
  RecipeInput,
  UUID,
} from "../types/models";

export function listRecipes(
  query: PaginationQuery = {},
  signal?: AbortSignal,
): Promise<Recipe[]> {
  return api.get<Recipe[]>("/recipes", { query, signal });
}

export function listCookableRecipes(
  query: CookableRecipesQuery = {},
  signal?: AbortSignal,
): Promise<Recipe[]> {
  return api.get<Recipe[]>("/recipes/cookable", { query, signal });
}

export function getRecipe(id: UUID, signal?: AbortSignal): Promise<Recipe> {
  return api.get<Recipe>(`/recipes/${encodeURIComponent(id)}`, { signal });
}

export function createRecipe(input: RecipeInput): Promise<Recipe> {
  return api.post<Recipe>("/recipes", input);
}

export function updateRecipe(id: UUID, input: RecipeInput): Promise<Recipe> {
  return api.put<Recipe>(`/recipes/${encodeURIComponent(id)}`, input);
}

export function deleteRecipe(id: UUID): Promise<void> {
  return api.del<void>(`/recipes/${encodeURIComponent(id)}`);
}
