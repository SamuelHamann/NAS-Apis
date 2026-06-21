/** Pantry stock — current home inventory. */

import { api } from "./client";
import type {
  PaginationQuery,
  PantryInput,
  PantryIngredient,
  UUID,
} from "../types/models";

export function listPantry(
  query: PaginationQuery = {},
  signal?: AbortSignal,
): Promise<PantryIngredient[]> {
  return api.get<PantryIngredient[]>("/pantry", { query, signal });
}

export function getPantryItem(
  id: UUID,
  signal?: AbortSignal,
): Promise<PantryIngredient> {
  return api.get<PantryIngredient>(`/pantry/${encodeURIComponent(id)}`, {
    signal,
  });
}

export function createPantryItem(
  input: PantryInput,
): Promise<PantryIngredient> {
  return api.post<PantryIngredient>("/pantry", input);
}

export function updatePantryItem(
  id: UUID,
  input: PantryInput,
): Promise<PantryIngredient> {
  return api.put<PantryIngredient>(
    `/pantry/${encodeURIComponent(id)}`,
    input,
  );
}

export function deletePantryItem(id: UUID): Promise<void> {
  return api.del<void>(`/pantry/${encodeURIComponent(id)}`);
}
