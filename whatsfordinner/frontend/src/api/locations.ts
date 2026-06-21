/** Food locations — where ingredients are stored (fridge, freezer, pantry). */

import { api } from "./client";
import type {
  FoodLocation,
  NamedInput,
  PaginationQuery,
} from "../types/models";

export function listLocations(
  query: PaginationQuery = {},
  signal?: AbortSignal,
): Promise<FoodLocation[]> {
  return api.get<FoodLocation[]>("/locations", { query, signal });
}

export function getLocation(
  id: number,
  signal?: AbortSignal,
): Promise<FoodLocation> {
  return api.get<FoodLocation>(`/locations/${id}`, { signal });
}

export function createLocation(input: NamedInput): Promise<FoodLocation> {
  return api.post<FoodLocation>("/locations", input);
}

export function updateLocation(
  id: number,
  input: NamedInput,
): Promise<FoodLocation> {
  return api.put<FoodLocation>(`/locations/${id}`, input);
}

export function deleteLocation(id: number): Promise<void> {
  return api.del<void>(`/locations/${id}`);
}
