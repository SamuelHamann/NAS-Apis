import { listPantry } from "../api/pantry";
import { renderResourceList } from "./resourceList";

export async function renderPantry(): Promise<void> {
  await renderResourceList({
    title: "Pantry",
    columns: [
      { header: "Ingredient ID", value: (p) => String(p.ingredient_id) },
      { header: "Quantity", value: (p) => String(p.quantity) },
      { header: "Unit ID", value: (p) => String(p.unit_id) },
      { header: "Note", value: (p) => p.note ?? "—" },
      {
        header: "Location",
        value: (p) => (p.location_id != null ? String(p.location_id) : "—"),
      },
    ],
    fetcher: () => listPantry(),
  });
}
