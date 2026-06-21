/**
 * Shared helper for "list view" pages.
 *
 * Each resource page (recipes, ingredients, …) is a thin wrapper that picks
 * which columns to render and which API call to make. Forms for create/update
 * are intentionally not wired up yet — this is the initial scaffolding.
 */

import { ApiError } from "../api/client";
import { h, mount } from "../dom";

export interface Column<T> {
  header: string;
  value: (item: T) => string;
}

export interface ResourceListOptions<T> {
  title: string;
  columns: Column<T>[];
  fetcher: () => Promise<T[]>;
}

export async function renderResourceList<T>(
  opts: ResourceListOptions<T>,
): Promise<void> {
  mount(
    "#app",
    h("h2", null, opts.title),
    h("p", { class: "muted" }, "Loading…"),
  );

  let items: T[];
  try {
    items = await opts.fetcher();
  } catch (err) {
    const message =
      err instanceof ApiError
        ? `${err.status} — ${err.message}`
        : (err as Error).message;
    mount(
      "#app",
      h("h2", null, opts.title),
      h("p", { class: "error" }, `Failed to load: ${message}`),
    );
    return;
  }

  const headerRow = h(
    "tr",
    null,
    ...opts.columns.map((col) => h("th", null, col.header)),
  );

  const bodyRows =
    items.length === 0
      ? [
          h(
            "tr",
            null,
            h(
              "td",
              { class: "muted", colspan: String(opts.columns.length) },
              "No items yet.",
            ),
          ),
        ]
      : items.map((item) =>
          h(
            "tr",
            null,
            ...opts.columns.map((col) => h("td", null, col.value(item))),
          ),
        );

  mount(
    "#app",
    h("h2", null, opts.title),
    h(
      "p",
      { class: "muted" },
      `${items.length} item${items.length === 1 ? "" : "s"}`,
    ),
    h(
      "table",
      { class: "resource-table" },
      h("thead", null, headerRow),
      h("tbody", null, ...bodyRows),
    ),
  );
}
