/** Tags — recipe categories (vegan, quick, gluten-free, …). */

import { api } from "./client";
import type { NamedInput, PaginationQuery, Tag } from "../types/models";

export function listTags(
  query: PaginationQuery = {},
  signal?: AbortSignal,
): Promise<Tag[]> {
  return api.get<Tag[]>("/tags", { query, signal });
}

export function getTag(id: number, signal?: AbortSignal): Promise<Tag> {
  return api.get<Tag>(`/tags/${id}`, { signal });
}

export function createTag(input: NamedInput): Promise<Tag> {
  return api.post<Tag>("/tags", input);
}

export function updateTag(id: number, input: NamedInput): Promise<Tag> {
  return api.put<Tag>(`/tags/${id}`, input);
}

export function deleteTag(id: number): Promise<void> {
  return api.del<void>(`/tags/${id}`);
}
