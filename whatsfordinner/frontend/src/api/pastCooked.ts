/** Cooking history — one row per recipe with a counter and last-cooked time. */

import { api } from "./client";
import type {
  PaginationQuery,
  PastCookedCreateInput,
  PastCookedRecipe,
  PastCookedUpdateInput,
  UUID,
} from "../types/models";

export function listPastCooked(
  query: PaginationQuery = {},
  signal?: AbortSignal,
): Promise<PastCookedRecipe[]> {
  return api.get<PastCookedRecipe[]>("/past-cooked", { query, signal });
}

export function listMostCooked(
  query: PaginationQuery = {},
  signal?: AbortSignal,
): Promise<PastCookedRecipe[]> {
  return api.get<PastCookedRecipe[]>("/past-cooked/most-cooked", {
    query,
    signal,
  });
}

export function getPastCooked(
  id: UUID,
  signal?: AbortSignal,
): Promise<PastCookedRecipe> {
  return api.get<PastCookedRecipe>(`/past-cooked/${encodeURIComponent(id)}`, {
    signal,
  });
}

export function createPastCooked(
  input: PastCookedCreateInput,
): Promise<PastCookedRecipe> {
  return api.post<PastCookedRecipe>("/past-cooked", input);
}

export function updatePastCooked(
  id: UUID,
  input: PastCookedUpdateInput,
): Promise<PastCookedRecipe> {
  return api.put<PastCookedRecipe>(
    `/past-cooked/${encodeURIComponent(id)}`,
    input,
  );
}

export function deletePastCooked(id: UUID): Promise<void> {
  return api.del<void>(`/past-cooked/${encodeURIComponent(id)}`);
}
