/** Units — canonical measurement units (g, ml, cup, …). */

import { api } from "./client";
import type { NamedInput, PaginationQuery, Unit } from "../types/models";

export function listUnits(
  query: PaginationQuery = {},
  signal?: AbortSignal,
): Promise<Unit[]> {
  return api.get<Unit[]>("/units", { query, signal });
}

export function getUnit(id: number, signal?: AbortSignal): Promise<Unit> {
  return api.get<Unit>(`/units/${id}`, { signal });
}

export function createUnit(input: NamedInput): Promise<Unit> {
  return api.post<Unit>("/units", input);
}

export function updateUnit(id: number, input: NamedInput): Promise<Unit> {
  return api.put<Unit>(`/units/${id}`, input);
}

export function deleteUnit(id: number): Promise<void> {
  return api.del<void>(`/units/${id}`);
}
