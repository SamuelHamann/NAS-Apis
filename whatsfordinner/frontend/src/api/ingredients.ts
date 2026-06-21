/** Ingredients — canonical, reusable ingredient names. */

import { api } from "./client";
import type {
  Ingredient,
  NamedInput,
  PaginationQuery,
} from "../types/models";

export function listIngredients(
  query: PaginationQuery = {},
  signal?: AbortSignal,
): Promise<Ingredient[]> {
  return api.get<Ingredient[]>("/ingredients", { query, signal });
}

export function getIngredient(
  id: number,
  signal?: AbortSignal,
): Promise<Ingredient> {
  return api.get<Ingredient>(`/ingredients/${id}`, { signal });
}

export function createIngredient(input: NamedInput): Promise<Ingredient> {
  return api.post<Ingredient>("/ingredients", input);
}

export function updateIngredient(
  id: number,
  input: NamedInput,
): Promise<Ingredient> {
  return api.put<Ingredient>(`/ingredients/${id}`, input);
}

export function deleteIngredient(id: number): Promise<void> {
  return api.del<void>(`/ingredients/${id}`);
}
