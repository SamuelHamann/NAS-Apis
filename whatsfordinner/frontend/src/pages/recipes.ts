import { listRecipes } from "../api/recipes";
import { renderResourceList } from "./resourceList";

export async function renderRecipes(): Promise<void> {
  await renderResourceList({
    title: "Recipes",
    columns: [
      { header: "Name", value: (r) => r.name },
      {
        header: "Servings",
        value: (r) => (r.servings != null ? String(r.servings) : "—"),
      },
      {
        header: "Prep (min)",
        value: (r) =>
          r.prep_time_minutes != null ? String(r.prep_time_minutes) : "—",
      },
      {
        header: "Cook (min)",
        value: (r) =>
          r.cook_time_minutes != null ? String(r.cook_time_minutes) : "—",
      },
    ],
    fetcher: () => listRecipes(),
  });
}
