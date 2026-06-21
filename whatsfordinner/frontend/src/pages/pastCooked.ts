import { listPastCooked } from "../api/pastCooked";
import { renderResourceList } from "./resourceList";

export async function renderPastCooked(): Promise<void> {
  await renderResourceList({
    title: "Cooking history",
    columns: [
      { header: "Recipe ID", value: (p) => p.recipe_id },
      { header: "Times cooked", value: (p) => String(p.times_cooked) },
      { header: "Last cooked", value: (p) => p.last_cooked_at },
    ],
    fetcher: () => listPastCooked(),
  });
}
