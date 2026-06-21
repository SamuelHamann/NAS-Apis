import { listIngredients } from "../api/ingredients";
import { renderResourceList } from "./resourceList";

export async function renderIngredients(): Promise<void> {
  await renderResourceList({
    title: "Ingredients",
    columns: [
      { header: "ID", value: (i) => String(i.id) },
      { header: "Name", value: (i) => i.name },
    ],
    fetcher: () => listIngredients(),
  });
}
